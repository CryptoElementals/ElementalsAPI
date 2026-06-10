package worker

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
)

type fakeBatchSubmitter struct {
	calls    int
	batches  [][]RoomContractTask
	failCall int
}

func (f *fakeBatchSubmitter) SubmitTasks(tasks []RoomContractTask) error {
	f.calls++
	f.batches = append(f.batches, tasks)
	if f.failCall > 0 && f.calls == f.failCall {
		return errors.New("submit failed")
	}
	return nil
}

func makeCreateRoomRow(id uint, gameID int64) db.ChainTxPoolPendingRow {
	payload, _ := json.Marshal(&RequireGameCreationEvent{
		GameID:         gameID,
		RoundTimeout:   10,
		MaxRoundNumber: 3,
		InitialHP:      30,
		Players: []PlayerAddress{
			{Id: 1, TemporaryAddress: "0x0000000000000000000000000000000000000001"},
			{Id: 2, TemporaryAddress: "0x0000000000000000000000000000000000000002"},
		},
	})
	return db.ChainTxPoolPendingRow{
		ID:      id,
		Kind:    dao.ChainTxPoolKindCreateRoom,
		GameID:  gameID,
		Payload: payload,
	}
}

func TestRunDrainLoopSubmitsUntilPoolDrained(t *testing.T) {
	origClaim := claimChainTxPoolBatchForChain
	origDelete := deleteChainTxPoolItemsByIDs
	defer func() {
		claimChainTxPoolBatchForChain = origClaim
		deleteChainTxPoolItemsByIDs = origDelete
	}()

	claimCalls := 0
	claimChainTxPoolBatchForChain = func(chainID int64, limit int, claimTimeout time.Duration) ([]db.ChainTxPoolPendingRow, error) {
		claimCalls++
		require.EqualValues(t, 9, chainID)
		require.Equal(t, 2, limit)
		require.Equal(t, 2*time.Second, claimTimeout)
		switch claimCalls {
		case 1:
			return []db.ChainTxPoolPendingRow{makeCreateRoomRow(1, 100)}, nil
		case 2:
			return []db.ChainTxPoolPendingRow{makeCreateRoomRow(2, 101)}, nil
		default:
			return nil, nil
		}
	}

	var deleted [][]uint
	deleteChainTxPoolItemsByIDs = func(ids []uint) error {
		copied := append([]uint(nil), ids...)
		deleted = append(deleted, copied)
		return nil
	}

	sub := &fakeBatchSubmitter{}
	pool := newTxPoolWithSubmitter(sub, 9, 2, 2*time.Second)
	require.NoError(t, pool.runDrainLoop())

	require.Equal(t, 3, claimCalls)
	require.Equal(t, 2, sub.calls)
	require.Len(t, sub.batches, 2)
	require.Len(t, sub.batches[0], 1)
	require.Len(t, sub.batches[1], 1)
	require.Len(t, deleted, 2)
}

func TestRunDrainLoopStopsOnSubmitFailure(t *testing.T) {
	origClaim := claimChainTxPoolBatchForChain
	defer func() { claimChainTxPoolBatchForChain = origClaim }()

	claimCalls := 0
	claimChainTxPoolBatchForChain = func(chainID int64, limit int, claimTimeout time.Duration) ([]db.ChainTxPoolPendingRow, error) {
		claimCalls++
		if claimCalls == 1 {
			return []db.ChainTxPoolPendingRow{makeCreateRoomRow(1, 100)}, nil
		}
		return []db.ChainTxPoolPendingRow{makeCreateRoomRow(2, 101)}, nil
	}

	sub := &fakeBatchSubmitter{failCall: 1}
	pool := newTxPoolWithSubmitter(sub, 9, 2, 2*time.Second)
	require.NoError(t, pool.runDrainLoop())

	require.Equal(t, 1, claimCalls)
	require.Equal(t, 1, sub.calls)
}
