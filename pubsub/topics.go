package pubsub

// gRPC/Redis stream names. Player-facing streams use Event.receivers for filtering; TopicRoomSettlementPVP is lobby-internal only.
const (
	TopicRoom                     = "room_events"
	TopicLobby                    = "lobby_events"
	TopicRoomSettlementPVP        = "room_settlement_PVP"
	TopicRoomSettlementTournament = "room_settlement_tournament"
)
