package pubsub

// gRPC/Redis stream names. Player-facing streams use Event.receivers for filtering; TopicRoomSettlement is lobby-internal only.
const (
	TopicRoom           = "room_events"
	TopicLobby          = "lobby_events"
	TopicRoomSettlement = "room_settlement"
)
