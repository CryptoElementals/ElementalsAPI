package db

import (
	"testing"
	"time"

	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
)

func setupUserTokenTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, Init(&Config{Development: true}))
	Get().Logger = Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, MigrateMemDb())
}

func countUserTokensUnscoped(t *testing.T, playerID int64) (active, softDeleted int64) {
	t.Helper()
	require.NoError(t, Get().Model(&dao.UserToken{}).Where("player_id = ?", playerID).Count(&active).Error)
	require.NoError(t, Get().Unscoped().Model(&dao.UserToken{}).
		Where("player_id = ? AND deleted_at IS NOT NULL", playerID).Count(&softDeleted).Error)
	return active, softDeleted
}

func TestEnsureUserTokenByPlayerIDIdempotent(t *testing.T) {
	setupUserTokenTestDB(t)

	first, err := EnsureUserTokenByPlayerID(77001)
	require.NoError(t, err)
	require.NotZero(t, first.ID)

	second, err := EnsureUserTokenByPlayerID(77001)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)

	active, softDeleted := countUserTokensUnscoped(t, 77001)
	require.Equal(t, int64(1), active)
	require.Equal(t, int64(0), softDeleted)
}

func TestEnsureUserTokenAfterSoftDeleteCreatesNewActiveRow(t *testing.T) {
	setupUserTokenTestDB(t)

	first, err := EnsureUserTokenByPlayerID(77002)
	require.NoError(t, err)
	require.NoError(t, Get().Delete(first).Error)

	active, softDeleted := countUserTokensUnscoped(t, 77002)
	require.Equal(t, int64(0), active)
	require.Equal(t, int64(1), softDeleted)

	second, err := EnsureUserTokenByPlayerID(77002)
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)

	active, softDeleted = countUserTokensUnscoped(t, 77002)
	require.Equal(t, int64(1), active)
	require.Equal(t, int64(1), softDeleted)
}

func TestUserTokenMultipleSoftDeletedPlusOneActiveAllowed(t *testing.T) {
	setupUserTokenTestDB(t)

	require.NoError(t, Get().Create(&dao.UserToken{PlayerId: 77003, Points: 1}).Error)
	require.NoError(t, Get().Where("player_id = ?", 77003).Delete(&dao.UserToken{}).Error)
	require.NoError(t, Get().Create(&dao.UserToken{PlayerId: 77003, Points: 2}).Error)
	require.NoError(t, Get().Where("player_id = ?", 77003).Delete(&dao.UserToken{}).Error)

	past := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, Get().Unscoped().Create(&dao.UserToken{
		BaseModel: dao.BaseModel{DeletedAt: gorm.DeletedAt{Time: past, Valid: true}},
		PlayerId:  77003,
		Points:    3,
	}).Error)

	token, err := EnsureUserTokenByPlayerID(77003)
	require.NoError(t, err)
	require.NotZero(t, token.ID)

	activeCount, softDeleted := countUserTokensUnscoped(t, 77003)
	require.Equal(t, int64(1), activeCount)
	require.GreaterOrEqual(t, softDeleted, int64(2))
}
