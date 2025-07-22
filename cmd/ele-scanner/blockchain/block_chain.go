package blockchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

// Global HTTP client for connection reuse
var defaultHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

// OptimismBlock represents a block returned by Optimism JSON-RPC
type OptimismBlock struct {
	Number       string        `json:"number"`
	Hash         string        `json:"hash"`
	ParentHash   string        `json:"parentHash"`
	Timestamp    string        `json:"timestamp"`
	Transactions []interface{} `json:"transactions"`
}

// OptimismTx represents a parsed transaction in Optimism
type OptimismTx struct {
	Hash      string `json:"hash"`
	From      string `json:"from"`
	To        string `json:"to,omitempty"`
	Value     string `json:"value"`
	Type      string `json:"type"`
	Input     string `json:"input"`
	Nonce     string `json:"nonce"`
	Gas       string `json:"gas"`
	GasPrice  string `json:"gasPrice"`
	BlockHash string `json:"blockHash"`
}

// GetOptimismBlockByNumber fetches and parses a block from Optimism by height
func GetOptimismBlockByNumber(ctx context.Context, rpcUrl string, height *big.Int) (*OptimismBlock, error) {
	// Build JSON-RPC request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []interface{}{fmt.Sprintf("0x%x", height), true},
		"id":      1,
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", rpcUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result *OptimismBlock `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if rpcResp.Result == nil {
		return nil, fmt.Errorf("block not found at height %s", height.String())
	}
	return rpcResp.Result, nil
}

// ParseOptimismTransactions parses the transactions field of an Optimism block
func ParseOptimismTransactions(txs []interface{}) ([]OptimismTx, error) {
	var result []OptimismTx
	for i, tx := range txs {
		txMap, ok := tx.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("transaction %d is not a map", i)
		}
		parsed := OptimismTx{
			Hash:      getString(txMap, "hash"),
			From:      getString(txMap, "from"),
			To:        getString(txMap, "to"),
			Value:     getString(txMap, "value"),
			Type:      getString(txMap, "type"),
			Input:     getString(txMap, "input"),
			Nonce:     getString(txMap, "nonce"),
			Gas:       getString(txMap, "gas"),
			GasPrice:  getString(txMap, "gasPrice"),
			BlockHash: getString(txMap, "blockHash"),
		}
		result = append(result, parsed)
	}
	return result, nil
}

// getString safely gets a string from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
