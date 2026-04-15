package chain

import (
	"context"
	"os"
	"testing"
	"time"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

var client *ethclient.Client

const roomMamangerAddress = "0x20ae7393Fe6eC4218E0E27452Cf158FC4c1Ba06C"

func TestMain(m *testing.M) {
	time.Local = time.UTC
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
	ethC, err := ethclient.Dial("http://152.32.231.145:8545")
	if err != nil {
		panic(err)
	}
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
