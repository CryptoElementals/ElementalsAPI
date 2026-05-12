package api

import (
	"strconv"
	"testing"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
	gorm_logger "gorm.io/gorm/logger"
)

func setupTestDBForDailyRewardAPI(t *testing.T) {
	t.Helper()
	require.NoError(t, db.Init(&db.Config{Development: true}))
	db.Get().Logger = db.Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, db.MigrateMemDb())
}

func TestDailyReward_DisabledByConfig(t *testing.T) {
	setupTestDBForDailyRewardAPI(t)
	config.InitializeGameParams(&config.GameParamConfig{
		EnableDailyReward: false,
	})

	profile := &dao.UserProfile{
		PlayerID: 5001,
		Address:  "0xdaily_reward_disabled",
		Name:     "daily_reward_disabled",
	}
	require.NoError(t, db.Get().Create(profile).Error)

	playerID := strconv.FormatInt(profile.PlayerID, 10)

	checkParams := map[string]interface{}{
		"Action":      HAS_COLLECTED_DAILY_REWARD_LABEL,
		"RequestUUID": "test-daily-has-disabled",
		"PlayerID":    playerID,
	}
	checkTaskIntf, err := NewHasCollectedDailyRewardTask(&checkParams)
	require.NoError(t, err)
	_, err = checkTaskIntf.Run(setupGinContext())
	require.Error(t, err)
	hasErr, ok := err.(cmnErrors.Error)
	require.True(t, ok)
	require.Contains(t, hasErr.String(), "not enabled")

	collectParams := map[string]interface{}{
		"Action":      COLLECT_DAILY_REWARD_LABEL,
		"RequestUUID": "test-daily-collect-disabled",
		"PlayerID":    playerID,
	}
	collectTaskIntf, err := NewCollectDailyRewardTask(&collectParams)
	require.NoError(t, err)
	_, err = collectTaskIntf.Run(setupGinContext())
	require.Error(t, err)
	customErr, ok := err.(cmnErrors.Error)
	require.True(t, ok)
	require.Contains(t, customErr.String(), "not enabled")
}
