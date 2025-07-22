package blockchain

import (
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	contracts "github.com/CryptoElementals/common/contracts"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestCreateRoomAndWaitReceiptAndParseEvent(t *testing.T) {
	rpcUrl := os.Getenv("OPSTACK_RPC_URL")
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	contractAddr := os.Getenv("ROOM_MANAGER_CONTRACT")
	chainIDStr := os.Getenv("CHAIN_ID")

	if rpcUrl == "" || privateKeyHex == "" || contractAddr == "" || chainIDStr == "" {
		t.Skip("Set OPSTACK_RPC_URL, PRIVATE_KEY, ROOM_MANAGER_CONTRACT, CHAIN_ID env to run this test")
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Could not get current test file path")
	}
	abiPath := filepath.Join(filepath.Dir(filename), "../../../contracts/room_manager.abi")
	// 读取并解析 ABI
	abiBytes, err := os.ReadFile(abiPath)
	if err != nil {
		t.Fatalf("Failed to read ABI file: %v", err)
	}
	contractAbi, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		t.Fatalf("Failed to parse ABI: %v", err)
	}

	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		t.Fatalf("Failed to connect to RPC: %v", err)
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}
	chainID := new(big.Int)
	chainID.SetString(chainIDStr, 10)
	bindOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		t.Fatalf("Failed to create bind opts: %v", err)
	}

	// 合约实例
	roomManager, err := contracts.NewRoomManagerContract(common.HexToAddress(contractAddr), client)
	if err != nil {
		t.Fatalf("Failed to instantiate contract: %v", err)
	}

	player1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	player2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	temp1 := common.HexToAddress("0x3333333333333333333333333333333333333333")
	temp2 := common.HexToAddress("0x4444444444444444444444444444444444444444")
	roundTimeout := big.NewInt(60)
	maxRounds := big.NewInt(3)
	timeout := 3 * time.Minute

	eventData, err := CreateRoomAndWaitReceiptAndParseEvent(
		client, common.HexToAddress(contractAddr), &contractAbi, roomManager, bindOpts,
		player1, player2, temp1, temp2, roundTimeout, maxRounds, timeout,
	)
	if err != nil {
		t.Fatalf("CreateRoomAndWaitReceiptAndParseEvent failed: %v", err)
	}
	t.Logf("RoomCreated event data: %+v", eventData)
}
