package evmrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

const (
	rpcHTTPClientTimeout   = 5 * time.Second
	rpcMaxIdleConns        = 100
	rpcMaxIdleConnsPerHost = 32
	rpcIdleConnTimeout     = 90 * time.Second
)

func newRPCHTTPTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = rpcMaxIdleConns
	transport.MaxIdleConnsPerHost = rpcMaxIdleConnsPerHost
	transport.IdleConnTimeout = rpcIdleConnTimeout
	return transport
}

var defaultHTTPClient = &http.Client{
	Timeout:   rpcHTTPClientTimeout,
	Transport: newRPCHTTPTransport(),
}

// Block is a generic EVM JSON-RPC block payload.
type Block struct {
	Number       string        `json:"number"`
	Hash         string        `json:"hash"`
	ParentHash   string        `json:"parentHash"`
	Timestamp    string        `json:"timestamp"`
	Transactions []interface{} `json:"transactions"`
}

func rpcPost(ctx context.Context, rpcURL string, reqBody any, result any) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("rpc error: %s", rpcResp.Error.Message)
	}
	if len(rpcResp.Result) == 0 || string(rpcResp.Result) == "null" {
		return fmt.Errorf("empty rpc result")
	}
	return json.Unmarshal(rpcResp.Result, result)
}

// GetBlockByNumber fetches a block by height or tag (e.g. "finalized", "latest").
func GetBlockByNumber(ctx context.Context, rpcURL string, height any, fullTx bool) (*Block, error) {
	var heightParam any
	switch v := height.(type) {
	case *big.Int:
		heightParam = fmt.Sprintf("0x%x", v)
	case uint64:
		heightParam = fmt.Sprintf("0x%x", v)
	case string:
		heightParam = v
	default:
		return nil, fmt.Errorf("unsupported block height type %T", height)
	}

	var block Block
	err := rpcPost(ctx, rpcURL, map[string]any{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []any{heightParam, fullTx},
		"id":      1,
	}, &block)
	if err != nil {
		return nil, err
	}
	if block.Number == "" {
		return nil, fmt.Errorf("block not found at height %v", heightParam)
	}
	return &block, nil
}

// GetFinalizedBlockNumber returns the current finalized block height.
func GetFinalizedBlockNumber(ctx context.Context, rpcURL string) (uint64, error) {
	block, err := GetBlockByNumber(ctx, rpcURL, "finalized", false)
	if err != nil {
		return 0, err
	}
	return parseHexUint64(block.Number)
}

// GetChainID returns the chain ID from eth_chainId.
func GetChainID(ctx context.Context, rpcURL string) (uint64, error) {
	var chainIDHex string
	err := rpcPost(ctx, rpcURL, map[string]any{
		"jsonrpc": "2.0",
		"method":  "eth_chainId",
		"params":  []any{},
		"id":      1,
	}, &chainIDHex)
	if err != nil {
		return 0, err
	}
	return parseHexUint64(chainIDHex)
}

func parseHexUint64(hexStr string) (uint64, error) {
	var n uint64
	_, err := fmt.Sscanf(hexStr, "0x%x", &n)
	return n, err
}
