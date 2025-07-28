package chain

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CryptoElementals/common/cache"
	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/room_server/worker"
	tt "github.com/CryptoElementals/common/room_server/worker/testing"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
		receipt, err := ec.TransactionReceipt(context.Background(), common.HexToHash("0xf04a4c037814673c0b182fdf4c6441209c423b8c0cf8bcf6b1e2efb78fe3b97d"))
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
		WalletAddress:    "0x123",
		TemporaryAddress: "0x456",
	}
	player2 := types.PlayerAddress{
		WalletAddress:    "0x789",
		TemporaryAddress: "0xabc",
	}

	svc := NewService(context.Background(), testWorkerManager, int64(chainID), client, roomMamangerAddress, w, cache.NewMemCache(), true)

	svc.Start()
	mockRoomHandler := tt.NewMockEventHandler(gomock.NewController(t))
	ackReceived := make(chan struct{})
	mockRoomHandler.EXPECT().Handle(gomock.Any(), tt.NewEventTypeMatcher(&types.AckEvent{})).AnyTimes().DoAndReturn(func(ctx context.Context, event *types.Event) error {
		close(ackReceived)
		return nil
	})
	mockRoomHandler.EXPECT().Handle(gomock.Any(), tt.NewEventTypeMatcher(&types.ErrorEvent{})).AnyTimes().DoAndReturn(func(ctx context.Context, event *types.Event) error {
		evt := event.Data.(*types.ErrorEvent)
		t.Errorf("ErrorEvent should not be sent, err: %v", evt.Err)
		close(ackReceived)
		return nil
	})
	testWorkerManager.SpwanWorker(context.Background(), roomWorkerID, types.WORKER_TYPE_GAME, mockRoomHandler)

	testWorkerManager.SendEvent(types.CHAIN_MANAGER_ID, types.NewEvent(roomWorkerID, &types.RequireContractCreationEvent{
		GameID:         uint(gameID),
		Players:        []types.PlayerAddress{player1, player2},
		RoundTimeout:   10,
		MaxRoundNumber: 3,
		InitialHP:      1000,
	}, true))
	<-ackReceived
	tx, err := db.GetCreateRoomTx(uint(gameID))
	require.NoError(t, err)
	require.NotEmpty(t, tx)
	ackReceived = make(chan struct{})
	testWorkerManager.SendEvent(types.CHAIN_MANAGER_ID, types.NewEvent(roomWorkerID, &types.RequireSetupNewRoundEvent{
		GameID:          uint(gameID),
		RoundNumber:     2,
		ContractAddress: roomContractAddress,
	}, true))
	<-ackReceived

	evtMatcher := tt.NewEventTypeMatcher(
		&types.RoomContractCreated{},
		&types.NewRoundSetupComplete{},
		&types.PlayerCommitmentOnChain{},
		&types.PlayerCardsOnChain{},
	)
	txHash, err := hexutil.Decode(tx.TxHash)
	require.NoError(t, err)
	mockRoomHandler.EXPECT().Handle(gomock.Any(), evtMatcher).Times(5).Return(nil)
	err = svc.SubmitTransactions(&proto.TransactionBatch{
		BlockHash:   []byte("0x123"),
		Timestamp:   uint64(time.Now().Unix()),
		BlockNumber: 1,
		Transactions: []*proto.Transaction{
			{
				TxHash: txHash,
				Tx: &proto.Transaction_RoomContractCreated{
					RoomContractCreated: &proto.TxRoomContractCreated{
						RoomContractAddress: roomContractAddress,
					},
				},
			},
			{
				TxHash: []byte("RoomContractSetup"),
				Tx: &proto.Transaction_RoomContractSetupReady{
					RoomContractSetupReady: &proto.TxRoomContractRoundSetupReady{
						RoomContractAddress: roomContractAddress,
						RoundNumber:         2,
					},
				},
			},
			{
				TxHash: []byte("Transaction_CommitmentsOnChain"),
				Tx: &proto.Transaction_CommitmentsOnChain{
					CommitmentsOnChain: &proto.TxCommitmentsOnChain{
						RoomContractAddress: roomContractAddress,
						Address:             player1.ToProto(),
						RoundNumber:         2,
						Commitment:          []byte("0xcommitment"),
					},
				},
			},
			{
				TxHash: []byte("Transaction_CardsOnChain"),
				Tx: &proto.Transaction_CardsOnChain{
					CardsOnChain: &proto.TxCardsOnChain{
						RoomContractAddress: roomContractAddress,
						Address:             player1.ToProto(),
						RoundNumber:         2,
						Salt:                []byte("0x123"),
						Cards:               []uint32{1, 2, 3},
					},
				},
			},
			{
				TxHash: []byte("Transaction_CardsOnChain"),
				Tx: &proto.Transaction_CardsOnChain{
					CardsOnChain: &proto.TxCardsOnChain{
						RoomContractAddress: roomContractAddress,
						Address:             player2.ToProto(),
						RoundNumber:         2,
						Salt:                []byte("0x123"),
						Cards:               []uint32{4, 5, 6},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	{
		tx, err = db.GetCreateRoomTx(uint(gameID))
		require.NoError(t, err)
		require.NotEmpty(t, tx)
	}
	{
		txs, err := db.GetCommitmentOnChainTx(uint(gameID), 2)
		require.NoError(t, err)
		require.NotEmpty(t, txs)
	}

	{
		txs, err := db.GetCardsOnChainTx(uint(gameID), 2)
		require.NoError(t, err)
		require.NotEmpty(t, txs)
	}
	{
		tx, err := db.GetSetRoundReadyTx(uint(gameID), 2)
		require.NoError(t, err)
		require.NotEmpty(t, tx)
	}

}
