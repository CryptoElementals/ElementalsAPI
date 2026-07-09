package db

import (
	"testing"

	cmnErrors "github.com/CryptoElementals/common/errors"
	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
)

func setupTestDBForInvite(t *testing.T) {
	t.Helper()
	require.NoError(t, Init(&Config{Development: true}))
	Get().Logger = Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, MigrateMemDb())
}

func createNormalInviter(t *testing.T, playerID int64, code string) {
	t.Helper()
	require.NoError(t, Get().Create(&dao.UserProfile{
		PlayerID:   playerID,
		Address:    "0xinviter",
		Name:       "inviter",
		ServerType: dao.ServerTypeNormal,
	}).Error)
	require.NoError(t, Get().Create(&dao.UserInviteCode{
		PlayerID:    playerID,
		InviteCode:  code,
		InviteCount: 0,
	}).Error)
}

func TestApplyInviteReferralSuccess(t *testing.T) {
	setupTestDBForInvite(t)
	createNormalInviter(t, 9001, "INVITE01")
	require.NoError(t, Get().Create(&dao.UserProfile{
		PlayerID:   9002,
		Address:    "0xinvitee",
		Name:       "invitee",
		ServerType: dao.ServerTypeTrial,
	}).Error)

	outcome, err := ApplyInviteReferral(9002, "INVITE01", 50, 3)
	require.NoError(t, err)
	require.Equal(t, cmnErrors.WarnCodeOK, outcome.WarnCode)

	var count int64
	require.NoError(t, Get().Model(&dao.UserInviteRelation{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
	require.NoError(t, Get().Model(&dao.UserInviteeReward{}).Count(&count).Error)
	require.Equal(t, int64(1), count)

	var inviter dao.UserInviteCode
	require.NoError(t, Get().Where("player_id = ?", 9001).First(&inviter).Error)
	require.Equal(t, 1, inviter.InviteCount)
}

func TestApplyInviteReferralIdempotent(t *testing.T) {
	setupTestDBForInvite(t)
	createNormalInviter(t, 9011, "INVITE02")
	require.NoError(t, Get().Create(&dao.UserProfile{PlayerID: 9012, Address: "0xa", Name: "a", ServerType: dao.ServerTypeTrial}).Error)

	outcome, err := ApplyInviteReferral(9012, "INVITE02", 50, 3)
	require.NoError(t, err)
	require.Equal(t, cmnErrors.WarnCodeOK, outcome.WarnCode)

	outcome, err = ApplyInviteReferral(9012, "INVITE02", 50, 3)
	require.NoError(t, err)
	require.Equal(t, cmnErrors.WarnCodeOK, outcome.WarnCode)

	var count int64
	require.NoError(t, Get().Model(&dao.UserInviteRelation{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestApplyInviteReferralLimitAndInvalid(t *testing.T) {
	setupTestDBForInvite(t)
	createNormalInviter(t, 9021, "FULLCODE")
	require.NoError(t, Get().Model(&dao.UserInviteCode{}).Where("player_id = ?", 9021).Update("invite_count", 3).Error)

	outcome, err := ApplyInviteReferral(9022, "FULLCODE", 50, 3)
	require.NoError(t, err)
	require.Equal(t, cmnErrors.WInviteLimitReached, outcome.WarnCode)

	outcome, err = ApplyInviteReferral(9023, "NOPE1234", 50, 3)
	require.NoError(t, err)
	require.Equal(t, cmnErrors.WInviteInvalidCode, outcome.WarnCode)

	require.NoError(t, Get().Create(&dao.UserProfile{
		PlayerID: 9024, Address: "0xtrial", Name: "trial", ServerType: dao.ServerTypeTrial,
	}).Error)
	require.NoError(t, Get().Create(&dao.UserInviteCode{PlayerID: 9024, InviteCode: "TRIALCOD", InviteCount: 0}).Error)
	outcome, err = ApplyInviteReferral(9025, "TRIALCOD", 50, 3)
	require.NoError(t, err)
	require.Equal(t, cmnErrors.WInviteInvalidCode, outcome.WarnCode)
}

func TestProcessInviteeLevelMilestonesTx(t *testing.T) {
	setupTestDBForInvite(t)
	createNormalInviter(t, 9031, "MILECODE")
	relation := dao.UserInviteRelation{InviterPlayerID: 9031, InviteePlayerID: 9032, InviteCode: "MILECODE"}
	require.NoError(t, Get().Create(&relation).Error)

	err := Get().Transaction(func(tx *gorm.DB) error {
		return ProcessInviteeLevelMilestonesTx(tx, 9032, 0, 5000)
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, Get().Model(&dao.UserInviterReward{}).Where("relation_id = ?", relation.ID).Count(&count).Error)
	require.Equal(t, int64(0), count)

	err = Get().Transaction(func(tx *gorm.DB) error {
		return ProcessInviteeLevelMilestonesTx(tx, 9032, 0, 20500)
	})
	require.NoError(t, err)
	require.NoError(t, Get().Model(&dao.UserInviterReward{}).Where("relation_id = ? AND milestone_level = 3", relation.ID).Count(&count).Error)
	require.Equal(t, int64(1), count)

	err = Get().Transaction(func(tx *gorm.DB) error {
		return ProcessInviteeLevelMilestonesTx(tx, 9032, 20300, 20500)
	})
	require.NoError(t, err)
	require.NoError(t, Get().Model(&dao.UserInviterReward{}).Where("relation_id = ? AND milestone_level = 3", relation.ID).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestEnsureInviteCodeSkipsTrial(t *testing.T) {
	setupTestDBForInvite(t)
	require.NoError(t, Get().Create(&dao.UserProfile{
		PlayerID: 9041, Address: "0xt", Name: "t", ServerType: dao.ServerTypeTrial,
	}).Error)
	require.NoError(t, EnsureInviteCode(9041))
	var count int64
	require.NoError(t, Get().Model(&dao.UserInviteCode{}).Where("player_id = ?", 9041).Count(&count).Error)
	require.Equal(t, int64(0), count)

	require.NoError(t, Get().Create(&dao.UserProfile{
		PlayerID: 9042, Address: "0xn", Name: "n", ServerType: dao.ServerTypeNormal,
	}).Error)
	require.NoError(t, EnsureInviteCode(9042))
	require.NoError(t, Get().Model(&dao.UserInviteCode{}).Where("player_id = ?", 9042).Count(&count).Error)
	require.Equal(t, int64(1), count)
}
