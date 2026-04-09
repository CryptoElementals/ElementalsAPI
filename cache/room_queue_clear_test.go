package cache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClearRoomServerQueueAndTokenKeys_MemCache(t *testing.T) {
	c := NewMemCache()
	qc := WithPrefix(roomQueueInfoPrefix, c)
	tc := WithPrefix(roomLockedTokenPrefix, c)
	require.NoError(t, qc.Set("p1", "v", 0))
	require.NoError(t, tc.Set("p2", "v", 0))

	q, tok, err := ClearRoomServerQueueAndTokenKeys(c)
	require.NoError(t, err)
	require.Equal(t, 1, q)
	require.Equal(t, 1, tok)

	_, err = qc.Get("p1")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = tc.Get("p2")
	require.ErrorIs(t, err, ErrNotFound)
}
