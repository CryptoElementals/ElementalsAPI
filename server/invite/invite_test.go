package invite

import (
	"testing"

	cmnErrors "github.com/CryptoElementals/common/errors"
	"github.com/stretchr/testify/require"
)

func TestTryApplyOnLogin(t *testing.T) {
	SetUpTestDB(t)
	SetUpDualLobbyClients(t)

	code, msg, err := TryApplyOnLogin(false, 1, "ABC")
	require.NoError(t, err)
	require.Equal(t, int(cmnErrors.WInviteExistingUser), code)
	require.NotEmpty(t, msg)

	code, msg, err = TryApplyOnLogin(true, 1, "")
	require.NoError(t, err)
	require.Equal(t, 0, code)
	require.Empty(t, msg)
}
