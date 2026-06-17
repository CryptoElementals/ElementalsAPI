package evmrpc

import "fmt"

// Tx is a parsed transaction from a block.
type Tx struct {
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to,omitempty"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	Input       string `json:"input"`
	Nonce       string `json:"nonce"`
	Gas         string `json:"gas"`
	GasPrice    string `json:"gasPrice"`
	BlockHash   string `json:"blockHash"`
	TxIndex     uint32 `json:"transactionIndex"`
	BlockNumber string `json:"blockNumber"`
}

// ParseTransactions parses the transactions field of a block.
func ParseTransactions(txs []interface{}) ([]Tx, error) {
	var result []Tx
	for i, tx := range txs {
		txMap, ok := tx.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("transaction %d is not a map", i)
		}
		parsed := Tx{
			Hash:        getString(txMap, "hash"),
			From:        getString(txMap, "from"),
			To:          getString(txMap, "to"),
			Value:       getString(txMap, "value"),
			Type:        getString(txMap, "type"),
			Input:       getString(txMap, "input"),
			Nonce:       getString(txMap, "nonce"),
			Gas:         getString(txMap, "gas"),
			GasPrice:    getString(txMap, "gasPrice"),
			BlockHash:   getString(txMap, "blockHash"),
			BlockNumber: getString(txMap, "blockNumber"),
		}
		if idx := getString(txMap, "transactionIndex"); idx != "" {
			var txIndex uint64
			if _, err := fmt.Sscanf(idx, "0x%x", &txIndex); err == nil {
				parsed.TxIndex = uint32(txIndex)
			}
		} else {
			parsed.TxIndex = uint32(i)
		}
		result = append(result, parsed)
	}
	return result, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
