package db

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
	gorm_logger "gorm.io/gorm/logger"
)

func setupTestDBForServerTypePromote(t *testing.T) {
	t.Helper()
	require.NoError(t, Init(&Config{Development: true}))
	Get().Logger = Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, MigrateMemDb())
}

func TestPromoteUserServerTypeToNormalIfTrial(t *testing.T) {
	setupTestDBForServerTypePromote(t)

	trialProfile := &dao.UserProfile{
		PlayerID:   4001,
		Address:    "0xservertype1",
		Name:       "server_type_trial",
		ServerType: dao.ServerTypeTrial,
	}
	require.NoError(t, Get().Create(trialProfile).Error)

	normalProfile := &dao.UserProfile{
		PlayerID:   4002,
		Address:    "0xservertype2",
		Name:       "server_type_normal",
		ServerType: dao.ServerTypeNormal,
	}
	require.NoError(t, Get().Create(normalProfile).Error)

	updated, err := PromoteUserServerTypeToNormalIfTrial(4001)
	require.NoError(t, err)
	require.True(t, updated)

	profile, err := GetUserProfileByPlayerIDInt(4001)
	require.NoError(t, err)
	require.Equal(t, dao.ServerTypeNormal, profile.ServerType)

	updated, err = PromoteUserServerTypeToNormalIfTrial(4001)
	require.NoError(t, err)
	require.False(t, updated)

	updated, err = PromoteUserServerTypeToNormalIfTrial(4002)
	require.NoError(t, err)
	require.False(t, updated)

	updated, err = PromoteUserServerTypeToNormalIfTrial(99999)
	require.NoError(t, err)
	require.False(t, updated)
}
