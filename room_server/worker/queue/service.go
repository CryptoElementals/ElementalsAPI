package queue

import (
	"context"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type Service struct {
	ctx                 context.Context
	queue               *Queue
	minTokenToJoinQueue int32
	botWaitTime         int64
}

func NewService(ctx context.Context,
	workerManager *worker.WorkerManager,
	cache cache.Cache,
	gameCreator GameCreator,
	minTokenToJoinQueue int32,
	continueTimeout int64,
	continueTimeoutRedundancy int64,
	botWaitTime int64,
	statServiceEndpoint string,
) *Service {
	s := &Service{
		ctx:                 ctx,
		queue:               NewQueue(ctx, workerManager, cache, gameCreator, continueTimeout, continueTimeoutRedundancy, botWaitTime, minTokenToJoinQueue, statServiceEndpoint),
		minTokenToJoinQueue: minTokenToJoinQueue,
		botWaitTime:         botWaitTime,
	}
	return s
}

func (s *Service) Start() error {
	return s.queue.start()
}

func (s *Service) Stop() {
	s.queue.close()
}

func (s *Service) IsPlayerInQueue(address types.PlayerAddress) bool {
	return s.queue.isPlayerInQueue(address)
}

func (s *Service) GetPlayerContinueInfo(address types.PlayerAddress) *types.GameContinueInfo {
	return s.queue.getPlayerContinueInfo(address)
}

func (s *Service) HandleJoinQueueEvent(event *types.JoinQueueEvent) error {
	return s.queue.HandleJoinQueueEvent(event)
}

func (s *Service) HandleExitQueueEvent(event *types.ExitQueueEvent) error {
	return s.queue.HandleExitQueueEvent(event)
}

func (s *Service) HandleContinueGameEvent(event *types.PlayerContinueEvent) error {
	return s.queue.HandleContinueGameEvent(event)
}

func (s *Service) GetPlayerToken(playerId int64) (*proto.GetPlayerTokenResponse, error) {
	userToken, err := db.GetPlayerToken(s.ctx, playerId)
	if err != nil {
		log.Error("GetPlayerToken failed, err: ", err)
		return nil, err
	}
	return conversion.DbUserTokenToProtoGetPlayerTokenResponse(userToken), nil
}

func (s *Service) GameResultSettlement(event *types.GameCompletedEvent) error {
	return s.queue.GameResultSettlement(event)
}

func (s *Service) RefuseContinueGame(playerAddress types.PlayerAddress, lastGameID uint) error {
	return s.queue.RefuseContinueGame(playerAddress, lastGameID)
}

func (s *Service) RegisterBots(addrs ...*types.PlayerAddress) error {
	return s.queue.RegisterBots(addrs...)
}

func (s *Service) UnregisterBots(addrs ...*types.PlayerAddress) error {
	return s.queue.UnregisterBots(addrs...)
}
