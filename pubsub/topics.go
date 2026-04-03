package pubsub

// Shared Redis stream / gRPC topic names. All room events use TopicRoom; all lobby queue events use TopicLobby.
// Subscribers filter by [proto.Event.Receivers].
const (
	TopicRoom = "room_events"
	// TopicLobby is the shared lobby→client stream (matchmaking, settlement to players, etc.).
	TopicLobby = "lobby_events"
	// TopicRoomSettlement is room→lobby only: game finished (see GameCompletedNotice). Not subscribed by player clients.
	TopicRoomSettlement = "room_settlement"
)
