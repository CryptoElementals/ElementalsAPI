package invite

import (
	"context"
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
)

// CreditUserPointsAllEnvironments credits points in every configured lobby environment.
func CreditUserPointsAllEnvironments(playerID int64, delta int32, reason string) error {
	return client.ForEachConfiguredLobby(func(_ string, cl proto.LobbyServiceClient) error {
		_, err := cl.CreditUserPoints(context.Background(), &proto.CreditUserPointsRequest{
			PlayerID: playerID,
			Delta:    delta,
			Reason:   reason,
		})
		return err
	})
}

// CreditUserPointsForServerType credits points in the lobby for the given server type.
func CreditUserPointsForServerType(serverType string, playerID int64, delta int32, reason string) error {
	cl := client.LobbyClientForType(serverType)
	if cl == nil {
		return cmnErrors.ActionError("gRPC lobby client not initialized")
	}
	_, err := cl.CreditUserPoints(context.Background(), &proto.CreditUserPointsRequest{
		PlayerID: playerID,
		Delta:    delta,
		Reason:   reason,
	})
	return err
}

// MaxInviteesPerCode returns the maximum successful invitees per invite code.
func MaxInviteesPerCode() int {
	if config.GConf.MaxInviteesPerCode > 0 {
		return config.GConf.MaxInviteesPerCode
	}
	return 3
}

// InviteeInitialPoints returns configured invitee initial reward points.
func InviteeInitialPoints() int {
	if config.GConf.InviteeInitialPoints > 0 {
		return config.GConf.InviteeInitialPoints
	}
	return 50
}

// PlayerPointsFromLobby reads points for a player from the lobby matching server type.
func PlayerPointsFromLobby(serverType string, playerID int64) (int, error) {
	cl := client.LobbyClientForType(serverType)
	if cl == nil {
		return 0, cmnErrors.ActionError("gRPC lobby client not initialized")
	}
	resp, err := cl.GetPlayerToken(context.Background(), &proto.GetPlayerTokenRequest{Id: playerID})
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, nil
	}
	return int(resp.GetPoints()), nil
}

// EnsureInviterCodeOnLogin creates the user's own invite code for normal accounts.
func EnsureInviterCodeOnLogin(playerID int64, serverType string) error {
	if dao.NormalizeServerType(serverType) != dao.ServerTypeNormal {
		return nil
	}
	return db.EnsureInviteCode(playerID, MaxInviteesPerCode())
}

// TryApplyOnLogin applies invite binding for new users when a code was provided.
func TryApplyOnLogin(isNewUser bool, inviteePlayerID int64, inviteCode string) (warnCode int, warnMessage string, err error) {
	inviteCode = strings.TrimSpace(inviteCode)
	if inviteCode == "" {
		return int(cmnErrors.WarnCodeOK), "", nil
	}
	if !isNewUser {
		entry := cmnErrors.WarnInviteExistingUser
		return int(entry.Code), entry.Message, nil
	}
	outcome, err := db.ApplyInviteReferral(
		inviteePlayerID,
		inviteCode,
		int32(InviteeInitialPoints()),
	)
	if err != nil {
		return 0, "", err
	}
	return int(outcome.WarnCode), outcome.Message, nil
}
