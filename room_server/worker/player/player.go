package player

import (
	"context"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type Player struct {
	ctx          context.Context
	address      types.PlayerAddress
	publisher    Publisher
	workerManger *worker.WorkerManager
	status       proto.PlayerStatus
}

func NewPlayer(ctx context.Context,
	address types.PlayerAddress,
	publisher Publisher,
	workerManger *worker.WorkerManager,
) *Player {
	p := &Player{
		ctx:          ctx,
		address:      address,
		publisher:    publisher,
		workerManger: workerManger,
	}
	return p
}

func (p *Player) createSelf() {
	p.workerManger.SpwanWorker(p.ctx, p.address.String(), types.WORKER_TYPE_PLAYER, p)
}

func (p *Player) Handle(ctx context.Context, event *types.Event) error {
	switch evt := event.Data.(type) {
	case *types.GameCreatedEvent:
		return p.handleNewGameEvent(p.ctx, evt)
	case *types.GameReadyEvent:
		p.handleGameReadyEvent(p.ctx, evt)
	case *types.RoundReadyEvent:
		p.handleRoundReadyEvent(p.ctx, evt)
	case *types.RoundPartialReadyEvent:
		p.handleRoundPartialReadyEvent(p.ctx, evt)
	case *types.CommitmentsOnChainEvent:
		p.handleCommitmentsOnChainEvent(p.ctx, evt)
	case *types.CardsOnChainEvent:
		p.handleCardsOnChainEvent(p.ctx, evt)
	case *types.RoundCompletedEvent:
		p.handleRoundCompletedEvent(p.ctx, evt)
	case *types.GameCompletedEvent:
		p.handleGameCompletedEvent(p.ctx, evt)
	case *types.ContinueCanceledEvent:
		p.handleContinueCanceledEvent(ctx, evt)
	}
	return nil
}

func (p *Player) handleNewGameEvent(ctx context.Context, evt *types.GameCreatedEvent) error {
	// send nothing if continued game
	if !evt.IsContinueGame {
		log.Debugw("publish event", "event type", proto.EventType_TYPE_MATCHED, "receiver", p.address.String(), "game id", evt.GameID)
		p.publisher.Publish(ctx, &proto.PublishRequest{
			Topic: p.address.String(),
			Event: &proto.Event{
				Type: proto.EventType_TYPE_MATCHED,
			},
		})
	}
	return nil
}

func (p *Player) handleGameReadyEvent(ctx context.Context, evt *types.GameReadyEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_GAME_CREATED, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_GAME_CREATED,
		},
	})
	return nil
}

func (p *Player) handleRoundPartialReadyEvent(ctx context.Context, evt *types.RoundPartialReadyEvent) error {
	// don't send event to itself
	if p.address == evt.ReadyAddress {
		return nil
	}
	log.Debugw("publish event", "event type", proto.EventType_TYPE_PART_CONFIRMED, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_PART_CONFIRMED,
		},
	})
	return nil
}

func (p *Player) handleRoundReadyEvent(ctx context.Context, evt *types.RoundReadyEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_ROUND_READY, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_READY,
		},
	})
	return nil
}

func (p *Player) handleCommitmentsOnChainEvent(ctx context.Context, evt *types.CommitmentsOnChainEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_COMMITMENTS_ON_CHAIN, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_COMMITMENTS_ON_CHAIN,
		},
	})
	return nil
}

func (p *Player) handleCardsOnChainEvent(ctx context.Context, evt *types.CardsOnChainEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_CARDS_ON_CHAIN, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_CARDS_ON_CHAIN,
		},
	})
	return nil
}

func (p *Player) handleContinueCanceledEvent(ctx context.Context, evt *types.ContinueCanceledEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_CONTINUE_CANCELED, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_CONTINUE_CANCELED,
		},
	})
	return nil
}

func (p *Player) handleRoundCompletedEvent(ctx context.Context, evt *types.RoundCompletedEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_ROUND_COMPLETE, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_COMPLETE,
		},
	})
	return nil
}

func (p *Player) handleGameCompletedEvent(ctx context.Context, evt *types.GameCompletedEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_ROUND_COMPLETE, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_COMPLETE,
		},
	})
	log.Debugw("publish event", "event type", proto.EventType_TYPE_GAME_COMPLETE, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_GAME_COMPLETE,
		},
	})
	return nil
}
