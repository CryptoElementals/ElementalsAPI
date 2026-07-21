package api

import (
	"testing"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/server/invite"
	dao "github.com/CryptoElementals/common/models"
	"github.com/stretchr/testify/require"
)

func TestClaimInviteeRewardDualWrite(t *testing.T) {
	invite.SetUpTestDB(t)
	invite.SetUpDualLobbyClients(t)

	inviterID := int64(9101)
	inviteeID := int64(9102)
	require.NoError(t, db.Get().Create(&dao.UserProfile{
		PlayerID: inviteeID, Address: "0xinvitee", Name: "invitee", ServerType: dao.ServerTypeTrial,
	}).Error)
	relation := dao.UserInviteRelation{InviterPlayerID: inviterID, InviteePlayerID: inviteeID, InviteCode: "CODE9101"}
	require.NoError(t, db.Get().Create(&relation).Error)
	require.NoError(t, db.Get().Create(&dao.UserInviteeReward{
		RelationID: relation.ID, InviteePlayerID: inviteeID, Point: 50,
	}).Error)
	require.NoError(t, db.Get().Create(&dao.UserToken{PlayerId: inviteeID, Points: 0, TokenAmount: 0}).Error)

	params := map[string]interface{}{
		"Action":      CLAIM_INVITEE_REWARD_LABEL,
		"RequestUUID": "claim-invitee-1",
		"PlayerID":    "9102",
	}
	task, err := NewClaimInviteeRewardTask(&params)
	require.NoError(t, err)
	resp, err := task.Run(setupGinContext())
	require.NoError(t, err)
	typed, ok := resp.(*ClaimInviteeRewardResponse)
	require.True(t, ok)
	require.Equal(t, int32(50), typed.Point)

	var token dao.UserToken
	require.NoError(t, db.Get().Where("player_id = ?", inviteeID).First(&token).Error)
	// trial + normal dual-write hits the same sqlite row in tests
	require.GreaterOrEqual(t, token.Points, int32(50))
}
