package chain

import (
	"context"
	"os"
	"testing"
	"time"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

var testWorkerManager *worker.WorkerManager
var client *ethclient.Client
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
	for {
		receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash("0x3a9738a38f35ce59fea8d56ef6ca74d81e46e57b99bdf5a8575f92975ce25e98"))
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
	gameID := int64(123)
	player1 := types.PlayerAddress{
		Id:               123,
		TemporaryAddress: "0x456",
	}
	player2 := types.PlayerAddress{
		Id:               789,
		TemporaryAddress: "0xabc",
	}

	svc, err := NewChain(context.Background(), testWorkerManager, int64(chainID), client, roomContractAddress, []*wallet.Wallet{w})
	require.NoError(t, err)

	svc.NotifyTxsCompleted(&proto.TransactionBatch{
		BlockHash:   []byte("0x123"),
		Timestamp:   uint64(time.Now().Unix()),
		BlockNumber: 1,
		Transactions: []*proto.Transaction{
			{
				TxHash: []byte("test_tx_hash"),
				GameId: gameID,
				Tx:     &proto.Transaction_GameCreated{GameCreated: &proto.TxGameCreated{}},
			},
			{
				TxHash: []byte("GameTurnSetupReady"),
				GameId: gameID,
				Tx: &proto.Transaction_GameTurnSetupReady{
					GameTurnSetupReady: &proto.TxGameTurnSetupReady{
						RoundNumber: 2,
						TurnNumber:  1,
					},
				},
			},
			{
				TxHash: []byte("Transaction_CommitmentOnChain"),
				GameId: gameID,
				Tx: &proto.Transaction_CommitmentOnChain{
					CommitmentOnChain: &proto.TxCommitmentOnChain{
						Address:     player1.ToProto(),
						RoundNumber: 2,
						TurnNumber:  1,
						Commitment:  []byte("0xcommitment"),
					},
				},
			},
			{
				TxHash: []byte("Transaction_CardOnChain_1"),
				GameId: gameID,
				Tx: &proto.Transaction_CardOnChain{
					CardOnChain: &proto.TxCardOnChain{
						Address:     player1.ToProto(),
						RoundNumber: 2,
						TurnNumber:  1,
						Salt:        []byte("0x123"),
						CardId:      1,
					},
				},
			},
			{
				TxHash: []byte("Transaction_CardOnChain_2"),
				GameId: gameID,
				Tx: &proto.Transaction_CardOnChain{
					CardOnChain: &proto.TxCardOnChain{
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
}
