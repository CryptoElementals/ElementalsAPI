package player

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type playerGameInfo struct {
	currentGame     uint
	currentRound    uint
	contractAddress string
	roundStarted    int64
	roundTimeout    int64

	players map[types.PlayerAddress]bool
}

type Player struct {
	ctx          context.Context
	lock         sync.RWMutex
	address      types.PlayerAddress
	info         playerGameInfo
	publisher    Publisher
	workerManger *worker.WorkerManager
	status       proto.PlayerStatus
}

func NewPlayer(ctx context.Context,
	address types.PlayerAddress,
	publisher Publisher,
	workerManger *worker.WorkerManager) *Player {
	p := &Player{
		ctx:          ctx,
		address:      address,
		publisher:    publisher,
		workerManger: workerManger,
		info: playerGameInfo{
			players: map[types.PlayerAddress]bool{},
		},
	}
	p.createSelf()
	return p
}

func (p *Player) createSelf() {
	p.workerManger.SpwanWorker(p.ctx, p.address.String(), types.WORKER_TYPE_PLAYER, p)
}

func (p *Player) Handle(ctx context.Context, event *types.Event) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if evt, ok := event.Data.(*types.ErrorEvent); ok {
		log.Errorf("received error: %T, err: %s", evt.OriginalEvent.Data, evt.Err.Error())
		return nil
	}
	if p.status == proto.PlayerStatus_PLAYER_KNOWN {
		evt := event.Data.(*types.GameCreatedEvent)
		p.setNewGameInfo(evt)
		p.status = proto.PlayerStatus_PLAYER_IN_GAME
	}
	if p.status == proto.PlayerStatus_PLAYER_IN_QUEUE {
		evt := event.Data.(*types.GameCreatedEvent)
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

	p.workerManger.SendEvent(types.QUEUE_MANAGER_ID, types.NewEvent(p.address.String(), &types.JoinQueueEvent{
		PlayerAddress: p.address,
	}))
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
	p.workerManger.SendEvent(types.QUEUE_MANAGER_ID, types.NewEvent(p.address.String(), &types.ExitQueueEvent{
		PlayerAddress: p.address,
	}))
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
	p.setNewGameInfo(evt)
}

func (p *Player) setNewGameInfo(evt *types.GameCreatedEvent) {
	p.info.currentGame = uint(evt.GameID)
	for _, player := range evt.Players {
		p.info.players[player] = false
	}
}

func (p *Player) handleGameReadyEvent(ctx context.Context, evt *types.GameReadyEvent) {
	p.info.contractAddress = evt.ContractAddress
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_GAME_CREATED,
		},
	})
}

func (p *Player) handleRoundPartialReadyEvent(ctx context.Context, evt *types.RoundPartialReadyEvent) {
	p.info.players[evt.ReadyAddress] = true
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
	p.info.currentRound = uint(evt.RoundNumber)
	p.info.roundStarted = evt.RoundStartedAt
	p.info.roundTimeout = evt.RoundTimeout
}

func (p *Player) handleCommitmentsOnChainEvent(ctx context.Context, evt *types.CommitmentsOnChainEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_COMMITMENTS_ON_CHAIN,
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

func (p *Player) ToGamePhase() *proto.GamePhase {
	protoPlayers := make([]*proto.GamePhasePlayer, 0)
	for addr, p := range p.info.players {
		protoPlayers = append(protoPlayers, &proto.GamePhasePlayer{
			Address:     addr.ToProto(),
			IsConfirmed: p,
		})
	}
	return &proto.GamePhase{
		PvPInfo: &proto.PvPInfo{
			GameID:          uint32(p.info.currentGame),
			Status:          p.status,
			ContractAddress: p.info.contractAddress,
			BeginAt:         uint64(p.info.roundStarted),
			TimeoutDuration: uint64(p.info.roundTimeout),
		},
		Players: protoPlayers,
	}
}
