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

func TestUpdateUserProfileAddressFromDeposit(t *testing.T) {
	require.NoError(t, initMemDbSqlite())
	require.NoError(t, MigrateMemDb())

	profile := &dao.UserProfile{
		PlayerID:   6001,
		Address:    "",
		Name:       "deposit_addr_user",
		ServerType: dao.ServerTypeTrial,
	}
	require.NoError(t, Get().Create(profile).Error)

	updated, err := UpdateUserProfileAddressFromDeposit(6001, "0xDepositorABC")
	require.NoError(t, err)
	require.True(t, updated)

	got, err := GetUserProfileByPlayerIDInt(6001)
	require.NoError(t, err)
	require.Equal(t, "0xdepositorabc", got.Address)

	updated, err = UpdateUserProfileAddressFromDeposit(6001, "0xother")
	require.NoError(t, err)
	require.True(t, updated)

	got, err = GetUserProfileByPlayerIDInt(6001)
	require.NoError(t, err)
	require.Equal(t, "0xother", got.Address)
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
