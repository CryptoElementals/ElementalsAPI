package queue

import (
	"context"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

// forEachSettlementParticipant runs fn for each participant (player id + room temp address) from player_result_infos.
func forEachSettlementParticipant(gr *dao.GameResult, fn func(playerID int64, temporaryAddress string)) {
	if gr == nil {
		return
	}
	for _, pri := range gr.PlayerResultInfos {
		if pri == nil {
			continue
		}
		fn(pri.PlayerId, pri.TemporaryAddress)
	}
}

func settlementPlayerIDs(gr *dao.GameResult) []int64 {
	if gr == nil {
		return nil
	}
	ids := make([]int64, 0, len(gr.PlayerResultInfos))
	for _, pri := range gr.PlayerResultInfos {
		if pri == nil {
			continue
		}
		ids = append(ids, pri.PlayerId)
	}
	return ids
}

// availableTokens returns recorded balance minus locked rows (same gate as joining the matchmaking queue).
func availableTokens(ut *dao.UserToken) int {
	var locked int32
	for _, row := range ut.LockedTokens {
		locked += row.TokenAmount
	}
	return int(ut.TokenAmount) - int(locked)
}

// anyHumanPlayerBelowQueueThreshold reports whether any non-bot player cannot afford the queue lock after settlement.
func (q *Queue) anyHumanPlayerBelowQueueThreshold(gr *dao.GameResult, bots Set[types.PlayerAddress]) bool {
	var below bool
	forEachSettlementParticipant(gr, func(playerID int64, temporaryAddress string) {
		if below {
			return
		}
		addr := types.NewPlayerAddress(playerID, temporaryAddress)
		if bots.Contains(*addr) {
			return
		}
		ut, err := db.GetPlayerToken(context.Background(), playerID)
		if err != nil {
			log.Errorw("failed to get player token after settlement", "player_id", playerID, "err", err)
			return
		}
		avail := availableTokens(ut)
		if avail < int(q.minTokenToJoinQueue) {
			log.Infow("player doesn't have enough tokens after settlement",
				"player_id", playerID,
				"available_tokens", avail,
				"required_tokens", q.minTokenToJoinQueue)
			below = true
		}
	})
	return below
}
