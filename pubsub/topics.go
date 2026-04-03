package pubsub

// Shared Redis stream / gRPC topic names. All room events use TopicRoom; all lobby queue events use TopicLobby.
// Subscribers filter by [proto.Event.Receivers].
const (
	TopicRoom  = "room_events"
	TopicLobby = "lobby_events"
)
