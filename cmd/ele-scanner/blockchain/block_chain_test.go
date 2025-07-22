package blockchain

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"
)

func TestGetOptimismBlockByNumber(t *testing.T) {
	rpcUrl := os.Getenv("OPSTACK_RPC_URL")
	if rpcUrl == "" {
		t.Skip("Set OPSTACK_RPC_URL env to run this test")
	}
	height := big.NewInt(100) // Replace with a real block height on your chain

	block, err := GetOptimismBlockByNumber(context.Background(), rpcUrl, height)
	if err != nil {
		t.Fatalf("Failed to get block: %v", err)
	}
	if block.Number == "" || block.Hash == "" {
		t.Errorf("Block fields should not be empty: %+v", block)
	}
	t.Logf("Block number: %s, hash: %s, tx count: %d", block.Number, block.Hash, len(block.Transactions))

	// 解析交易
	parsedTxs, err := ParseOptimismTransactions(block.Transactions)
	if err != nil {
		t.Fatalf("ParseOptimismTransactions failed: %v", err)
	}
	if len(parsedTxs) != len(block.Transactions) {
		t.Errorf("Parsed tx count %d does not match raw tx count %d", len(parsedTxs), len(block.Transactions))
	}
	if len(parsedTxs) > 0 {
		t.Logf("First parsed tx: %+v", parsedTxs[0])
	}
}

func TestParseOptimismTransactions(t *testing.T) {
	blockJson := `
	{
		"number": "0x12345",
		"transactions": [
			{
				"hash": "0xabc",
				"from": "0x111",
				"to": "0x222",
				"value": "0x10",
				"type": "0x0",
				"input": "0x",
				"nonce": "0x1",
				"gas": "0x5208",
				"gasPrice": "0x3b9aca00",
				"blockHash": "0xdef"
			},
			{
				"hash": "0xdef",
				"from": "0x333",
				"to": null,
				"value": "0x20",
				"type": "0x7e",
				"input": "0x1234",
				"nonce": "0x2",
				"gas": "0x5208",
				"gasPrice": "0x3b9aca00",
				"blockHash": "0x456"
			}
		]
	}`

	var block map[string]interface{}
	if err := json.Unmarshal([]byte(blockJson), &block); err != nil {
		t.Fatalf("Failed to unmarshal block: %v", err)
	}
	txs, ok := block["transactions"].([]interface{})
	if !ok {
		t.Fatalf("transactions field not found or not array")
	}

	parsed, err := ParseOptimismTransactions(txs)
	if err != nil {
		t.Fatalf("ParseOptimismTransactions failed: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(parsed))
	}
	if parsed[0].Hash != "0xabc" || parsed[1].Type != "0x7e" {
		t.Errorf("unexpected parse result: %+v", parsed)
	}
	t.Logf("Parsed transactions: %+v", parsed)
}
