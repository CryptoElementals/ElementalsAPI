package api

import (
	"testing"

	cmnErrors "github.com/CryptoElementals/common/errors"
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
