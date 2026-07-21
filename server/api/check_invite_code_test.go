package api

import (
	"testing"

	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/server/invite"
	"github.com/stretchr/testify/require"
)

func TestCheckInviteCodeAPI(t *testing.T) {
	invite.SetUpTestDB(t)
	invite.SetUpDualLobbyClients(t)

	params := map[string]interface{}{
		"Action":      CHECK_INVITE_CODE_LABEL,
		"RequestUUID": "check-1",
		"InviteCode":  "MISSING1",
	}
	task, err := NewCheckInviteCodeTask(&params)
	require.NoError(t, err)
	resp, err := task.Run(setupGinContext())
	require.NoError(t, err)
	typed, ok := resp.(*CheckInviteCodeResponse)
	require.True(t, ok)
	require.Equal(t, int(cmnErrors.WInviteInvalidCode), typed.WarnCode)
}

func TestCheckInviteCodeAPIReturnsPersistedMax(t *testing.T) {
	invite.SetUpTestDB(t)
	invite.SetUpDualLobbyClients(t)

	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: 9101, Address: "0xinv", Name: "inv", ServerType: dao.ServerTypeNormal,
	}).Error)
	require.NoError(t, db.Get().Create(&dao.UserInviteCode{
		PlayerID: 9101, InviteCode: "PERSIST1", InviteCount: 1, MaxInviteCount: 8,
	}).Error)

	params := map[string]interface{}{
		"Action": CHECK_INVITE_CODE_LABEL, "RequestUUID": "check-persist",
		"InviteCode": "PERSIST1",
	}
	task, err := NewCheckInviteCodeTask(&params)
	require.NoError(t, err)
	resp, err := task.Run(setupGinContext())
	require.NoError(t, err)
	typed, ok := resp.(*CheckInviteCodeResponse)
	require.True(t, ok)
	require.Equal(t, cmnErrors.WarnCodeOK, cmnErrors.WarnCode(typed.WarnCode))
	require.Equal(t, 8, typed.MaxInviteesPerCode)
	require.Equal(t, 7, typed.InviteSlotsRemaining)
	require.NotNil(t, typed.Inviter)
	require.Equal(t, "9101", typed.Inviter.PlayerID)
	require.Equal(t, "inv", typed.Inviter.Username)
}

func TestGetUserProfileReturnsPersistedMaxInvitees(t *testing.T) {
	invite.SetUpTestDB(t)
	invite.SetUpDualLobbyClients(t)

	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: 9102, Address: "0xself", Name: "self", ServerType: dao.ServerTypeNormal,
	}).Error)
	require.NoError(t, db.Get().Create(&dao.UserInviteCode{
		PlayerID: 9102, InviteCode: "SELF01", InviteCount: 2, MaxInviteCount: 6,
	}).Error)

	params := map[string]interface{}{
		"Action": GET_USER_PROFILE_LABEL, "RequestUUID": "profile-max",
		"PlayerID": "9102",
	}
	task, err := NewGetUserProfileTask(&params)
	require.NoError(t, err)
	resp, err := task.Run(setupGinContext())
	require.NoError(t, err)
	typed, ok := resp.(*GetUserProfileResponse)
	require.True(t, ok)
	require.Equal(t, 6, typed.UserInfo.MaxInviteesPerCode)
	require.Equal(t, 2, typed.UserInfo.InvitedCount)
}
