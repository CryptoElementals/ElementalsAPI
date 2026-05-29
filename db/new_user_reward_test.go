package db

import (
	"errors"
	"strconv"
	"sync"
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
)

func setupTestDBForNewUserReward(t *testing.T) {
	t.Helper()
	require.NoError(t, Init(&Config{Development: true}))
	Get().Logger = Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, MigrateMemDb())
}

func TestNewUserRewardStatusAndMark(t *testing.T) {
	setupTestDBForNewUserReward(t)

	profile := &dao.UserProfile{
		PlayerID: 3001,
		Address:  "0xnewuserreward1",
		Name:     "new_user_reward_1",
	}
	require.NoError(t, Get().Create(profile).Error)

	playerID := strconv.FormatInt(profile.PlayerID, 10)

	collected, err := HasCollectedNewUserRewardByPlayerID(playerID)
	require.NoError(t, err)
	require.False(t, collected)

	err = Get().Transaction(func(tx *gorm.DB) error {
		marked, markErr := MarkNewUserRewardCollectedByPlayerIDTx(tx, playerID)
		require.NoError(t, markErr)
		require.True(t, marked)
		return nil
	})
	require.NoError(t, err)

	collected, err = HasCollectedNewUserRewardByPlayerID(playerID)
	require.NoError(t, err)
	require.True(t, collected)

	err = Get().Transaction(func(tx *gorm.DB) error {
		marked, markErr := MarkNewUserRewardCollectedByPlayerIDTx(tx, playerID)
		require.NoError(t, markErr)
		require.False(t, marked)
		return nil
	})
	require.NoError(t, err)
}

func TestMarkNewUserRewardRollback(t *testing.T) {
	setupTestDBForNewUserReward(t)

	profile := &dao.UserProfile{
		PlayerID: 3002,
		Address:  "0xnewuserreward2",
		Name:     "new_user_reward_2",
	}
	require.NoError(t, Get().Create(profile).Error)
	playerID := strconv.FormatInt(profile.PlayerID, 10)

	rollbackErr := errors.New("rollback sentinel")
	err := Get().Transaction(func(tx *gorm.DB) error {
		marked, markErr := MarkNewUserRewardCollectedByPlayerIDTx(tx, playerID)
		require.NoError(t, markErr)
		require.True(t, marked)
		return rollbackErr
	})
	require.Error(t, err)

	collected, err := HasCollectedNewUserRewardByPlayerID(playerID)
	require.NoError(t, err)
	require.False(t, collected)
}

func TestMarkNewUserRewardConcurrentOnlyOneWins(t *testing.T) {
	setupTestDBForNewUserReward(t)

	profile := &dao.UserProfile{
		PlayerID: 3003,
		Address:  "0xnewuserreward3",
		Name:     "new_user_reward_3",
	}
	require.NoError(t, Get().Create(profile).Error)
	playerID := strconv.FormatInt(profile.PlayerID, 10)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCnt := 0
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := Get().Transaction(func(tx *gorm.DB) error {
				marked, markErr := MarkNewUserRewardCollectedByPlayerIDTx(tx, playerID)
				if markErr != nil {
					return markErr
				}
				if marked {
					mu.Lock()
					successCnt++
					mu.Unlock()
				}
				return nil
			})
			require.NoError(t, err)
		}()
	}
	wg.Wait()
	require.Equal(t, 1, successCnt)
}
