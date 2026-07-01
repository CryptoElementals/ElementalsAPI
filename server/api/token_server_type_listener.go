package api

import (
	"sync"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
)

const tokenServerTypePromoterID = "token-server-type-promoter"

var (
	tokenServerTypeListenerMu sync.Mutex
	tokenServerTypeListenerOn bool
	tokenServerTypeStopCh     chan struct{}
)

// StartTokenServerTypeListener subscribes to all normal-environment token events and promotes trial users.
func StartTokenServerTypeListener() error {
	tokenServerTypeListenerMu.Lock()
	defer tokenServerTypeListenerMu.Unlock()
	if tokenServerTypeListenerOn {
		return nil
	}

	eventBus, err := getSubscribeTokenEventBus()
	if err != nil {
		return err
	}

	msgCh, errCh := eventBus.RegisterAllEventsSubscriber(tokenServerTypePromoterID)
	stopCh := make(chan struct{})
	tokenServerTypeStopCh = stopCh
	tokenServerTypeListenerOn = true

	go runTokenServerTypeListener(msgCh, errCh, stopCh)
	log.Info("token server type listener started")
	return nil
}

// StopTokenServerTypeListener unregisters the all-events token subscriber.
func StopTokenServerTypeListener() {
	tokenServerTypeListenerMu.Lock()
	if !tokenServerTypeListenerOn {
		tokenServerTypeListenerMu.Unlock()
		return
	}
	close(tokenServerTypeStopCh)
	tokenServerTypeStopCh = nil
	tokenServerTypeListenerOn = false
	tokenServerTypeListenerMu.Unlock()

	eventBus, err := getSubscribeTokenEventBus()
	if err != nil {
		log.Errorw("stop token server type listener: event bus unavailable", "err", err)
		return
	}
	eventBus.UnregisterAllEventsSubscriber(tokenServerTypePromoterID)
	log.Info("token server type listener stopped")
}

func runTokenServerTypeListener(msgCh <-chan *proto.Message, errCh <-chan error, stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		case err, ok := <-errCh:
			if !ok {
				return
			}
			if err != nil {
				log.Errorw("token server type listener subscriber error", "err", err)
			}
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			handleTokenServerTypePromotion(msg)
		}
	}
}

func handleTokenServerTypePromotion(msg *proto.Message) {
	if msg == nil || msg.GetEvent() == nil {
		return
	}
	if msg.GetEvent().GetType() != proto.EventType_TYPE_TOKEN_UPDATED {
		return
	}
	tu := msg.GetEvent().GetTokenUpdated()
	if tu == nil {
		return
	}
	playerID := tu.GetPlayerId()
	if playerID <= 0 {
		return
	}

	updated, err := db.PromoteUserServerTypeToNormalIfTrial(playerID)
	if err != nil {
		log.Errorw("promote user server type to normal", "player_id", playerID, "err", err)
		return
	}
	if updated {
		log.Infow("promoted user server type to normal", "player_id", playerID)
	}
}
