package pubsub

// gRPC/Redis stream names. Player-facing streams use Event.receivers for filtering; TopicRoomSettlementPVP is lobby-internal only.
const (
	TopicRoom  = "room_events"
	TopicLobby = "lobby_events"
	// TopicTournamentRoster is a broadcast stream (empty Event.receivers): anyone on subscribe_game_info refreshes tournament snapshot.
	TopicTournamentRoster         = "tournament_roster_events"
	// TopicToken is wallet balance updates from ledger-server; SubscribeTokenUpdates filters by player_id.
	TopicToken                    = "token_events"
	TopicRoomSettlementPVP        = "room_settlement_PVP"
	TopicRoomSettlementTournament = "room_settlement_tournament"
)
