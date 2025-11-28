package chain

import (
	"context"
	"os"
	"testing"
	"time"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/room_server/worker"
	tt "github.com/CryptoElementals/common/room_server/worker/testing"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var testWorkerManager *worker.WorkerManager
var client bind.ContractBackend
var w *wallet.Wallet

var chainID uint64

const roomMamangerAddress = "0x20ae7393Fe6eC4218E0E27452Cf158FC4c1Ba06C"
const roomContractAddress = "0x8c21e3B3A6Cc3739f418535FDE4Bf76F4DfF8535"

func TestMain(m *testing.M) {
	time.Local = time.UTC
	testWorkerManager = worker.NewWorkerManager(context.Background())
	err := db.Init(&db.Config{
		Development: true,
	})
	if err != nil {
		panic(err)
	}
	err = db.MigrateMemDb()
	if err != nil {
		panic(err)
	}
	// load ecdsa private key from hex
	priv, err := crypto.HexToECDSA("909a42bf20b616a7d317ecc92bde2c88241509745aade0721ff8a917295d31e2")
	if err != nil {
		panic(err)
	}
	w = wallet.NewWalletFromPrivKey(priv)
	ethC, err := ethclient.Dial("http://152.32.231.145:8545")
	if err != nil {
		panic(err)
	}
	id, err := ethC.ChainID(context.Background())
	if err != nil {
		panic(err)
	}
	chainID = id.Uint64()
	client = ethC
	os.Exit(m.Run())
}

func TestFilterEvent(t *testing.T) {
	ec := client.(*ethclient.Client)
	for {
		receipt, err := ec.TransactionReceipt(context.Background(), common.HexToHash("0x3a9738a38f35ce59fea8d56ef6ca74d81e46e57b99bdf5a8575f92975ce25e98"))
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		require.Equal(t, uint64(1), receipt.Status)
		ctrt, err := contract.NewRoomManagerContract(common.HexToAddress(roomMamangerAddress), client)
		require.NoError(t, err)
		parsed, err := ctrt.ParseRoomCreated(*receipt.Logs[0])
		require.NoError(t, err)
		addr := parsed.RoomAddress.String()
		blockNum := receipt.BlockNumber.Uint64()
		t.Log(blockNum)
		t.Log(addr)
		break
	}
}

func TestFilterRoomEvent(t *testing.T) {
	ctrt, err := contract.NewRoomContract(common.HexToAddress("0xCaa747a5B46427ED0b4203637e5F6F7FDbA4c541"), client)
	require.NoError(t, err)
	hh, err := ctrt.CurrentRound(nil)
	require.NoError(t, err)
	num := hh.Uint64()
	t.Log(num)
}

func TestChainContractInteraction(t *testing.T) {
	roomWorkerID := "123"
	gameID := 123
	player1 := types.PlayerAddress{
		Id:               123,
		TemporaryAddress: "0x456",
	}
	player2 := types.PlayerAddress{
		Id:               789,
		TemporaryAddress: "0xabc",
	}

	svc, _ := NewService(context.Background(), testWorkerManager, int64(chainID), client, "", []*wallet.Wallet{w}, true)

	svc.Start()
	mockRoomHandler := tt.NewMockEventHandler(gomock.NewController(t))
	ackReceived := make(chan struct{})
	testWorkerManager.SpwanWorker(context.Background(), roomWorkerID, types.WORKER_TYPE_GAME, mockRoomHandler)

	testWorkerManager.SendEvent(types.CHAIN_MANAGER_ID, types.NewEvent(roomWorkerID, &types.RequireGameCreationEvent{
		GameID:         uint(gameID),
		Players:        []types.PlayerAddress{player1, player2},
		RoundTimeout:   10,
		MaxRoundNumber: 3,
		InitialHP:      1000,
	}, true))
	<-ackReceived
	// Transaction tables removed - no longer checking database
	ackReceived = make(chan struct{})
	testWorkerManager.SendEvent(types.CHAIN_MANAGER_ID, types.NewEvent(roomWorkerID, &types.RequireSetupNewRoundEvent{
		GameID:      uint(gameID),
		RoundNumber: 2,
		// ContractAddress removed - always uses RoomV2 contract address
	}, true))
	<-ackReceived

	evtMatcher := tt.NewEventTypeMatcher(
		&types.RoomCreated{},
		&types.NewTurnSetupComplete{},
		&types.PlayerCommitmentOnChain{},
		&types.PlayerCardOnChain{},
	)
	// Use a test tx hash since we're no longer getting it from database
	txHash := []byte("test_tx_hash")
	mockRoomHandler.EXPECT().Handle(gomock.Any(), evtMatcher).Times(5).Return(nil)
	err := svc.SubmitTransactions(&proto.TransactionBatch{
		BlockHash:   []byte("0x123"),
		Timestamp:   uint64(time.Now().Unix()),
		BlockNumber: 1,
		Transactions: []*proto.Transaction{
			{
				TxHash: txHash,
				Tx: &proto.Transaction_GameCreated{
					GameCreated: &proto.TxGameCreated{
						GameId: int64(gameID),
					},
				},
			},
			{
				TxHash: []byte("GameTurnSetupReady"),
				Tx: &proto.Transaction_GameTurnSetupReady{
					GameTurnSetupReady: &proto.TxGameTurnSetupReady{
						GameId:      int64(gameID),
						RoundNumber: 2,
						TurnNumber:  1,
					},
				},
			},
			{
				TxHash: []byte("Transaction_CommitmentOnChain"),
				Tx: &proto.Transaction_CommitmentOnChain{
					CommitmentOnChain: &proto.TxCommitmentOnChain{
						GameId:      int64(gameID),
						Address:     player1.ToProto(),
						RoundNumber: 2,
						TurnNumber:  1,
						Commitment:  []byte("0xcommitment"),
					},
				},
			},
			{
				TxHash: []byte("Transaction_CardOnChain"),
				Tx: &proto.Transaction_CardOnChain{
					CardOnChain: &proto.TxCardOnChain{
						GameId:      int64(gameID),
						Address:     player1.ToProto(),
						RoundNumber: 2,
						TurnNumber:  1,
						Salt:        []byte("0x123"),
						CardId:      1,
					},
				},
			},
			{
				TxHash: []byte("Transaction_CardOnChain"),
				Tx: &proto.Transaction_CardOnChain{
					CardOnChain: &proto.TxCardOnChain{
						GameId:      int64(gameID),
						Address:     player2.ToProto(),
						RoundNumber: 2,
						TurnNumber:  1,
						Salt:        []byte("0x123"),
						CardId:      4,
					},
				},
			},
		},
	})
	require.NoError(t, err)
	// Transaction tables removed - no longer checking database records
}
