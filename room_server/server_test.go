package roomserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

var chainRPC = "http://152.32.231.145:8545"
var svrUrl = "localhost:30011"
var roomManagerAddress = "0x20ae7393Fe6eC4218E0E27452Cf158FC4c1Ba06C"
var fakeRoomAddress = "0x22767b2bA3Cba853AF78c9D91c6c520a2B5Cb428"

func setupTestSvc(t *testing.T) {
	tempFile := path.Join(t.TempDir(), "test_wallet_file")
	require.NoError(t, os.WriteFile(tempFile, []byte("909a42bf20b616a7d317ecc92bde2c88241509745aade0721ff8a917295d31e2"), 0644))
	cfg := &config.RoomServerConfig{
		LogCfg: log.Config{
			Development: true,
		},
		ChainCfg: config.ChainConfig{
			NodeConfig: config.NodeConfig{
				HttpRpc: chainRPC,
			},
			ContractConfig: config.ContractConfig{
				RoomManagerAddress: roomManagerAddress,
			},
		},
		WalletPath:    tempFile,
		RoundTimeout:  0,
		MaxRounds:     3,
		GameInitialHP: 10000,
		ListenPort:    30011,
	}
	svr, err := New(context.Background(), cfg, true)
	if err != nil {
		require.NoError(t, err)
	}
	err = svr.Start()
	if err != nil {
		require.NoError(t, err)
	}
}

func TestMain(m *testing.M) {

	os.Exit(m.Run())
}

func setupMemDb(t *testing.T) {
	err := db.Init(&db.Config{Development: true})
	require.NoError(t, err)
	err = db.MigrateMemDb()
	require.NoError(t, err)
}

func TestInsertCards(t *testing.T) {
	err := db.Init(&db.Config{Endpoint: "10.9.176.247:3306", User: "root", Password: "KYq9gcN82dKWCRTb", DbName: "elementals"})
	require.NoError(t, err)
	prepareCards(t)
}

func prepareCards(t *testing.T) {
	t.Helper()
	cards := []dao.Card{
		{CardID: 1, ElementType: "Metal", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Kylin", Description: "Kylin clad in armor, representing strength and protection"},
		{CardID: 2, ElementType: "Wood", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Forest Spirit", Description: "Forest Spirit controlling the cycle of life and death"},
		{CardID: 3, ElementType: "Water", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Siren", Description: "Siren, half-human half-beast, possessing enchanting charm"},
		{CardID: 4, ElementType: "Fire", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Phoenix", Description: "Phoenix with flames and rebirth, symbolizing eternal life"},
		{CardID: 5, ElementType: "Earth", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "World Turtle", Description: "World Turtle, steady and powerful with immense strength"},
	}
	require.NoError(t, db.Get().Save(&cards).Error)
}

func toJsonLoggable(obj any) string {
	res, _ := json.MarshalIndent(obj, "", "  ")
	return string(res)
}

func runClient(t *testing.T,
	ctx context.Context,
	wg *sync.WaitGroup,
	client *rpc.Client,
	addr *types.PlayerAddress,
	submittedCards []uint32,
	txChan chan *proto.TransactionBatch,
	partConfimredChan chan uint,
	gameIDChan chan uint,
) {
	chanEvt := make(chan *proto.Event)
	chanErr := make(chan error)

	go func() {
		defer wg.Done()
		gameID := uint(0)
		round := uint(1)
		for {
			select {
			case evt, ok := <-chanEvt:
				if !ok {
					break
				}
				switch evt.Type {
				case proto.EventType_TYPE_KNOWN:
					t.Errorf("unhandled event type from: %s", addr)
					return
				case proto.EventType_TYPE_MATCHED:
					t.Log("player matched")
					phase, err := client.RpcClient.GetGamePhase(ctx, addr)
					require.NoError(t, err)
					gameID = uint(phase.PvPInfo.GameID)
					gameIDChan <- gameID
					require.NoError(t, client.RpcClient.ConfirmBattle(ctx, addr, gameID, round))
				case proto.EventType_TYPE_PART_CONFIRMED:
					t.Log("player part confirmed")
					partConfimredChan <- round
				case proto.EventType_TYPE_GAME_CREATED:
					t.Log("game created")
					// get contract
					phase, err := client.RpcClient.GetGamePhase(ctx, addr)
					require.NoError(t, err)
					gameID = uint(phase.PvPInfo.GameID)
					t.Log("game phase: ", toJsonLoggable(phase))
					require.Equal(t, fakeRoomAddress, phase.PvPInfo.ContractAddress)
				case proto.EventType_TYPE_ROUND_READY:
					t.Log("round ready")
					// submit commitments
					txChan <- &proto.TransactionBatch{
						BlockHash:   []byte("0x123"),
						Timestamp:   uint64(time.Now().Unix()),
						BlockNumber: 1,
						Transactions: []*proto.Transaction{
							{
								TxHash: []byte(fmt.Sprintf("%s_%s_%d", "submit_trasactions", addr.String(), round)),
								Tx: &proto.Transaction_CommitmentsOnChain{
									CommitmentsOnChain: &proto.TxCommitmentsOnChain{
										RoomContractAddress: fakeRoomAddress,
										Address:             addr.ToProtoNoWallet(),
										RoundNumber:         uint32(round),
										Commitment:          fmt.Appendf(nil, "%s_%s_%d", "card_commitments", addr.String(), round),
									},
								},
							},
						},
					}
				case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
					t.Log("commitments on chain")
					// submit cards
					txChan <- &proto.TransactionBatch{
						BlockHash:   []byte("0x123"),
						Timestamp:   uint64(time.Now().Unix()),
						BlockNumber: 1,
						Transactions: []*proto.Transaction{
							{
								TxHash: []byte(fmt.Sprintf("%s_%s_%d", "submit_cards", addr.String(), round)),
								Tx: &proto.Transaction_CardsOnChain{
									CardsOnChain: &proto.TxCardsOnChain{
										RoomContractAddress: fakeRoomAddress,
										Address:             addr.ToProtoNoWallet(),
										RoundNumber:         uint32(round),
										Cards:               submittedCards,
									},
								},
							},
						},
					}
				case proto.EventType_TYPE_CARDS_ON_CHAIN:
					t.Log("cards on chain")
					// skip
				case proto.EventType_TYPE_ROUND_COMPLETE:
					t.Log("round complete")
					battleInfo, err := client.RpcClient.GetBattleInfo(ctx, gameID, round)
					require.NoError(t, err)
					t.Log("battle info: ", toJsonLoggable(battleInfo))
					if !battleInfo.RoundResult.IsGameOver {
						round++
						require.NoError(t, client.RpcClient.ConfirmBattle(ctx, addr, gameID, round))
					}
				case proto.EventType_TYPE_GAME_COMPLETE:
					t.Log("game complete")
					close(partConfimredChan)
					return
				}
			case err := <-chanErr:
				require.NoError(t, err)
			}
		}
	}()
	require.NoError(t, client.PubSubClient.Subscribe(addr.String(), addr.String(), chanEvt, chanErr))
}

func TestServer_BattleFullLogic(t *testing.T) {
	setupMemDb(t)
	prepareCards(t)
	setupTestSvc(t)
	client, err := rpc.NewClient(svrUrl)
	require.NoError(t, err)
	defer client.Close()
	ctx, ccl := context.WithTimeout(context.Background(), 30*time.Second)
	defer ccl()
	addr1 := &types.PlayerAddress{
		WalletAddress:    "wallet1",
		TemporaryAddress: "tmp1",
	}
	addr2 := &types.PlayerAddress{
		WalletAddress:    "wallet2",
		TemporaryAddress: "tmp2",
	}
	chan1 := make(chan uint, 3)
	chan2 := make(chan uint, 3)
	gameIDChan := make(chan uint, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)
	doneChan := make(chan struct{})
	txChan := make(chan *proto.TransactionBatch, 10)
	go func() {
		for tx := range txChan {
			err := client.RpcClient.SubmitTransactions(ctx, tx)
			require.NoError(t, err)
		}
	}()
	runClient(t, ctx, &wg, client, addr1, []uint32{4, 5, 3}, txChan, chan1, gameIDChan)
	runClient(t, ctx, &wg, client, addr2, []uint32{1, 2, 4}, txChan, chan2, gameIDChan)
	require.NoError(t, client.RpcClient.JoinQueue(ctx, addr1))
	require.NoError(t, client.RpcClient.JoinQueue(ctx, addr2))
	go func() {
		wg.Wait()
		close(doneChan)
	}()
	round := uint(0)
	gameID := <-gameIDChan
	var txHashBytes []byte
	for {
		var round1 uint
		var round2 uint
		select {
		case <-ctx.Done():
			t.Errorf("timeout waiting game complete, round: %d", round1)
			return
		case r1 := <-chan1:
			round1 = r1
		}
		select {
		case <-ctx.Done():
			t.Errorf("timeout waiting game complete, round: %d", round2)
			return
		case r2 := <-chan2:
			round2 = r2
		}
		require.Equal(t, round1, round2)
		require.Less(t, round1, uint(4))
		if round1 == 0 {
			// chan closed, game completed
			break
		}
		round = round1
		time.Sleep(3 * time.Second)
		if round == 1 {
			tx, err := db.GetCreateRoomTx(gameID)
			require.NoError(t, err)
			txHash := tx.TxHash
			txHashBytes, err = hexutil.Decode(txHash)
			require.NoError(t, err)
			txChan <- &proto.TransactionBatch{
				BlockHash:   []byte("0x123"),
				Timestamp:   uint64(time.Now().Unix()),
				BlockNumber: 1,
				Transactions: []*proto.Transaction{
					{
						TxHash: txHashBytes,
						Tx: &proto.Transaction_RoomContractCreated{
							RoomContractCreated: &proto.TxRoomContractCreated{
								RoomContractAddress: fakeRoomAddress,
							},
						},
					},
				},
			}
		} else {
			txChan <- &proto.TransactionBatch{
				BlockHash:   []byte("0x123"),
				Timestamp:   uint64(time.Now().Unix()),
				BlockNumber: 1,
				Transactions: []*proto.Transaction{
					{
						TxHash: fmt.Appendf(nil, "%s_%d", "round_ready", round),
						Tx: &proto.Transaction_RoomContractSetupReady{
							RoomContractSetupReady: &proto.TxRoomContractRoundSetupReady{
								RoomContractAddress: fakeRoomAddress,
								RoundNumber:         uint32(round),
							},
						},
					},
				},
			}
		}
	}
	select {
	case <-ctx.Done():
		t.Error("timeout waiting game complete")
	case <-doneChan:
	}
	battleInfo, err := client.RpcClient.GetBattleInfo(ctx, gameID, round)
	require.NoError(t, err)
	require.NotNil(t, battleInfo)
	require.NotNil(t, battleInfo.RoundResult)
	require.NotNil(t, battleInfo.GameResult)
	require.True(t, battleInfo.RoundResult.IsGameOver)
}

func TestServer_BattleTimeout(t *testing.T) {
	setupMemDb(t)
	prepareCards(t)
	setupTestSvc(t)
	client, err := rpc.NewClient(svrUrl)
	require.NoError(t, err)
	defer client.Close()
	ctx, ccl := context.WithTimeout(context.Background(), 30*time.Second)
	defer ccl()
	addr1 := &types.PlayerAddress{
		WalletAddress:    "wallet1",
		TemporaryAddress: "tmp1",
	}
	addr2 := &types.PlayerAddress{
		WalletAddress:    "wallet2",
		TemporaryAddress: "tmp2",
	}
	chan1 := make(chan uint, 3)
	chan2 := make(chan uint, 3)
	gameIDChan := make(chan uint, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)
	doneChan := make(chan struct{})
	txChan := make(chan *proto.TransactionBatch, 10)
	go func() {
		for tx := range txChan {
			err := client.RpcClient.SubmitTransactions(ctx, tx)
			require.NoError(t, err)
		}
	}()
	runClient(t, ctx, &wg, client, addr1, []uint32{4, 5, 3}, txChan, chan1, gameIDChan)
	runClient(t, ctx, &wg, client, addr2, []uint32{1, 2, 4}, txChan, chan2, gameIDChan)
	require.NoError(t, client.RpcClient.JoinQueue(ctx, addr1))
	require.NoError(t, client.RpcClient.JoinQueue(ctx, addr2))
	go func() {
		wg.Wait()
		close(doneChan)
	}()
	round := uint(0)
	gameID := <-gameIDChan
	var txHashBytes []byte
	for {
		var round1 uint
		var round2 uint
		select {
		case <-ctx.Done():
			t.Errorf("timeout waiting game complete, round: %d", round1)
			return
		case r1 := <-chan1:
			round1 = r1
		}
		select {
		case <-ctx.Done():
			t.Errorf("timeout waiting game complete, round: %d", round2)
			return
		case r2 := <-chan2:
			round2 = r2
		}
		require.Equal(t, round1, round2)
		require.Less(t, round1, uint(4))
		if round1 == 0 {
			// chan closed, game completed
			break
		}
		round = round1
		time.Sleep(3 * time.Second)
		if round == 1 {
			tx, err := db.GetCreateRoomTx(gameID)
			require.NoError(t, err)
			txHash := tx.TxHash
			txHashBytes, err = hexutil.Decode(txHash)
			require.NoError(t, err)
			txChan <- &proto.TransactionBatch{
				BlockHash:   []byte("0x123"),
				Timestamp:   uint64(time.Now().Unix()),
				BlockNumber: 1,
				Transactions: []*proto.Transaction{
					{
						TxHash: txHashBytes,
						Tx: &proto.Transaction_RoomContractCreated{
							RoomContractCreated: &proto.TxRoomContractCreated{
								RoomContractAddress: fakeRoomAddress,
							},
						},
					},
				},
			}
		} else {
			txChan <- &proto.TransactionBatch{
				BlockHash:   []byte("0x123"),
				Timestamp:   uint64(time.Now().Unix()),
				BlockNumber: 1,
				Transactions: []*proto.Transaction{
					{
						TxHash: fmt.Appendf(nil, "%s_%d", "round_ready", round),
						Tx: &proto.Transaction_RoomContractSetupReady{
							RoomContractSetupReady: &proto.TxRoomContractRoundSetupReady{
								RoomContractAddress: fakeRoomAddress,
								RoundNumber:         uint32(round),
							},
						},
					},
				},
			}
		}
	}
	select {
	case <-ctx.Done():
		t.Error("timeout waiting game complete")
	case <-doneChan:
	}
	battleInfo, err := client.RpcClient.GetBattleInfo(ctx, gameID, round)
	require.NoError(t, err)
	require.NotNil(t, battleInfo)
	require.NotNil(t, battleInfo.RoundResult)
	require.NotNil(t, battleInfo.GameResult)
	require.True(t, battleInfo.RoundResult.IsGameOver)
}

func TestServer_BattleContinue(t *testing.T) {
	setupMemDb(t)
	prepareCards(t)
	setupTestSvc(t)
	client, err := rpc.NewClient(svrUrl)
	require.NoError(t, err)
	defer client.Close()
	ctx, ccl := context.WithTimeout(context.Background(), 300*time.Second)
	defer ccl()
	addr1 := &types.PlayerAddress{
		WalletAddress:    "wallet1",
		TemporaryAddress: "tmp1",
	}
	addr2 := &types.PlayerAddress{
		WalletAddress:    "wallet2",
		TemporaryAddress: "tmp2",
	}

	var gameID uint
	var round uint
	var setupBattle = func(doneChan chan struct{}, doContinue bool) {
		chan1 := make(chan uint, 3)
		chan2 := make(chan uint, 3)
		gameIDChan := make(chan uint, 2)
		wg := sync.WaitGroup{}
		wg.Add(2)

		txChan := make(chan *proto.TransactionBatch, 10)
		go func() {
			for tx := range txChan {
				err := client.RpcClient.SubmitTransactions(ctx, tx)
				require.NoError(t, err)
			}
		}()
		runClient(t, ctx, &wg, client, addr1, []uint32{4, 5, 3}, txChan, chan1, gameIDChan)
		runClient(t, ctx, &wg, client, addr2, []uint32{1, 2, 4}, txChan, chan2, gameIDChan)
		if doContinue {
			require.NoError(t, client.RpcClient.ContinueGame(ctx, addr1, gameID))
			require.NoError(t, client.RpcClient.ContinueGame(ctx, addr2, gameID))
			// input expected game id manually
			gameIDChan <- 2
			chan1 <- 1
			chan2 <- 1
		} else {
			require.NoError(t, client.RpcClient.JoinQueue(ctx, addr1))
			require.NoError(t, client.RpcClient.JoinQueue(ctx, addr2))
		}

		go func() {
			wg.Wait()
			close(doneChan)
		}()
		gameID = <-gameIDChan
		var txHashBytes []byte
		for {
			var round1 uint
			var round2 uint
			select {
			case <-ctx.Done():
				t.Errorf("timeout waiting game complete, round: %d", round1)
				return
			case r1 := <-chan1:
				round1 = r1
			}
			select {
			case <-ctx.Done():
				t.Errorf("timeout waiting game complete, round: %d", round2)
				return
			case r2 := <-chan2:
				round2 = r2
			}
			require.Equal(t, round1, round2)
			require.Less(t, round1, uint(4))
			if round1 == 0 {
				// chan closed, game completed
				break
			}
			round = round1
			time.Sleep(3 * time.Second)
			if round == 1 {
				tx, err := db.GetCreateRoomTx(gameID)
				require.NoError(t, err)
				txHash := tx.TxHash
				txHashBytes, err = hexutil.Decode(txHash)
				require.NoError(t, err)
				txChan <- &proto.TransactionBatch{
					BlockHash:   []byte("0x123"),
					Timestamp:   uint64(time.Now().Unix()),
					BlockNumber: 1,
					Transactions: []*proto.Transaction{
						{
							TxHash: txHashBytes,
							Tx: &proto.Transaction_RoomContractCreated{
								RoomContractCreated: &proto.TxRoomContractCreated{
									RoomContractAddress: fakeRoomAddress,
								},
							},
						},
					},
				}
			} else {
				txChan <- &proto.TransactionBatch{
					BlockHash:   []byte("0x123"),
					Timestamp:   uint64(time.Now().Unix()),
					BlockNumber: 1,
					Transactions: []*proto.Transaction{
						{
							TxHash: fmt.Appendf(nil, "%s_%d", "round_ready", round),
							Tx: &proto.Transaction_RoomContractSetupReady{
								RoomContractSetupReady: &proto.TxRoomContractRoundSetupReady{
									RoomContractAddress: fakeRoomAddress,
									RoundNumber:         uint32(round),
								},
							},
						},
					},
				}
			}
		}
	}
	doneChan := make(chan struct{})
	setupBattle(doneChan, false)
	select {
	case <-ctx.Done():
		t.Error("timeout waiting game complete")
	case <-doneChan:
	}
	battleInfo, err := client.RpcClient.GetBattleInfo(ctx, gameID, round)
	require.NoError(t, err)
	require.NotNil(t, battleInfo)
	require.NotNil(t, battleInfo.RoundResult)
	require.NotNil(t, battleInfo.GameResult)
	require.True(t, battleInfo.RoundResult.IsGameOver)
	// continue game
	doneChan = make(chan struct{})
	setupBattle(doneChan, true)
	select {
	case <-ctx.Done():
		t.Error("timeout waiting game complete")
	case <-doneChan:
	}
	battleInfo, err = client.RpcClient.GetBattleInfo(ctx, gameID, round)
	require.NoError(t, err)
	require.NotNil(t, battleInfo)
	require.NotNil(t, battleInfo.RoundResult)
	require.NotNil(t, battleInfo.GameResult)
	require.True(t, battleInfo.RoundResult.IsGameOver)
}
