package evmrpc

import (
	"context"
	"fmt"
	"sort"
)

// ReceiptLog is a log entry from eth_getBlockReceipts with ordering metadata.
type ReceiptLog struct {
	Address  string   `json:"address"`
	Topics   []string `json:"topics"`
	Data     string   `json:"data"`
	TxHash   string   `json:"transactionHash"`
	TxIndex  uint32   `json:"-"`
	LogIndex uint32   `json:"logIndex"`
}

type blockReceipt struct {
	TransactionHash  string `json:"transactionHash"`
	TransactionIndex string `json:"transactionIndex"`
	Status           string `json:"status"`
	Logs             []struct {
		Address     string   `json:"address"`
		Topics      []string `json:"topics"`
		Data        string   `json:"data"`
		LogIndex    string   `json:"logIndex"`
		TxHash      string   `json:"transactionHash"`
		TxIndex     string   `json:"transactionIndex"`
	} `json:"logs"`
}

// GetBlockReceiptLogs fetches all logs in a block via eth_getBlockReceipts, sorted by tx/log index.
func GetBlockReceiptLogs(ctx context.Context, rpcURL string, blockNum uint64) ([]ReceiptLog, error) {
	var receipts []blockReceipt
	err := rpcPost(ctx, rpcURL, map[string]any{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockReceipts",
		"params":  []any{fmt.Sprintf("0x%x", blockNum)},
		"id":      1,
	}, &receipts)
	if err != nil {
		return nil, err
	}

	var logs []ReceiptLog
	for _, receipt := range receipts {
		txIndex := parseHexUint32(receipt.TransactionIndex)
		statusOK := receipt.Status == "" || receipt.Status == "0x1"
		if !statusOK {
			continue
		}
		for _, lg := range receipt.Logs {
			logs = append(logs, ReceiptLog{
				Address:  lg.Address,
				Topics:   lg.Topics,
				Data:     lg.Data,
				TxHash:   firstNonEmpty(lg.TxHash, receipt.TransactionHash),
				TxIndex:  firstNonZero(parseHexUint32(lg.TxIndex), txIndex),
				LogIndex: parseHexUint32(lg.LogIndex),
			})
		}
	}

	sort.Slice(logs, func(i, j int) bool {
		if logs[i].TxIndex != logs[j].TxIndex {
			return logs[i].TxIndex < logs[j].TxIndex
		}
		return logs[i].LogIndex < logs[j].LogIndex
	})
	return logs, nil
}

func parseHexUint32(hexStr string) uint32 {
	if hexStr == "" {
		return 0
	}
	var n uint64
	if _, err := fmt.Sscanf(hexStr, "0x%x", &n); err != nil {
		return 0
	}
	return uint32(n)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonZero(values ...uint32) uint32 {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}
