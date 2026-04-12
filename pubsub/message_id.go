package pubsub

import (
	"fmt"

	"github.com/CryptoElementals/common/rpc/proto"
)

var eventTypeNumber = map[proto.EventType]int{
	proto.EventType_TYPE_GAME_CREATED:           1,
	proto.EventType_TYPE_PART_CONFIRMED:         2,
	proto.EventType_TYPE_ROUND_READY:            3,
	proto.EventType_TYPE_TURN_READY:             4,
	proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:   5,
	proto.EventType_TYPE_TURN_COMPLETE:          6,
	proto.EventType_TYPE_GAME_SETTLEMENT_RESULT: 7,
	proto.EventType_TYPE_NOT_MATCHABLE:          8,
	proto.EventType_TYPE_MATCHED:                9,
	proto.EventType_TYPE_MATCH_CANCELED:         10,
	proto.EventType_TYPE_TOURNAMENT_MATCH_OUTCOME: 11,
}

// BuildEventMessageID returns rr-tt-mm-i for room/lobby facing events.
func BuildEventMessageID(evt *proto.Event) string {
	if evt == nil {
		return ""
	}

	primaryID := int64(0)
	gameID := int64(0)
	rr := uint32(0)
	tt := uint32(0)

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
		gameID = evt.GetGamePhase().GetGameID()
		primaryID = gameID
		rr = evt.GetGamePhase().GetRoundNumber()
		tt = evt.GetGamePhase().GetTurnNumber()
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
	}

	mm := eventTypeNumber[evt.GetType()]
	isSync := 0
	if evt.GetType() == proto.EventType_TYPE_GAME_PHASE_SYNC {
		isSync = 1
	}

	return fmt.Sprintf("%d%02d%02d%02d%d", primaryID, rr, tt, mm, isSync)
}
