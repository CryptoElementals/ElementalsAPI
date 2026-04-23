package pubsub

import (
	"fmt"

	"github.com/CryptoElementals/common/rpc/proto"
)

var eventTypeNumber = map[proto.EventType]int{
	proto.EventType_TYPE_GAME_CREATED:             1,
	proto.EventType_TYPE_PART_CONFIRMED:           2,
	proto.EventType_TYPE_ROUND_READY:              3,
	proto.EventType_TYPE_TURN_READY:               4,
	proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:     5,
	proto.EventType_TYPE_TURN_COMPLETE:            6,
	proto.EventType_TYPE_GAME_SETTLEMENT_RESULT:   7,
	proto.EventType_TYPE_NOT_MATCHABLE:            8,
	proto.EventType_TYPE_MATCHED:                  9,
	proto.EventType_TYPE_MATCH_CANCELED:           10,
	proto.EventType_TYPE_TOURNAMENT_MATCH_OUTCOME: 11,
	proto.EventType_TYPE_TOURNAMENT_ROSTER_UPDATE: 12,
}

// gamePhaseSyncMessageTypeIndex maps GamePhase.{TurnStatus, PlayerTurnStatus} to the same semantic
// mm slot as the corresponding TYPE_* events in eventTypeNumber (TurnReady=4, CommitmentsOnChain=5,
// TurnComplete=6, etc.), so sync message IDs line up with dedup keys for that game stream.
func gamePhaseSyncMessageTypeIndex(ts proto.TurnStatus) int {
	switch ts {
	case proto.TurnStatus_TURN_ROUND_COMPLETED, proto.TurnStatus_TURN_COMPLETED:
		return eventTypeNumber[proto.EventType_TYPE_TURN_COMPLETE]
	case proto.TurnStatus_TURN_WAITTING_COMMITMENTS:
		return eventTypeNumber[proto.EventType_TYPE_TURN_READY]
	case proto.TurnStatus_TURN_WAITTING_CARDS:
		return eventTypeNumber[proto.EventType_TYPE_COMMITMENTS_ON_CHAIN]
	case proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION, proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN:
		return eventTypeNumber[proto.EventType_TYPE_PART_CONFIRMED]
	default:
		return 0
	}
}

// BuildEventMessageID returns a dedup key for room/lobby facing events.
// TYPE_GAME_PHASE_SYNC appends the caller's PlayerTurnStatus so distinct sync snapshots do not collide.
func BuildEventMessageID(evt *proto.Event) string {
	if evt == nil {
		return ""
	}

	primaryID := int64(0)
	gameID := int64(0)
	rr := uint32(0)
	tt := uint32(0)
	mm := eventTypeNumber[evt.GetType()]

	switch evt.GetType() {
	case proto.EventType_TYPE_MATCHED:
		primaryID = evt.GetGameMatched().GetMatchId()
	case proto.EventType_TYPE_MATCH_CANCELED:
		primaryID = evt.GetMatchCanceled().GetMatchId()
	case proto.EventType_TYPE_ROUND_READY:
		gameID = evt.GetRoundReady().GetGameId()
		primaryID = gameID
		rr = evt.GetRoundReady().GetRoundNum()
	case proto.EventType_TYPE_TURN_READY:
		gameID = evt.GetTurnReady().GetGameId()
		primaryID = gameID
		rr = evt.GetTurnReady().GetRoundNum()
		tt = evt.GetTurnReady().GetTurnNum()
	case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
		gameID = evt.GetCommitmentsOnChain().GetGameId()
		primaryID = gameID
		rr = evt.GetCommitmentsOnChain().GetRoundNum()
		tt = evt.GetCommitmentsOnChain().GetTurnNum()
	case proto.EventType_TYPE_TURN_COMPLETE:
		gameID = evt.GetTurnCompleted().GetGameId()
		primaryID = gameID
		rr = evt.GetTurnCompleted().GetRoundNum()
		tt = evt.GetTurnCompleted().GetTurnNum()
	case proto.EventType_TYPE_GAME_CREATED:
		gameID = evt.GetGameReady().GetGameId()
		primaryID = gameID
	case proto.EventType_TYPE_GAME_PHASE_SYNC:
		gp := evt.GetGamePhase()
		gameID = gp.GetGameID()
		primaryID = gameID
		rr = gp.GetRoundNumber()
		tt = gp.GetTurnNumber()
		mm = gamePhaseSyncMessageTypeIndex(gp.GetTurnStatus())
	case proto.EventType_TYPE_GAME_SETTLEMENT_RESULT:
		gameID = evt.GetGameSettlementResult().GetGameId()
		primaryID = gameID
		rr = 99
		tt = 99
	case proto.EventType_TYPE_NOT_MATCHABLE:
		gameID = evt.GetNotMatchable().GetLastGameId()
		primaryID = gameID
		rr = 99
		tt = 99
	case proto.EventType_TYPE_TOURNAMENT_MATCH_OUTCOME:
		o := evt.GetTournamentMatchOutcome()
		if o != nil {
			gameID = o.GetGameId()
			primaryID = gameID
			rr = o.GetRoundNo()
			tt = o.GetMatchNo()
		}
	case proto.EventType_TYPE_TOURNAMENT_ROSTER_UPDATE:
		tid := evt.GetTournamentRosterUpdate().GetTournamentID()
		var h uint64 = 5381
		for _, c := range tid {
			h = (h << 5) + h + uint64(c)
		}
		primaryID = int64(h & 0x7fffffffffffffff)
	}

	isSync := 0
	if evt.GetType() == proto.EventType_TYPE_GAME_PHASE_SYNC {
		isSync = 1
	}

	if evt.GetType() == proto.EventType_TYPE_GAME_PHASE_SYNC {
		gp := evt.GetGamePhase()
		pts := 0
		if gp != nil {
			pts = int(gp.GetPlayerTurnStatus())
		}
		return fmt.Sprintf("%d%02d%02d%02d%02d%d", primaryID, rr, tt, mm, pts, isSync)
	}
	return fmt.Sprintf("%d%02d%02d%02d%d", primaryID, rr, tt, mm, isSync)
}
