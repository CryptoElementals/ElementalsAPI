package api

import (
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/server/invite"
	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
)

func TestGetMyInviter(t *testing.T) {
	invite.SetUpTestDB(t)
	invite.SetUpDualLobbyClients(t)

	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: 9301, Address: "0xinv", Name: "inv", ServerType: dao.ServerTypeNormal,
	}).Error)
	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: 9302, Address: "0xee", Name: "ee", ServerType: dao.ServerTypeTrial,
	}).Error)
	require.NoError(t, db.Get().Create(&dao.UserInviteRelation{
		InviterPlayerID: 9301, InviteePlayerID: 9302, InviteCode: "CODE9301",
	}).Error)

	params := map[string]interface{}{
		"Action": GET_MY_INVITER_LABEL, "RequestUUID": "my-inviter-1", "PlayerID": "9302",
	}
	task, err := NewGetMyInviterTask(&params)
	require.NoError(t, err)
	resp, err := task.Run(setupGinContext())
	require.NoError(t, err)
	typed, ok := resp.(*GetMyInviterResponse)
	require.True(t, ok)
	require.True(t, typed.HasInviter)
	require.Equal(t, "9301", typed.PlayerID)
	require.Equal(t, "inv", typed.Username)
}

func TestListInviterRewardsFilter(t *testing.T) {
	invite.SetUpTestDB(t)
	invite.SetUpDualLobbyClients(t)

	relation := dao.UserInviteRelation{InviterPlayerID: 9401, InviteePlayerID: 9402, InviteCode: "C9401"}
	require.NoError(t, db.Get().Create(&relation).Error)
	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: 9401, Name: "inv", ServerType: dao.ServerTypeNormal,
	}).Error)
	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: 9402, Name: "ee", ServerType: dao.ServerTypeTrial,
	}).Error)
	require.NoError(t, db.Get().Create(&dao.UserInviterReward{
		RelationID: relation.ID, InviterPlayerID: 9401, InviteePlayerID: 9402,
		MilestoneLevel: 3, Point: 25,
	}).Error)

	params := map[string]interface{}{
		"Action": LIST_INVITER_REWARDS_LABEL, "RequestUUID": "list-rewards-1",
		"PlayerID": "9401", "InviteePlayerID": "9402",
	}
	task, err := NewListInviterRewardsTask(&params)
	require.NoError(t, err)
	resp, err := task.Run(setupGinContext())
	require.NoError(t, err)
	typed, ok := resp.(*ListInviterRewardsResponse)
	require.True(t, ok)
	require.Len(t, typed.Rewards, 1)
	require.Equal(t, 3, typed.Rewards[0].MilestoneLevel)
}
