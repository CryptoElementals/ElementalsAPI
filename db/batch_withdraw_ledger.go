package db

import (
	"encoding/hex"
	"errors"
	"strings"

	dao "github.com/CryptoElementals/common/models"
)

func InsertBatchWithdrawLedger(row *dao.BatchWithdrawLedger) (uint, error) {
	if row == nil {
		return 0, errors.New("nil batch withdraw ledger row")
	}
	row.CollectorAddress = strings.ToLower(strings.TrimSpace(row.CollectorAddress))
	row.TxHash = strings.ToLower(strings.TrimSpace(row.TxHash))
	row.Signature = strings.ToLower(strings.TrimSpace(row.Signature))
	if err := Get().Create(row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func FormatWithdrawSignatureHex(signature []byte) string {
	return "0x" + strings.ToLower(hex.EncodeToString(signature))
}
