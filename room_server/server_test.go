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
var fakeRoomAddress = "0x22767b2ba3cba853af78c9d91c6c520a2b5cb428"

func setupTestSvc(t *testing.T, timeout ...int64) {
	tempFile := path.Join(t.TempDir(), "test_wallet_file")
	require.NoError(t, os.WriteFile(tempFile, []byte("909a42bf20b616a7d317ecc92bde2c88241509745aade0721ff8a917295d31e2"), 0644))
	var gametTimeout int64
	if len(timeout) > 0 {
		gametTimeout = timeout[0]
	}
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
		WalletPaths:     []string{tempFile},
		RoundTimeout:    gametTimeout,
		ContinueTimeout: gametTimeout,
		MaxRounds:       3,
		GameInitialHP:   3000,
		ListenPort:      30011,
		GameParams: config.GameParamConfig{
			TokenThreshold:    1000,
			MaxHP:             3000,
			InitialMultiplier: 1,
			SystemFeeRate:     0.016,
			WinnerPointRate:   0.012,
			LoserPointRate:    0.004,
			TieTokenRate:      0.008,
			TiePointRate:      0.008,
			BaseStake:         1000,
		},
	}
	config.InitializeGameParams(&cfg.GameParams)
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

func TestInsertUserProfile(t *testing.T) {
	err := db.Init(&db.Config{Endpoint: "10.9.176.247:3306", User: "root", Password: "KYq9gcN82dKWCRTb", DbName: "elementals"})
	require.NoError(t, err)
	userProfile1 := dao.UserProfile{
		Address:       "0x1a8aea9401d35cd6d6a955a386480e7917d0d89b",
		Name:          "bot1",
		AvatarURL:     "avatar_6.png",
		BackgroundURL: "bg_6.png",
	}
	userProfile2 := dao.UserProfile{
		Address:       "0xcd57ac6115d73b401ce53e34b1320e6265261a91",
		Name:          "bot2",
		AvatarURL:     "avatar_5.png",
		BackgroundURL: "bg_5.png",
	}
	err = db.CreateUserProfile(&userProfile1)
	require.NoError(t, err)
	err = db.CreateUserProfile(&userProfile2)
	require.NoError(t, err)
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

func prepareUserTokens(t *testing.T) {
	t.Helper()
	userTokens := []dao.UserToken{
		{WalletAddress: "wallet1", TokenAmount: 1000000, Points: 0},
		{WalletAddress: "wallet2", TokenAmount: 1000000, Points: 0},
	}
	require.NoError(t, db.SaveUserToken(userTokens...))
}

func toJsonLoggable(obj any) string {
	res, _ := json.MarshalIndent(obj, "", "  ")
	return string(res)
}

var timeoutMapByPlayer = map[types.PlayerAddress]map[proto.EventType]time.Duration{}

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
		defer fmt.Println("palyer quit", addr)
		defer wg.Done()
		gameID := uint(0)
		round := uint(1)
		for {
			select {
			case evt, ok := <-chanEvt:
				if !ok {
					break
				}
				timeoutMap, ok := timeoutMapByPlayer[*addr]
				if ok {
					timeout, ok := timeoutMap[evt.Type]
					if ok {
						time.Sleep(timeout)
					}
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
										Salt:                []byte("salt"),
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
						t.Logf("confirm submitted, addr: %s, round %d, game: %d", addr.String(), round, gameID)
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
	time.Sleep(1 * time.Second)
}

func TestServer_BattleFullLogic(t *testing.T) {
	setupMemDb(t)
	prepareCards(t)
	prepareUserTokens(t)
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
	token, err := client.RpcClient.GetPlayerToken(ctx, addr1.WalletAddress)
	require.NoError(t, err)
	require.NotNil(t, token)
	chanEvt1 := make(chan *proto.Event, 2)
	chanErr1 := make(chan error, 2)
	chanEvt2 := make(chan *proto.Event, 2)
	chanErr2 := make(chan error, 2)
	require.NoError(t, client.PubSubClient.Subscribe(addr1.String(), addr1.String(), chanEvt1, chanErr1))
	require.NoError(t, client.PubSubClient.Subscribe(addr2.String(), addr2.String(), chanEvt2, chanErr2))
	time.Sleep(1 * time.Second)
	client.RpcClient.RefuseContinueGame(ctx, addr1, gameID)
	for i := 0; i < 2; i++ {
		select {
		case evt := <-chanEvt1:
			require.NotNil(t, evt)
			require.Equal(t, proto.EventType_TYPE_CONTINUE_CANCELED, evt.Type)
		case evt := <-chanEvt2:
			require.NotNil(t, evt)
			require.Equal(t, proto.EventType_TYPE_CONTINUE_CANCELED, evt.Type)
		case err := <-chanErr1:
			require.NoError(t, err)
		case err := <-chanErr2:
			require.NoError(t, err)
		}
	}

}

func TestServer_BattleTimeout(t *testing.T) {
	setupMemDb(t)
	prepareUserTokens(t)
	prepareCards(t)
	setupTestSvc(t, 10)
	client, err := rpc.NewClient(svrUrl)
	require.NoError(t, err)
	defer client.Close()
	ctx := context.Background()
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
		go func() {
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
				time.Sleep(5 * time.Second)
				select {
				case <-ctx.Done():
					t.Errorf("timeout waiting game complete, round: %d", round)
					return
				case <-doneChan:
					return
				default:
				}
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
		}()

	}
	var checkBasicResult = func(battleInfo *proto.GetBattleInfoResponse, err error) {
		require.NoError(t, err)
		require.NotNil(t, battleInfo)
		require.NotNil(t, battleInfo.RoundResult)
		require.NotNil(t, battleInfo.GameResult)
		require.True(t, battleInfo.RoundResult.IsGameOver)
	}
	var runCase = func() {
		doneChan := make(chan struct{})
		setupBattle(doneChan, false)
		select {
		case <-ctx.Done():
			t.Error("timeout waiting game complete")
		case <-doneChan:
		}
		// should stop at round one
		battleInfo, err := client.RpcClient.GetBattleInfo(ctx, gameID, 1)
		checkBasicResult(battleInfo, err)

	}
	var setupTimeout = func(addr *types.PlayerAddress, evtType proto.EventType, interval time.Duration) {
		if timeoutMapByPlayer[*addr] == nil {
			timeoutMapByPlayer[*addr] = make(map[proto.EventType]time.Duration)
		}
		timeoutMap := timeoutMapByPlayer[*addr]
		timeoutMap[evtType] = interval
	}
	// // test confirm timeout
	// setupTimeout(addr1, proto.EventType_TYPE_MATCHED, 12*time.Second)
	// runCase()
	// clear(timeoutMapByPlayer)

	// test submit commitments timeout
	setupTimeout(addr1, proto.EventType_TYPE_GAME_CREATED, 12*time.Second)
	runCase()
	clear(timeoutMapByPlayer)

	// test submit cards timeout
	setupTimeout(addr1, proto.EventType_TYPE_COMMITMENTS_ON_CHAIN, 12*time.Second)
	runCase()
	clear(timeoutMapByPlayer)

	//test round confirm timeout
	setupTimeout(addr1, proto.EventType_TYPE_ROUND_COMPLETE, 12*time.Second)
	runCase()
	clear(timeoutMapByPlayer)

	// test continue confirm timeout
	runCase()
	time.Sleep(12 * time.Second)
	err = client.RpcClient.ContinueGame(ctx, addr1, gameID)
	require.NotNil(t, err)
}

func TestServer_BattleContinue(t *testing.T) {
	setupMemDb(t)
	prepareUserTokens(t)
	prepareCards(t)
	setupTestSvc(t)
	client, err := rpc.NewClient(svrUrl)
	require.NoError(t, err)
	defer client.Close()
	ctx, ccl := context.WithTimeout(context.Background(), 500*time.Second)
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
			time.Sleep(5 * time.Second)
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
