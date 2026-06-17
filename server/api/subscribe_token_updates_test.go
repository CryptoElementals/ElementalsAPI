package api

import (
	"testing"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/sse"
	"github.com/stretchr/testify/require"
)

func TestSubscribeTokenUpdatesConvertTokenEventToSSE(t *testing.T) {
	task := &SubscribeTokenUpdatesTask{}
	msg := &proto.Message{
		Event: &proto.Event{
			Type:      proto.EventType_TYPE_TOKEN_UPDATED,
			MessageId: "mid-1",
			Event: &proto.Event_TokenUpdated{
				TokenUpdated: &proto.TokenUpdated{
					PlayerId:    42,
					TokenDelta:  1000,
					Tokens:      5000,
					Source:      "chain_deposit",
				},
			},
		},
	}

	ev := task.convertTokenEventToSSE(msg, "req-uuid")
	require.Equal(t, sse.EventTypeDataChange, ev.Type)
	data, ok := ev.Data.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "tokenUpdated", data["EventType"])
	require.Equal(t, "mid-1", data["MessageID"])
	payload, ok := data["Message"].(*proto.TokenUpdated)
	require.True(t, ok)
	require.Equal(t, int64(42), payload.GetPlayerId())
	require.Equal(t, int32(1000), payload.GetTokenDelta())
}
