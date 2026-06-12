package scanner

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenCollectorEventJSONDeposit(t *testing.T) {
	ev := TokenCollectorEvent{
		ChainID:          97,
		BlockNumber:      100,
		BlockHash:        "0xabc",
		Timestamp:        1700000000,
		TxHash:           "0xtx",
		LogIndex:         3,
		CollectorAddress: "0xcollector",
		EventType:        eventTypeDeposit,
		Deposit: &DepositPayload{
			PlayerID:    42,
			FromAddress: "0xfrom",
			Amount:      "1000",
			NewCredited: "5000",
		},
	}

	data, err := json.Marshal(ev)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	require.Contains(t, raw, "deposit")
	require.NotContains(t, raw, "withdraw")

	var deposit DepositPayload
	require.NoError(t, json.Unmarshal(raw["deposit"], &deposit))
	require.Equal(t, int64(42), deposit.PlayerID)
	require.Equal(t, "0xfrom", deposit.FromAddress)
	require.Equal(t, "1000", deposit.Amount)
	require.Equal(t, "5000", deposit.NewCredited)
}

func TestTokenCollectorEventJSONWithdraw(t *testing.T) {
	ev := TokenCollectorEvent{
		ChainID:          97,
		BlockNumber:      101,
		BlockHash:        "0xdef",
		Timestamp:        1700000001,
		TxHash:           "0xtx2",
		LogIndex:         1,
		CollectorAddress: "0xcollector",
		EventType:        eventTypeWithdraw,
		Withdraw: &WithdrawPayload{
			PlayerID:  7,
			Operator:  "0xop",
			ToAddress: "0xto",
			Amount:    "2000",
		},
	}

	data, err := json.Marshal(ev)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	require.Contains(t, raw, "withdraw")
	require.NotContains(t, raw, "deposit")

	var withdraw WithdrawPayload
	require.NoError(t, json.Unmarshal(raw["withdraw"], &withdraw))
	require.Equal(t, int64(7), withdraw.PlayerID)
	require.Equal(t, "0xop", withdraw.Operator)
	require.Equal(t, "0xto", withdraw.ToAddress)
	require.Equal(t, "2000", withdraw.Amount)
}
