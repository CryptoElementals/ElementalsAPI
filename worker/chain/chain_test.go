package chain

import (
	"context"
	"os"
	"testing"
	"time"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/CryptoElementals/common/worker"
	tt "github.com/CryptoElementals/common/worker/testing"
	"github.com/CryptoElementals/common/worker/types"
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

const roomMamangerAddress = "0x9e3F48723D94940F7E1c8dCc6c927A55Df896439"
const roomContractAddress = "0xeba41b209e3d8aab61f5072676854979051957f2"

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
	client, err = ethclient.Dial("http://117.50.80.239:8545")
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestFilterEvent(t *testing.T) {
	ec := client.(*ethclient.Client)
	for {
		receipt, err := ec.TransactionReceipt(context.Background(), common.HexToHash("0x8029cab6955f5cc97e89602afe2e9f64c07730d5833d61dc1e2679cb3be1d849"))
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		require.Equal(t, uint64(1), receipt.Status)
		ctrt, err := contract.NewRoomManagerContract(common.HexToAddress(roomMamangerAddress), client)
		require.NoError(t, err)
		parsed, err := ctrt.ParseRoomCreated(*receipt.Logs[0])
		require.NoError(t, err)
		addr := parsed.RoomAddress
		t.Log(addr)
		break
	}
}

func TestChainContractInteraction(t *testing.T) {
	workerID := "123"
	gameID := 123
	player1 := types.PlayerAddress{
		WalletAddress:    "0x123",
		TemporaryAddress: "0x456",
	}
	player2 := types.PlayerAddress{
		WalletAddress:    "0x789",
		TemporaryAddress: "0xabc",
	}

	svc := NewService(context.Background(), testWorkerManager, 707, client, roomMamangerAddress, w, 10, 3)

	svc.chain.bindOpts.NoSend = true
	mockRoomHandler := tt.NewMockEventHandler(gomock.NewController(t))
	ackReceived := make(chan struct{})
	mockRoomHandler.EXPECT().Handle(gomock.Any(), gomock.Any(), tt.NewEventTypeMatcher(&types.AckEvent{})).AnyTimes().DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		close(ackReceived)
		return nil
	})
	mockRoomHandler.EXPECT().Handle(gomock.Any(), gomock.Any(), tt.NewEventTypeMatcher(&types.ErrorEvent{})).AnyTimes().DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.ErrorEvent)
		t.Errorf("ErrorEvent should not be sent, err: %v", evt.Err)
		close(ackReceived)
		return nil
	})
	testWorkerManager.SpwanWorker(context.Background(), workerID, types.WORKER_TYPE_GAME, mockRoomHandler)
	testWorkerManager.SendEvent(types.CHAIN_MANAGER_ID, types.NewEvent(workerID, &types.RequireContractCreationEvent{
		GameID:  uint(gameID),
		Players: []types.PlayerAddress{player1, player2},
	}, true))
	<-ackReceived
	tx, err := db.GetCreateRoomTx(uint(gameID))
	require.NoError(t, err)
	require.NotEmpty(t, tx)
	ackReceived = make(chan struct{})
	testWorkerManager.SendEvent(types.CHAIN_MANAGER_ID, types.NewEvent(workerID, &types.RequireSetupNewRoundEvent{
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
	mockRoomHandler.EXPECT().Handle(gomock.Any(), gomock.Any(), evtMatcher).Times(5).Return(nil)
	err = svc.ReceiveTransactions(1, &proto.TransactionBatch{
		BlockHash: []byte("0x123"),
		Timestamp: uint64(time.Now().Unix()),
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
