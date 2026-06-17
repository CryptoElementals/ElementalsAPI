package ledgerserver

import (
	"context"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
)

const (
	tokenSourceChainDeposit  = "chain_deposit"
	tokenSourceChainWithdraw = "chain_withdraw"
)

func (s *Service) publishTokenUpdated(ctx context.Context, ev *proto.ChainTokenEvent, applyResult *db.ChainTokenEventApplyResult) {
	if s == nil || s.publisher == nil || ev == nil || applyResult == nil {
		return
	}
	if applyResult.Status != db.ChainTokenEventApplyFinalized {
		return
	}

	playerID := chainEventPlayerID(ev)
	if playerID < 0 {
		log.Errorf("publishTokenUpdated: invalid player_id for tx=%s", ev.GetTxHash())
		return
	}

	userToken, err := db.GetPlayerToken(ctx, playerID)
	if err != nil {
		log.Errorf("publishTokenUpdated: GetPlayerToken player=%d: %v", playerID, err)
		return
	}

	source := tokenSourceChainDeposit
	if dao.ChainTokenLedgerEventType(ev.GetEventType()) == dao.ChainTokenLedgerEventWithdraw {
		source = tokenSourceChainWithdraw
	}

	payload := &proto.TokenUpdated{
		PlayerId:         playerID,
		TokenDelta:       applyResult.TokenDelta,
		Tokens:           applyResult.NewBalance,
		Points:           userToken.Points,
		LockedTokens:     sumLockedTokens(userToken),
		Source:           source,
		ChainId:          ev.GetChainId(),
		TxHash:           strings.ToLower(strings.TrimSpace(ev.GetTxHash())),
		LogIndex:         ev.GetLogIndex(),
		CollectorAddress: strings.ToLower(strings.TrimSpace(ev.GetCollectorAddress())),
	}

	out := &proto.Event{
		Type: proto.EventType_TYPE_TOKEN_UPDATED,
		Event: &proto.Event_TokenUpdated{
			TokenUpdated: payload,
		},
	}
	if err := pubsub.Publish(ctx, s.publisher, out); err != nil {
		log.Errorf("publishTokenUpdated failed player=%d tx=%s: %v", playerID, ev.GetTxHash(), err)
	}
}

func chainEventPlayerID(ev *proto.ChainTokenEvent) int64 {
	if ev == nil {
		return -1
	}
	switch dao.ChainTokenLedgerEventType(ev.GetEventType()) {
	case dao.ChainTokenLedgerEventDeposit:
		if dep := ev.GetDeposit(); dep != nil {
			return dep.GetPlayerId()
		}
	case dao.ChainTokenLedgerEventWithdraw:
		if wd := ev.GetWithdraw(); wd != nil {
			return wd.GetPlayerId()
		}
	}
	return -1
}

func sumLockedTokens(userToken *dao.UserToken) int32 {
	if userToken == nil {
		return 0
	}
	var total int32
	for _, locked := range userToken.LockedTokens {
		if locked == nil {
			continue
		}
		total += locked.TokenAmount
	}
	return total
}
