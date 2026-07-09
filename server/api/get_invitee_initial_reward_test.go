package api

import (
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/server/invite"
	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
)

func TestGetInviteeInitialReward(t *testing.T) {
	invite.SetUpTestDB(t)

	inviteeID := int64(9501)
	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: inviteeID, Address: "0x9501", Name: "invitee", ServerType: dao.ServerTypeTrial,
	}).Error)
	relation := dao.UserInviteRelation{InviterPlayerID: 9500, InviteePlayerID: inviteeID, InviteCode: "CODE9500"}
	require.NoError(t, db.Get().Create(&relation).Error)
	require.NoError(t, db.Get().Create(&dao.UserInviteeReward{
		RelationID: relation.ID, InviteePlayerID: inviteeID, Point: 50,
	}).Error)

	params := map[string]interface{}{
		"Action":      GET_INVITEE_INITIAL_REWARD_LABEL,
		"RequestUUID": "get-invitee-reward-1",
		"PlayerID":    "9501",
	}
	task, err := NewGetInviteeInitialRewardTask(&params)
	require.NoError(t, err)
	resp, err := task.Run(setupGinContext())
	require.NoError(t, err)
	typed, ok := resp.(*GetInviteeInitialRewardResponse)
	require.True(t, ok)
	require.True(t, typed.HasReward)
	require.False(t, typed.Claimed)
	require.Equal(t, int32(50), typed.Point)
}

func TestGetInviteeInitialRewardNone(t *testing.T) {
	invite.SetUpTestDB(t)

	params := map[string]interface{}{
		"Action":      GET_INVITEE_INITIAL_REWARD_LABEL,
		"RequestUUID": "get-invitee-reward-2",
		"PlayerID":    "9599",
	}
	task, err := NewGetInviteeInitialRewardTask(&params)
	require.NoError(t, err)
	resp, err := task.Run(setupGinContext())
	require.NoError(t, err)
	typed, ok := resp.(*GetInviteeInitialRewardResponse)
	require.True(t, ok)
	require.False(t, typed.HasReward)
	require.False(t, typed.Claimed)
	require.Equal(t, int32(0), typed.Point)
}
