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

// join queue should be idempotent
func (p *Player) joinQueue() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.status != proto.PlayerStatus_PLAYER_UNKNOWN && p.status != proto.PlayerStatus_PLAYER_WAITTING_CONTINUE {
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
	p.status = proto.PlayerStatus_PLAYER_UNKNOWN
	p.queue.HandleExitQueueEvent(&types.ExitQueueEvent{
		PlayerAddress: p.address,
	})
	return nil
}

func (p *Player) handleNewGameEvent(ctx context.Context, evt *types.GameCreatedEvent) error {
	if p.status != proto.PlayerStatus_PLAYER_IN_QUEUE && p.status != proto.PlayerStatus_PLAYER_UNKNOWN {
		return fmt.Errorf("cannot handle new game event, player status %s", p.status)
	}
	// send nothing if continued game
	if !evt.IsContinueGame {
		p.status = proto.PlayerStatus_PLAYER_MATCHED
		p.publisher.Publish(ctx, &proto.PublishRequest{
			Topic: p.address.String(),
			Event: &proto.Event{
				Type: proto.EventType_TYPE_MATCHED,
			},
		})
	} else {
		p.status = proto.PlayerStatus_PLAYER_IN_GAME
	}
	return nil
}

func (p *Player) handleGameReadyEvent(ctx context.Context, evt *types.GameReadyEvent) error {
	p.status = proto.PlayerStatus_PLAYER_IN_GAME
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
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_PART_CONFIRMED,
		},
	})
	return nil
}

func (p *Player) handleRoundReadyEvent(ctx context.Context, evt *types.RoundReadyEvent) error {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_READY,
		},
	})
	return nil
}

func (p *Player) handleCommitmentsOnChainEvent(ctx context.Context, evt *types.CommitmentsOnChainEvent) error {
	if p.status != proto.PlayerStatus_PLAYER_IN_GAME {
		return fmt.Errorf("player not in game, but got event type %d", reflect.TypeOf(evt))
	}
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_COMMITMENTS_ON_CHAIN,
		},
	})
	return nil
}

func (p *Player) handleCardsOnChainEvent(ctx context.Context, evt *types.CardsOnChainEvent) error {
	if p.status != proto.PlayerStatus_PLAYER_IN_GAME {
		return fmt.Errorf("player not in game, but got event type %d", reflect.TypeOf(evt))
	}
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_CARDS_ON_CHAIN,
		},
	})
	return nil
}

func (p *Player) handleContinueCanceledEvent(ctx context.Context, evt *types.ContinueCanceledEvent) error {
	if p.status != proto.PlayerStatus_PLAYER_UNKNOWN {
		return fmt.Errorf("player status not match, event type %d", reflect.TypeOf(evt))
	}
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_CONTINUE_CANCELED,
		},
	})
	return nil
}

func (p *Player) handleRoundCompletedEvent(ctx context.Context, evt *types.RoundCompletedEvent) error {
	if p.status != proto.PlayerStatus_PLAYER_IN_GAME {
		return fmt.Errorf("player not in game, but got event type %d", reflect.TypeOf(evt))
	}
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_COMPLETE,
		},
	})
	return nil
}

func (p *Player) handleGameCompletedEvent(ctx context.Context, evt *types.GameCompletedEvent) error {
	if p.status != proto.PlayerStatus_PLAYER_IN_GAME && p.status != proto.PlayerStatus_PLAYER_MATCHED {
		return fmt.Errorf("player not in game or matched, but got event type %d", reflect.TypeOf(evt))
	}
	p.status = proto.PlayerStatus_PLAYER_UNKNOWN
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

	return nil
}
