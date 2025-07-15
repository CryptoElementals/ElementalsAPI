package player

import (
	"context"
	"fmt"
	"sync"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type Publisher interface {
	Publish(ctx context.Context, req *proto.PublishRequest) (*proto.PublishResponse, error)
}

type Player struct {
	ctx          context.Context
	lock         sync.RWMutex
	address      types.PlayerAddress
	currentGame  uint
	currentRound uint
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
	}
	p.createSelf()
	return p
}

func (p *Player) Handle(ctx context.Context, event *types.Event) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.status == proto.PlayerStatus_PLAYER_IN_QUEUE {
		if event.EventType == types.EVENT_TYPE_GAME_CREATED {
			evt := event.Data.(*types.GameCreatedEvent)
			p.handleNewGameEvent(p.ctx, evt)
			p.status = proto.PlayerStatus_PLAYER_IN_GAME
		} else {
			return fmt.Errorf("player in queue, but got event type %d", event.EventType)
		}
	}

	if p.status != proto.PlayerStatus_PLAYER_IN_GAME {
		return fmt.Errorf("player not in game, but got event type %d", event.EventType)
	}

	switch event.EventType {
	case types.EVENT_TYPE_GAME_READY:
		evt := event.Data.(*types.GameReadyEvent)
		p.handleGameReadyEvent(p.ctx, evt)
	case types.EVENT_TYPE_ROUND_READY:
		evt := event.Data.(*types.RoundReadyEvent)
		p.handleRoundReadyEvent(p.ctx, evt)
	case types.EVENT_TYPE_COMMITMENTS_ON_CHAIN:
		evt := event.Data.(*types.CommitmentsOnChainEvent)
		p.handleCommitmentsOnChainEvent(p.ctx, evt)
	case types.EVENT_TYPE_CARDS_ON_CHAIN:
		evt := event.Data.(*types.CardsOnChainEvent)
		p.handleCardsOnChainEvent(p.ctx, evt)
	case types.EVENT_TYPE_ROUND_COMPLETED:
		evt := event.Data.(*types.RoundCompletedEvent)
		p.handleRoundCompletedEvent(p.ctx, evt)
	case types.EVENT_TYPE_GAME_COMPLETED:
		evt := event.Data.(*types.GameCompletedEvent)
		p.handleGameCompletedEvent(p.ctx, evt)
		p.status = proto.PlayerStatus_PLAYER_KNOWN
	}
	return nil
}

func (p *Player) createSelf() {
	p.workerManger.SpwanWorker(p.address.String(), types.WORKER_TYPE_PLAYER, p)
}

// join queue should be idempotent
func (p *Player) joinQueue() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.status != proto.PlayerStatus_PLAYER_KNOWN {
		return fmt.Errorf("join queue failed, player status %s", p.status)
	}

	p.workerManger.SendEvent(types.QUEUE_MANAGER_ID, types.NewEvent(p.address.String(), types.EVENT_TYPE_JOIN_QUEUE, &types.JoinQueueEvent{
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
	p.workerManger.SendEvent(types.QUEUE_MANAGER_ID, types.NewEvent(p.address.String(), types.EVENT_TYPE_EXIT_QUEUE, &types.ExitQueueEvent{
		PlayerAddress: p.address,
	}))
	p.status = proto.PlayerStatus_PLAYER_KNOWN
	return nil
}

func (p *Player) sync(gameInfo *proto.GameInfo) error {
	if p.status != proto.PlayerStatus_PLAYER_KNOWN {
		return fmt.Errorf("sync failed, player status %s", p.status)
	}
	if gameInfo == nil {
		gameInfo = &proto.GameInfo{}
	}
	p.publisher.Publish(p.ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_SYNC_INFO,
			Data: &proto.Event_GameInfo{
				GameInfo: gameInfo,
			},
		},
	})
	return nil
}

func (p *Player) handleNewGameEvent(ctx context.Context, evt *types.GameCreatedEvent) {
	protoPlayers := make([]*proto.PlayerAddress, 0)
	for _, player := range evt.Players {
		protoPlayers = append(protoPlayers, player.ToProto())
	}
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_GAME_CREATED,
			Data: &proto.Event_GameCreated{
				GameCreated: &proto.GameCreated{
					GameId:  uint32(evt.GameID),
					Players: protoPlayers,
				},
			},
		},
	})
	p.currentGame = uint(evt.GameID)
}

func (p *Player) handleGameReadyEvent(ctx context.Context, evt *types.GameReadyEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_GAME_READY,
			Data: &proto.Event_GameReady{
				GameReady: &proto.GameReady{
					GameId:          uint32(evt.GameID),
					ContractAddress: evt.ContractAddress,
				},
			},
		},
	})
}

func (p *Player) handleRoundReadyEvent(ctx context.Context, evt *types.RoundReadyEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_ROUND_READY,
			Data: &proto.Event_RoundReady{
				RoundReady: &proto.RoundReady{
					GameId:   uint32(evt.GameID),
					RoundNum: uint32(evt.RoundNumber),
				},
			},
		},
	})
	p.currentRound = uint(evt.RoundNumber)
}

func (p *Player) handleCommitmentsOnChainEvent(ctx context.Context, evt *types.CommitmentsOnChainEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_COMMITMENTS_ON_CHAIN,
			Data: &proto.Event_CommitmentsOnChain{
				CommitmentsOnChain: &proto.CommitmentsOnChain{
					GameId:   uint32(evt.GameID),
					RoundNum: uint32(evt.RoundNumber),
				},
			},
		},
	})
}

func (p *Player) handleCardsOnChainEvent(ctx context.Context, evt *types.CardsOnChainEvent) {
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_CARDS_ON_CHAIN,
			Data: &proto.Event_CardsOnChain{
				CardsOnChain: &proto.CardsOnChain{
					GameId:   uint32(evt.GameID),
					RoundNum: uint32(evt.RoundNumber),
				},
			},
		},
	})
}

func (p *Player) handleRoundCompletedEvent(ctx context.Context, evt *types.RoundCompletedEvent) {
	protoRound := conversion.DbGameRoundToProtoGameRound(evt.RoundInfo)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_ROUND_COMPLETED,
			Data: &proto.Event_RoundCompleted{
				RoundCompleted: &proto.RoundCompleted{
					GameId:    uint32(evt.GameID),
					RoundInfo: protoRound,
				},
			},
		},
	})
}

func (p *Player) handleGameCompletedEvent(ctx context.Context, evt *types.GameCompletedEvent) {
	protoGameInfo := conversion.DbGameInfoToProtoGameInfo(evt.GameInfo)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_GAME_COMPLETED,
			Data: &proto.Event_GameInfo{
				GameInfo: protoGameInfo,
			},
		},
	})
}
