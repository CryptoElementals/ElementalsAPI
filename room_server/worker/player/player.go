package player

import (
	"context"

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
	case *types.TurnReadyEvent:
		p.handleTurnReadyEvent(p.ctx, evt)
	case *types.RoundPartialReadyEvent:
		p.handleRoundPartialReadyEvent(p.ctx, evt)
	case *types.CommitmentsOnChainEvent:
		p.handleCommitmentsOnChainEvent(p.ctx, evt)
	case *types.TurnCompletedEvent:
		p.handleTurnCompletedEvent(p.ctx, evt)
	// RoundCompletedEvent and GameCompletedEvent are now handled via TurnCompletedEvent with flags
	case *types.ContinueCanceledEvent:
		p.handleContinueCanceledEvent(ctx, evt)
	case *types.GamePhaseSyncEvent:
		p.handleGamePhaseSyncEvent(p.ctx, evt)
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
				Event: &proto.Event_GameMatched{
					GameMatched: &proto.GameMatched{
						GameId:              uint32(evt.GameID),
						Players:             players,
						ConfirmationTimeout: evt.ConfirmationTimeout,
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
					GameId:            uint32(evt.GameID),
					MaxRoundNum:       evt.MaxRoundNum,
					MaxTurnNum:        evt.MaxTurnNum,
					InitialHP:         evt.InitialHP,
					InitialMultiplier: evt.InitialMultiplier,
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

func (p *Player) handleTurnReadyEvent(ctx context.Context, evt *types.TurnReadyEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_TURN_READY, "receiver", p.address.String(), "game id", evt.GameID, "round", evt.RoundNumber, "turn", evt.TurnNumber)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_TURN_READY,
			Event: &proto.Event_TurnReady{
				TurnReady: &proto.TurnReady{
					GameId:                      uint32(evt.GameID),
					RoundNum:                    evt.RoundNumber,
					TurnNum:                     evt.TurnNumber,
					CommitmentSubmissionTimeout: evt.CommitmentSubmissionTimeout,
				},
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
	log.Debugw("publish event", "event type", proto.EventType_TYPE_COMMITMENTS_ON_CHAIN, "receiver", p.address.String(), "game id", evt.GameID, "round number", evt.RoundNumber, "turn number", evt.TurnNumber)
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_COMMITMENTS_ON_CHAIN,
			Event: &proto.Event_CommitmentsOnChain{
				CommitmentsOnChain: &proto.CommitmentsOnChain{
					GameId:                uint32(evt.GameID),
					RoundNum:              evt.RoundNumber,
					TurnNum:               evt.TurnNumber, // CardNum in proto corresponds to TurnNumber
					CardSubmissionTimeout: evt.CardSubmissionTimeout,
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

func (p *Player) handleTurnCompletedEvent(ctx context.Context, evt *types.TurnCompletedEvent) error {
	log.Debugw("publish event", "event type", proto.EventType_TYPE_TURN_COMPLETE, "receiver", p.address.String(), "game id", evt.GameID, "round", evt.RoundNumber, "turn", evt.TurnNumber, "isRoundComplete", evt.IsRoundComplete, "isGameComplete", evt.IsGameComplete)

	// Convert types.PlayerTurnInfo to proto.PlayerTurnInfo
	playerTurnInfos := make([]*proto.PlayerTurnInfo, 0, len(evt.PlayerTurnInfo))
	for _, playerTurnInfo := range evt.PlayerTurnInfo {
		if playerTurnInfo.SubmittedCard != nil {
			protoPlayerTurnInfo := &proto.PlayerTurnInfo{
				PlayerAddress: playerTurnInfo.PlayerAddress.ToProto(),
				SubmittedCard: conversion.TurnSubmittedCardToProtoRoundSubmittedCard(playerTurnInfo.SubmittedCard, evt.TurnNumber),
			}
			playerTurnInfos = append(playerTurnInfos, protoPlayerTurnInfo)
		}
	}

	// Convert dao.GameResult to battle.GameResult if game is complete
	var battleGameResult *proto.GameResult
	if evt.IsGameComplete && evt.GameResult != nil {
		battleGameResult = conversion.DbGameResultToProtoGameResult(evt.GameResult)
	}

	turnCompleted := &proto.TurnCompleted{
		GameId:          uint32(evt.GameID),
		RoundNum:        evt.RoundNumber,
		TurnNum:         evt.TurnNumber,
		IsRoundComplete: evt.IsRoundComplete,
		IsGameComplete:  evt.IsGameComplete,
		PlayerTurnInfos: playerTurnInfos,
	}
	if battleGameResult != nil {
		turnCompleted.GameResult = battleGameResult
	}
	if evt.ConfirmationTimeout != nil {
		turnCompleted.ConfirmationTimeout = evt.ConfirmationTimeout
	}
	if evt.GameContinueTimeout != nil {
		turnCompleted.GameContinueTimeout = evt.GameContinueTimeout
	}

	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_TURN_COMPLETE,
			Event: &proto.Event_TurnCompleted{
				TurnCompleted: turnCompleted,
			},
		},
	})
	return nil
}

// handleRoundCompletedEvent and handleGameCompletedEvent are no longer needed
// Round and game completion are now indicated via TurnCompletedEvent flags

func (p *Player) handleGamePhaseSyncEvent(ctx context.Context, evt *types.GamePhaseSyncEvent) error {
	log.Debugw("publish game phase sync event", "receiver", p.address.String())
	p.publisher.Publish(ctx, &proto.PublishRequest{
		Topic: p.address.String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_GAME_PHASE_SYNC,
			Event: &proto.Event_GamePhase{
				GamePhase: evt.GamePhase,
			},
		},
	})
	return nil
}
