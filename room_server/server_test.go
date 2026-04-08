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
	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	lobbyserver "github.com/CryptoElementals/common/lobby_server"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

var chainRPC = "http://152.32.231.145:8545"
var svrUrl = "localhost:30011"
var lobbyURL = "localhost:30012"
var roomV2ContractAddress = "0x20ae7393Fe6eC4218E0E27452Cf158FC4c1Ba06C"
var fakeRoomAddress = "0x22767b2ba3cba853af78c9d91c6c520a2b5cb428"

func setupTestSvc(t *testing.T, timeout ...int64) {
	tempFile := path.Join(t.TempDir(), "test_wallet_file")
	require.NoError(t, os.WriteFile(tempFile, []byte("909a42bf20b616a7d317ecc92bde2c88241509745aade0721ff8a917295d31e2"), 0644))
	var gametTimeout int64
	if len(timeout) > 0 {
		gametTimeout = timeout[0]
	}
	if gametTimeout == 0 {
		gametTimeout = 10
	}
	ga := &dao.GameArgs{
		MaxNormalRounds:                       3,
		MaxExtraRounds:                        0,
		MaxTurnsPerNormalRound:                3,
		MaxTurnsPerExtraRound:                 1,
		InitialHP:                             3000,
		BaseStake:                             1000,
		ConfirmationTimeout:                   gametTimeout,
		CommitmentSubmissionTimeout:           gametTimeout,
		CardSubmissionTimeout:                 gametTimeout,
		GameContinueTimeout:                   gametTimeout,
		ConfirmationTimeoutRedundancy:         10,
		CommitmentSubmissionTimeoutRedundancy: 10,
		CardSubmissionTimeoutRedundancy:       10,
		GameContinueTimeoutRedundancy:         10,
	}
	require.NoError(t, db.Get().Create(ga).Error)
	dao.MustValidateGameArgs(ga)
	cfg := &config.RoomServerConfig{
		LogCfg: log.Config{
			Development: true,
		},
		ChainCfg: config.ChainConfig{
			NodeConfig: config.NodeConfig{
				HttpRpc: chainRPC,
			},
			ContractConfig: config.ContractConfig{
				RoomV3ContractAddress: roomV2ContractAddress,
			},
		},
		WalletPaths: []string{tempFile},

		ListenPort:          30011,
		MinTokenToJoinQueue: 1000,
		GameArgsID:          ga.ID,
	}
	config.InitializeGameParams(&config.GameParamConfig{
		SystemFeeRate:   0.016,
		WinnerPointRate: 0.012,
		LoserPointRate:  0.004,
		TieTokenRate:    0.008,
		TiePointRate:    0.008,
		BaseStake:       1000,
		MaxRounds:       3,
		InitialHP:       3000,
	})
	if config.GameParams.BaseStake == 0 {
		config.GameParams.BaseStake = 1000
	}
	redisAddr := os.Getenv("ELEMENTALS_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	require.NoError(t, redis.Init(&redis.Config{Address: redisAddr, Size: 8}))
	svr, err := New(context.Background(), cfg, true)
	if err != nil {
		require.NoError(t, err)
	}
	err = svr.Start()
	if err != nil {
		require.NoError(t, err)
	}
	lcfg := &config.LobbyServerConfig{
		LogCfg:              cfg.LogCfg,
		DbCfg:               cfg.DbCfg,
		ListenPort:          30012,
		RoomServerAddress:   "127.0.0.1:30011",
		MinTokenToJoinQueue: cfg.MinTokenToJoinQueue,
		GameArgsID:          ga.ID,
		BotWaitTime:         0,
		StatServiceEndpoint: "",
		IsDevelop:           true,
	}
	go func() {
		ls, e := lobbyserver.New(context.Background(), lcfg)
		if e != nil {
			panic(e)
		}
		if e := ls.Start(); e != nil {
			panic(e)
		}
	}()
	time.Sleep(3 * time.Second)
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
	// ensure profiles
	_, err := db.GetOrCreateUserProfile("wallet1")
	require.NoError(t, err)
	_, err = db.GetOrCreateUserProfile("wallet2")
	require.NoError(t, err)
	userTokens := []dao.UserToken{
		{PlayerId: 1, TokenAmount: 1000000, Points: 0},
		{PlayerId: 2, TokenAmount: 1000000, Points: 0},
	}
	require.NoError(t, db.SaveUserToken(userTokens...))
}

func toJsonLoggable(obj any) string {
	res, _ := json.MarshalIndent(obj, "", "  ")
	return string(res)
}

func requireRoundGameOverFromDB(t *testing.T, gameID int64, round uint) {
	t.Helper()
	gameInfo, err := db.LoadGameByGameID(gameID)
	require.NoError(t, err)
	synth := conversion.RoundByNumber(gameInfo, uint32(round))
	require.NotNil(t, synth)
	rr := conversion.DbRoundToRoundResult(synth, gameInfo)
	require.True(t, rr.IsGameOver)
	require.NotNil(t, gameInfo.GameResult)
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
	gameIDChan chan int64,
) {
	chanEvt := make(chan *proto.Event)
	chanErr := make(chan error)

	go func() {
		defer fmt.Println("palyer quit", addr)
		defer wg.Done()
		gameID := int64(0)
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
					matched := evt.GetGameMatched()
					if matched != nil && matched.GetMatchId() != 0 {
						require.NoError(t, client.RpcClient.ConfirmMatch(ctx, addr, matched.GetMatchId()))
					}
				case proto.EventType_TYPE_PART_CONFIRMED:
					t.Log("player part confirmed")
					partConfimredChan <- round
				case proto.EventType_TYPE_GAME_CREATED:
					t.Log("game created")
					gr := evt.GetGameReady()
					require.NotNil(t, gr, "TYPE_GAME_CREATED must carry GameReady")
					gameID = gr.GetGameId()
					require.NotZero(t, gameID, "game id must be available after TYPE_GAME_CREATED")
					select {
					case gameIDChan <- gameID:
					default:
					}
					t.Log("game ready: ", toJsonLoggable(gr))
					require.NoError(t, client.RpcClient.ConfirmBattle(ctx, addr, gameID, round, 1))
					// ContractAddress removed - always uses RoomV2 contract address from config
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
								GameId: gameID,
								Tx: &proto.Transaction_CommitmentOnChain{
									CommitmentOnChain: &proto.TxCommitmentOnChain{
										Address:     addr.ToProtoNoWallet(),
										RoundNumber: uint32(round),
										TurnNumber:  1,
										Commitment:  fmt.Appendf(nil, "%s_%s_%d", "card_commitments", addr.String(), round),
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
								GameId: gameID,
								Tx: &proto.Transaction_CardOnChain{
									CardOnChain: &proto.TxCardOnChain{
										Address:     addr.ToProtoNoWallet(),
										RoundNumber: uint32(round),
										TurnNumber:  1,
										Salt:        []byte("salt"),
										CardId:      uint32(submittedCards[0]),
									},
								},
							},
						},
					}
				case proto.EventType_TYPE_TURN_COMPLETE:
					t.Log("turn complete")
					tc := evt.GetTurnCompleted()
					if tc == nil {
						break
					}
					if tc.GetIsGameComplete() {
						t.Log("game complete")
						close(partConfimredChan)
						return
					}
					if tc.GetIsRoundComplete() && !tc.GetIsGameComplete() {
						t.Log("round complete, advancing")
						round++
						require.NoError(t, client.RpcClient.ConfirmBattle(ctx, addr, gameID, round, 1)) // Turn 1 for first round
						t.Logf("confirm submitted, addr: %s, round %d, game: %d", addr.String(), round, gameID)
					}
				}
			case err := <-chanErr:
				require.NoError(t, err)
			}
		}
	}()
	require.NoError(t, client.PubSubClient.Subscribe("test-sub-"+addr.String(), addr.ToProto(), chanEvt, chanErr))
	time.Sleep(1 * time.Second)
}

func TestServer_BattleFullLogic(t *testing.T) {
	setupMemDb(t)
	prepareCards(t)
	prepareUserTokens(t)
	setupTestSvc(t)
	client, err := rpc.NewClient(svrUrl, lobbyURL)
	require.NoError(t, err)
	defer client.Close()
	ctx, ccl := context.WithTimeout(context.Background(), 30*time.Second)
	defer ccl()
	addr1 := &types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "tmp1",
	}
	addr2 := &types.PlayerAddress{
		Id:               2,
		TemporaryAddress: "tmp2",
	}
	chan1 := make(chan uint, 3)
	chan2 := make(chan uint, 3)
	gameIDChan := make(chan int64, 2)
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
			// Use fake tx hash since db.GetCreateRoomTx no longer exists
			txHashBytes := []byte(fmt.Sprintf("game_created_%d", gameID))
			txChan <- &proto.TransactionBatch{
				BlockHash:   []byte("0x123"),
				Timestamp:   uint64(time.Now().Unix()),
				BlockNumber: 1,
				Transactions: []*proto.Transaction{
					{
						TxHash: txHashBytes,
						GameId: gameID,
						Tx: &proto.Transaction_GameCreated{
							GameCreated: &proto.TxGameCreated{},
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
						GameId: gameID,
						Tx: &proto.Transaction_GameTurnSetupReady{
							GameTurnSetupReady: &proto.TxGameTurnSetupReady{
								RoundNumber: uint32(round),
								TurnNumber:  1,
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
	requireRoundGameOverFromDB(t, gameID, round)
	token, err := client.RpcClient.GetPlayerToken(ctx, addr1.Id)
	require.NoError(t, err)
	require.NotNil(t, token)
	// After game end, lobby may put humans in PLAYER_PENDING_QUEUE_MATCH (continue rematch pending ConfirmMatch); JoinQueue must fail until they ConfirmMatch or the rematch times out.
	require.Error(t, client.RpcClient.JoinQueue(ctx, addr1))
}

func TestServer_BattleTimeout(t *testing.T) {
	setupMemDb(t)
	prepareUserTokens(t)
	prepareCards(t)
	setupTestSvc(t, 10)
	client, err := rpc.NewClient(svrUrl, lobbyURL)
	require.NoError(t, err)
	defer client.Close()
	ctx := context.Background()
	addr1 := &types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "tmp1",
	}
	addr2 := &types.PlayerAddress{
		Id:               2,
		TemporaryAddress: "tmp2",
	}

	var gameID int64
	var round uint
	var setupBattle = func(doneChan chan struct{}, doContinue bool) {
		chan1 := make(chan uint, 3)
		chan2 := make(chan uint, 3)
		gameIDChan := make(chan int64, 2)
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
		_ = doContinue
		require.NoError(t, client.RpcClient.JoinQueue(ctx, addr1))
		require.NoError(t, client.RpcClient.JoinQueue(ctx, addr2))

		go func() {
			wg.Wait()
			close(doneChan)
		}()
		gameID = <-gameIDChan
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
					// Use fake tx hash since db.GetCreateRoomTx no longer exists
					txHashBytes := []byte(fmt.Sprintf("game_created_%d", gameID))
					txChan <- &proto.TransactionBatch{
						BlockHash:   []byte("0x123"),
						Timestamp:   uint64(time.Now().Unix()),
						BlockNumber: 1,
						Transactions: []*proto.Transaction{
							{
								TxHash: txHashBytes,
								GameId: gameID,
								Tx: &proto.Transaction_GameCreated{
									GameCreated: &proto.TxGameCreated{},
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
								GameId: gameID,
								Tx: &proto.Transaction_GameTurnSetupReady{
									GameTurnSetupReady: &proto.TxGameTurnSetupReady{
										RoundNumber: uint32(round),
										TurnNumber:  1,
									},
								},
							},
						},
					}
				}
			}
		}()

	}
	var checkBasicResult = func() {
		requireRoundGameOverFromDB(t, gameID, 1)
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
		checkBasicResult()
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

	// test round confirm timeout (signaled via TYPE_TURN_COMPLETE + IsRoundComplete)
	setupTimeout(addr1, proto.EventType_TYPE_TURN_COMPLETE, 12*time.Second)
	runCase()
	clear(timeoutMapByPlayer)

	// Continue rematch uses ConfirmMatch + match confirmation timeout (same as queue PVP); no ContinueGame RPC.
}

func TestServer_BattleContinue(t *testing.T) {
	setupMemDb(t)
	prepareUserTokens(t)
	prepareCards(t)
	setupTestSvc(t)
	client, err := rpc.NewClient(svrUrl, lobbyURL)
	require.NoError(t, err)
	defer client.Close()
	ctx, ccl := context.WithTimeout(context.Background(), 500*time.Second)
	defer ccl()
	addr1 := &types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "tmp1",
	}
	addr2 := &types.PlayerAddress{
		Id:               2,
		TemporaryAddress: "tmp2",
	}

	var gameID int64
	var round uint
	var setupBattle = func(doneChan chan struct{}, doContinue bool) {
		chan1 := make(chan uint, 3)
		chan2 := make(chan uint, 3)
		gameIDChan := make(chan int64, 2)
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
		_ = doContinue
		require.NoError(t, client.RpcClient.JoinQueue(ctx, addr1))
		require.NoError(t, client.RpcClient.JoinQueue(ctx, addr2))

		go func() {
			wg.Wait()
			close(doneChan)
		}()
		gameID = <-gameIDChan
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
				// Use fake tx hash since db.GetCreateRoomTx no longer exists
				txHashBytes := []byte(fmt.Sprintf("game_created_%d", gameID))
				txChan <- &proto.TransactionBatch{
					BlockHash:   []byte("0x123"),
					Timestamp:   uint64(time.Now().Unix()),
					BlockNumber: 1,
					Transactions: []*proto.Transaction{
						{
							TxHash: txHashBytes,
							GameId: gameID,
							Tx: &proto.Transaction_GameCreated{
								GameCreated: &proto.TxGameCreated{},
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
							GameId: gameID,
							Tx: &proto.Transaction_GameTurnSetupReady{
								GameTurnSetupReady: &proto.TxGameTurnSetupReady{
									RoundNumber: uint32(round),
									TurnNumber:  1,
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
	requireRoundGameOverFromDB(t, gameID, round)
	// continue game
	doneChan = make(chan struct{})
	setupBattle(doneChan, true)
	select {
	case <-ctx.Done():
		t.Error("timeout waiting game complete")
	case <-doneChan:
	}
	requireRoundGameOverFromDB(t, gameID, round)
}
