package queue

import (
	"context"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

type Service struct {
	ctx                 context.Context
	queue               *Queue
	minTokenToJoinQueue int32
}

func NewService(ctx context.Context,
	workerManager *worker.WorkerManager,
	cache cache.Cache,
	gameCreator GameCreator,
	minTokenToJoinQueue int32,
) *Service {
	return &Service{
		ctx:   ctx,
		queue: NewQueue(ctx, workerManager, cache, gameCreator),
	}
}

func (s *Service) Start() error {
	return s.queue.start()
}

func (s *Service) Stop() error {
	s.queue.close()
	return nil
}

func (s *Service) IsPlayerInQueue(address types.PlayerAddress) bool {
	return s.queue.isPlayerInQueue(address)
}

func (s *Service) HandleJoinQueueEvent(event *types.JoinQueueEvent) error {
	err := s.lockToken(&event.PlayerAddress)
	if err != nil {
		log.Error("lockToken failed, err: ", err)
		return err
	}
	return s.queue.HandleJoinQueueEvent(event)
}

func (s *Service) HandleExitQueueEvent(event *types.ExitQueueEvent) error {
	s.queue.HandleExitQueueEvent(event)
	return nil
}

func (s *Service) HandleContinueGameEvent(event *types.PlayerContinueEvent) error {
	return s.queue.HandleContinueGameEvent(event)
}

func (s *Service) GetPlayerToken(address *types.PlayerAddress) *dao.UserToken {
	userToken, err := db.GetPlayerToken(s.ctx, address.WalletAddress)
	if err != nil {
		log.Error("GetPlayerToken failed, err: ", err)
		return nil
	}
	return userToken
}

func (s *Service) GameResultSettlement(event *types.GameCompletedEvent) error {
	return s.queue.GameResultSettlement(event)
}

func (s *Service) lockToken(address *types.PlayerAddress) error {
	if s.minTokenToJoinQueue <= 0 {
		return nil
	}
	return db.LockUserToken(s.ctx, address.WalletAddress, address.TemporaryAddress, s.minTokenToJoinQueue)
}
