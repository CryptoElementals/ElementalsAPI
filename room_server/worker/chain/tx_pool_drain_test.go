package chain

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/stretchr/testify/require"
)

type fakeBatchSubmitter struct {
	calls    int
	batches  [][]types.RoomContractTask
	failCall int
}

func (f *fakeBatchSubmitter) SubmitTasks(tasks []types.RoomContractTask) error {
	f.calls++
	f.batches = append(f.batches, tasks)
	if f.failCall > 0 && f.calls == f.failCall {
		return errors.New("submit failed")
	}
	return nil
}

func makeCreateRoomRow(id uint, gameID int64) db.ChainTxPoolPendingRow {
	payload, _ := json.Marshal(&types.RequireGameCreationEvent{
		GameID:        gameID,
		RoundTimeout:  10,
		MaxRoundNumber: 3,
		InitialHP:     30,
		Players: []types.PlayerAddress{
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
	origPop := popChainTxPoolBatchForChain
	defer func() { popChainTxPoolBatchForChain = origPop }()

	popCalls := 0
	popChainTxPoolBatchForChain = func(chainID int64, limit int) ([]db.ChainTxPoolPendingRow, error) {
		popCalls++
		require.EqualValues(t, 9, chainID)
		require.Equal(t, 2, limit)
		switch popCalls {
		case 1:
			return []db.ChainTxPoolPendingRow{makeCreateRoomRow(1, 100)}, nil
		case 2:
			return []db.ChainTxPoolPendingRow{makeCreateRoomRow(2, 101)}, nil
		default:
			return nil, nil
		}
	}

	sub := &fakeBatchSubmitter{}
	pool := newTxPoolWithSubmitter(sub, 9, 2)
	require.NoError(t, pool.runDrainLoop())

	require.Equal(t, 3, popCalls)
	require.Equal(t, 2, sub.calls)
	require.Len(t, sub.batches, 2)
	require.Len(t, sub.batches[0], 1)
	require.Len(t, sub.batches[1], 1)
}

func TestRunDrainLoopStopsOnSubmitFailure(t *testing.T) {
	origPop := popChainTxPoolBatchForChain
	defer func() { popChainTxPoolBatchForChain = origPop }()

	popCalls := 0
	popChainTxPoolBatchForChain = func(chainID int64, limit int) ([]db.ChainTxPoolPendingRow, error) {
		popCalls++
		if popCalls == 1 {
			return []db.ChainTxPoolPendingRow{makeCreateRoomRow(1, 100)}, nil
		}
		return []db.ChainTxPoolPendingRow{makeCreateRoomRow(2, 101)}, nil
	}

	sub := &fakeBatchSubmitter{failCall: 1}
	pool := newTxPoolWithSubmitter(sub, 9, 2)
	require.NoError(t, pool.runDrainLoop())

	require.Equal(t, 1, popCalls)
	require.Equal(t, 1, sub.calls)
}
