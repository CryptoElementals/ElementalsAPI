package evmrpc

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultHTTPClientConnectionPool(t *testing.T) {
	transport, ok := defaultHTTPClient.Transport.(*http.Transport)
	require.True(t, ok)
	require.Equal(t, rpcMaxIdleConns, transport.MaxIdleConns)
	require.Equal(t, rpcMaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
	require.Equal(t, rpcIdleConnTimeout, transport.IdleConnTimeout)
}

func TestParseHexUint64(t *testing.T) {
	n, err := parseHexUint64("0x61")
	require.NoError(t, err)
	require.Equal(t, uint64(97), n)
}

func TestParseTransactions(t *testing.T) {
	txs := []interface{}{
		map[string]interface{}{
			"hash":  "0xabc",
			"from":  "0x111",
			"to":    "0x222",
			"input": "0x",
		},
	}
	parsed, err := ParseTransactions(txs)
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	require.Equal(t, "0xabc", parsed[0].Hash)
	require.Equal(t, uint32(0), parsed[0].TxIndex)
}
