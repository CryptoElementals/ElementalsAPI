package player

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type Player struct {
	ctx          context.Context
	lock         sync.RWMutex
	address      types.PlayerAddress
	publisher    Publisher
	workerManger *worker.WorkerManager
	status       proto.PlayerStatus
	queue        Queuer
}

func NewPlayer(ctx context.Context,
	address types.PlayerAddress,
	publisher Publisher,
	workerManger *worker.WorkerManager,
	queue Queuer,
) *Player {
	p := &Player{
		ctx:          ctx,
		address:      address,
		publisher:    publisher,
		workerManger: workerManger,
		queue:        queue,
	}
	return p
}

func (p *Player) createSelf() {
	p.workerManger.SpwanWorker(p.ctx, p.address.String(), types.WORKER_TYPE_PLAYER, p)
}

func (p *Player) Handle(ctx context.Context, event *types.Event) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.status == proto.PlayerStatus_PLAYER_KNOWN {
		switch evt := event.Data.(type) {
		case *types.ContinueCanceledEvent:
			p.handleContinueCanceledEvent(ctx, evt)
		case *types.GameCreatedEvent:
			p.status = proto.PlayerStatus_PLAYER_IN_GAME
		}
	}
	if p.status == proto.PlayerStatus_PLAYER_IN_QUEUE {
		evt, ok := event.Data.(*types.GameCreatedEvent)
		if !ok {
			return fmt.Errorf("player not in queue, but got event type %d", reflect.TypeOf(event.Data))
		}
		p.handleNewGameEvent(p.ctx, evt)
		p.status = proto.PlayerStatus_PLAYER_IN_GAME
	}

	if p.status != proto.PlayerStatus_PLAYER_IN_GAME {
		return fmt.Errorf("player not in game, but got event type %d", reflect.TypeOf(event.Data))
	}

	switch evt := event.Data.(type) {
	case *types.GameReadyEvent:
		p.handleGameReadyEvent(p.ctx, evt)
	case *types.RoundReadyEvent:
		p.handleRoundReadyEvent(p.ctx, evt)
	case *types.RoundPartialReadyEvent:
		p.handleRoundPartialReadyEvent(p.ctx, evt)
	case *types.CommitmentsOnChainEvent:
		p.handleCommitmentsOnChainEvent(p.ctx, evt)
	case *types.RoundCompletedEvent:
		p.handleRoundCompletedEvent(p.ctx, evt)
	case *types.GameCompletedEvent:
		p.handleGameCompletedEvent(p.ctx, evt)
		p.status = proto.PlayerStatus_PLAYER_KNOWN
	}
	return nil
}

// join queue should be idempotent
func (p *Player) joinQueue() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.status != proto.PlayerStatus_PLAYER_KNOWN && p.status != proto.PlayerStatus_PLAYER_WAITTING_CONTINUE {
		return fmt.Errorf("join queue failed, player status %s", p.status)
	}

	err := p.queue.HandleJoinQueueEvent(&types.JoinQueueEvent{
		PlayerAddress: p.address,
	})
	if err != nil {
		return err
	}
	p.status = proto.PlayerStatus_PLAYER_IN_QUEUE
	return nil
}

// join queue should be idempotent
func (p *Player) exitQueue() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.status != proto.PlayerStatus_PLAYER_IN_QUEUE {
		return fmt.Errorf("join queue failed, player status %s", p.status)
	}
	p.queue.HandleExitQueueEvent(&types.ExitQueueEvent{
		PlayerAddress: p.address,
	})
	p.status = proto.PlayerStatus_PLAYER_KNOWN
	return nil
}

func (p *Player) handleNewGameEvent(ctx context.Context, evt *types.GameCreatedEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_MATCHED,
		},
	})
}

func (p *Player) handleGameReadyEvent(ctx context.Context, evt *types.GameReadyEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_GAME_CREATED,
		},
	})
}

func (p *Player) handleRoundPartialReadyEvent(ctx context.Context, evt *types.RoundPartialReadyEvent) {
	// don't send event to itself
	if p.address == evt.ReadyAddress {
		return
	}
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_PART_CONFIRMED,
		},
	})
}

func (p *Player) handleRoundReadyEvent(ctx context.Context, evt *types.RoundReadyEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_READY,
		},
	})
}

func (p *Player) handleCommitmentsOnChainEvent(ctx context.Context, evt *types.CommitmentsOnChainEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_COMMITMENTS_ON_CHAIN,
		},
	})
}

func (p *Player) handleContinueCanceledEvent(ctx context.Context, evt *types.ContinueCanceledEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_CONTINUE_CANCELED,
		},
	})
}

func (p *Player) handleRoundCompletedEvent(ctx context.Context, evt *types.RoundCompletedEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_CARDS_ON_CHAIN,
		},
	})
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_COMPLETE,
		},
	})
}

func (p *Player) handleGameCompletedEvent(ctx context.Context, evt *types.GameCompletedEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_CARDS_ON_CHAIN,
		},
	})
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_COMPLETE,
		},
	})
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_GAME_COMPLETE,
		},
	})
	p.status = proto.PlayerStatus_PLAYER_KNOWN
}
