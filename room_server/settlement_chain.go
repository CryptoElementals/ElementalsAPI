package roomserver

import (
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/room_server/worker/queue"
	"github.com/CryptoElementals/common/room_server/worker/turnament"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

// gameResultSettlerChain runs PVP queue settlement first, then tournament bracket advancement.
type gameResultSettlerChain struct {
	queueSvc *queue.Service
	tournSvc *turnament.TournamentQueueService
}

func (c *gameResultSettlerChain) GameResultSettlement(e *types.GameCompletedEvent) error {
	if err := c.queueSvc.GameResultSettlement(e); err != nil {
		return err
	}
	if err := c.tournSvc.GameResultSettlementHook(e); err != nil {
		log.Errorw("tournament game completion", "err", err, "game_id", e.GameID)
	}
	return nil
}

var _ game.GameResultSettler = (*gameResultSettlerChain)(nil)
