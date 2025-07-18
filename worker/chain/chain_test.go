package chain

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/worker"
	"github.com/ethereum/go-ethereum/ethclient"
)

var testWorkerManager *worker.WorkerManager
var client *ethclient.Client

const roomMamangerAddress = "0xFFD251cBd389e482B0609D3B6389a1350827A6C2"

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
	client, err = ethclient.Dial("http://117.50.80.239:8545")
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestChainEvents(t *testing.T) {
	// svc := NewService(context.Background(), testWorkerManager, client, roomMamangerAddress, 10, 3)

}
