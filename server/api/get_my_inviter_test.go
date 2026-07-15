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
	require.Len(t, typed.Rewards, 4)
	require.Equal(t, 3, typed.Rewards[0].MilestoneLevel)
	require.Equal(t, "claimable", typed.Rewards[0].InviterRewardStatus)
	require.NotZero(t, typed.Rewards[0].RewardID)
	require.Equal(t, 6, typed.Rewards[1].MilestoneLevel)
	require.Equal(t, "locked", typed.Rewards[1].InviterRewardStatus)
	require.Zero(t, typed.Rewards[1].RewardID)
	require.Equal(t, 9, typed.Rewards[2].MilestoneLevel)
	require.Equal(t, "locked", typed.Rewards[2].InviterRewardStatus)
	require.Equal(t, 12, typed.Rewards[3].MilestoneLevel)
	require.Equal(t, "locked", typed.Rewards[3].InviterRewardStatus)
}

func TestListInviterRewardsAllLockedWhenNoRows(t *testing.T) {
	invite.SetUpTestDB(t)
	invite.SetUpDualLobbyClients(t)

	require.NoError(t, db.Get().Create(&dao.UserInviteRelation{
		InviterPlayerID: 9411, InviteePlayerID: 9412, InviteCode: "C9411",
	}).Error)
	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: 9411, Name: "inv2", ServerType: dao.ServerTypeNormal,
	}).Error)
	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: 9412, Name: "ee2", ServerType: dao.ServerTypeTrial,
	}).Error)

	params := map[string]interface{}{
		"Action": LIST_INVITER_REWARDS_LABEL, "RequestUUID": "list-rewards-locked",
		"PlayerID": "9411", "InviteePlayerID": "9412",
	}
	task, err := NewListInviterRewardsTask(&params)
	require.NoError(t, err)
	resp, err := task.Run(setupGinContext())
	require.NoError(t, err)
	typed, ok := resp.(*ListInviterRewardsResponse)
	require.True(t, ok)
	require.Len(t, typed.Rewards, 4)
	for i, wantLevel := range []int{3, 6, 9, 12} {
		require.Equal(t, wantLevel, typed.Rewards[i].MilestoneLevel)
		require.Equal(t, "locked", typed.Rewards[i].InviterRewardStatus)
		require.Zero(t, typed.Rewards[i].RewardID)
		require.Equal(t, 0, typed.Rewards[i].InviteeLevel)
	}
	require.Equal(t, int32(25), typed.Rewards[0].Point)
	require.Equal(t, int32(100), typed.Rewards[1].Point)
	require.Equal(t, int32(400), typed.Rewards[2].Point)
	require.Equal(t, int32(1300), typed.Rewards[3].Point)
}
