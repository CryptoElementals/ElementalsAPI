package stream

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAutoClaimReply(t *testing.T) {
	payload := []byte("hello")
	b64 := base64.StdEncoding.EncodeToString(payload)
	// XAUTOCLAIM uses the same stream-entry layout as XRANGE; redigo StringMap expects []byte values.
	entry := []interface{}{
		[]byte("12344-0"),
		[]interface{}{
			[]byte("topic"), []byte("room_settlement_PVP"),
			[]byte("payload"), []byte(b64),
			[]byte("ts"), []byte("99"),
		},
	}
	reply := []interface{}{
		[]byte("12345-0"),
		[]interface{}{entry},
	}
	entries, next, err := parseAutoClaimReply(reply)
	require.NoError(t, err)
	require.Equal(t, "12345-0", next)
	require.Len(t, entries, 1)
	require.Equal(t, "12344-0", entries[0].ID)
	require.Equal(t, payload, entries[0].Payload)
	require.Equal(t, int64(99), entries[0].Timestamp)
}

func TestParseAutoClaimReply_EmptyEntries(t *testing.T) {
	reply := []interface{}{
		"0-0",
		[]interface{}{},
	}
	entries, next, err := parseAutoClaimReply(reply)
	require.NoError(t, err)
	require.Equal(t, "0-0", next)
	require.Empty(t, entries)
}

func TestParseAutoClaimReply_Nil(t *testing.T) {
	entries, next, err := parseAutoClaimReply(nil)
	require.NoError(t, err)
	require.Equal(t, "0-0", next)
	require.Nil(t, entries)
}

func TestParseAutoClaimReply_NilEntriesSlice(t *testing.T) {
	reply := []interface{}{
		"99-0",
		nil,
	}
	entries, next, err := parseAutoClaimReply(reply)
	require.NoError(t, err)
	require.Equal(t, "99-0", next)
	require.Nil(t, entries)
}

func TestParseAutoClaimReply_Redis7ExtraFields(t *testing.T) {
	reply := []interface{}{
		"0-0",
		[]interface{}{},
		[]interface{}{"deleted-id-1"},
	}
	entries, next, err := parseAutoClaimReply(reply)
	require.NoError(t, err)
	require.Equal(t, "0-0", next)
	require.Empty(t, entries)
}

func TestParseAutoClaimReply_ShortReply(t *testing.T) {
	_, _, err := parseAutoClaimReply([]interface{}{"only-one"})
	require.Error(t, err)
}

func TestParseAutoClaimReply_InvalidNested(t *testing.T) {
	reply := []interface{}{
		"0-0",
		"not-an-array",
	}
	_, _, err := parseAutoClaimReply(reply)
	require.Error(t, err)
}
