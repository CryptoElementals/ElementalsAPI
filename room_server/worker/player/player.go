package player

import (
	"context"
	"errors"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
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
		players := make([]*proto.PlayerAddress, 0, len(evt.Players))
		for _, player := range evt.Players {
			players = append(players, player.ToProto())
		}
		p.publisher.Publish(ctx, &proto.PublishRequest{
			Topic: p.address.String(),
			Event: &proto.Event{
				Type: proto.EventType_TYPE_MATCHED,
				Event: &proto.Event_GameCreated{
					GameCreated: &proto.GameCreated{
						GameId:  uint32(evt.GameID),
						Players: players,
					},
				},
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
			Event: &proto.Event_GameReady{
				GameReady: &proto.GameReady{
					GameId:          uint32(evt.GameID),
					ContractAddress: evt.ContractAddress,
				},
			},
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
			Event: &proto.Event_X{
				X: &emptypb.Empty{},
			},
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
			Event: &proto.Event_RoundReady{
				RoundReady: &proto.RoundReady{
					GameId:   uint32(evt.GameID),
					RoundNum: evt.RoundNumber,
				},
			},
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
			Event: &proto.Event_CommitmentsOnChain{
				CommitmentsOnChain: &proto.CommitmentsOnChain{
					GameId:   uint32(evt.GameID),
					RoundNum: evt.RoundNumber,
				},
			},
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
			Event: &proto.Event_CardsOnChain{
				CardsOnChain: &proto.CardsOnChain{
					GameId:   uint32(evt.GameID),
					RoundNum: evt.RoundNumber,
				},
			},
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
			Event: &proto.Event_X{
				X: &emptypb.Empty{},
			},
		},
	})
	return nil
}

func (p *Player) handleRoundCompletedEvent(ctx context.Context, evt *types.RoundCompletedEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_ROUND_COMPLETE, "receiver", p.address.String(), "game id", evt.GameID)

	// Convert dao.Round to proto.Round (rpc.Round)
	var roundInfo *proto.Round
	if evt.RoundInfo != nil {
		roundInfo = conversion.DbGameRoundToProtoGameRound(evt.RoundInfo)
	}

	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_COMPLETE,
			Event: &proto.Event_RoundCompleted{
				RoundCompleted: &proto.RoundCompleted{
					GameId:    uint32(evt.GameID),
					RoundInfo: roundInfo,
				},
			},
		},
	})
	return nil
}

func (p *Player) handleGameCompletedEvent(ctx context.Context, evt *types.GameCompletedEvent) error {
	if evt.GameInfo == nil {
		return errors.New("game info is nil")
	}
	// Get the last round from GameInfo if available
	var roundInfo *proto.Round
	if len(evt.GameInfo.Rounds) > 0 {
		// Get the last round
		lastRound := evt.GameInfo.Rounds[len(evt.GameInfo.Rounds)-1]
		roundInfo = conversion.DbGameRoundToProtoGameRound(lastRound)
	}

	log.Debugw("publish event", "event type", proto.EventType_TYPE_ROUND_COMPLETE, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_ROUND_COMPLETE,
			Event: &proto.Event_RoundCompleted{
				RoundCompleted: &proto.RoundCompleted{
					GameId:    uint32(evt.GameID),
					RoundInfo: roundInfo,
				},
			},
		},
	})

	// Convert dao.GameResult to battle.GameResult
	var battleGameResult *proto.GameResult
	if evt.GameInfo.GameResult != nil {
		if evt.GameInfo.GameResult != nil {
			battleGameResult = conversion.DbGameResultToProtoGameResult(evt.GameInfo.GameResult)
			battleGameResult.GameContinueTimeout = uint64(evt.GameInfo.GameArgs.ContinueTimeout)
		}
	}

	log.Debugw("publish event", "event type", proto.EventType_TYPE_GAME_COMPLETE, "receiver", p.address.String(), "game id", evt.GameID)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_GAME_COMPLETE,
			Event: &proto.Event_GameCompleted{
				GameCompleted: &proto.GameCompleted{
					GameId:     uint32(evt.GameID),
					GameResult: battleGameResult,
				},
			},
		},
	})
	return nil
}
