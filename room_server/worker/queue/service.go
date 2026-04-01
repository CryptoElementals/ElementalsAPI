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

// Service is the queue / matchmaking façade: join/exit, post-game continue rematch, token checks, and stat callbacks.
type Service struct {
	ctx   context.Context
	queue *Queue
}

func NewService(ctx context.Context,
	workerManager *worker.WorkerManager,
	pub EventPublisher,
	cache cache.Cache,
	gameCreator GameCreator,
	minTokenToJoinQueue int32,
	matchConfirmationTimeout int64,
	continueTimeout int64,
	continueTimeoutRedundancy int64,
	botWaitTime int64,
	statServiceEndpoint string,
) *Service {
	return &Service{
		ctx:   ctx,
		queue: NewQueue(ctx, workerManager, pub, cache, gameCreator, matchConfirmationTimeout, continueTimeout, continueTimeoutRedundancy, botWaitTime, minTokenToJoinQueue, statServiceEndpoint),
	}
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

func (s *Service) HandleJoinQueueEvent(player *proto.PlayerAddress) error {
	return s.queue.HandleJoinQueueEvent(player)
}

func (s *Service) HandleExitQueueEvent(player *proto.PlayerAddress) error {
	return s.queue.HandleExitQueueEvent(player)
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

func (s *Service) RegisterBots(addrs ...*types.PlayerAddress) error {
	return s.queue.RegisterBots(addrs...)
}

func (s *Service) UnregisterBots(addrs ...*types.PlayerAddress) error {
	return s.queue.UnregisterBots(addrs...)
}

// HandleConfirmMatch records a queue-side confirmation for a pending game_match (see Rpc ConfirmMatch).
func (s *Service) HandleConfirmMatch(req *proto.ConfirmMatchRequest) error {
	return s.queue.HandleConfirmMatch(req)
}

// HandleCancelMatch cancels a pending game_match (see Rpc CancelMatch).
func (s *Service) HandleCancelMatch(req *proto.CancelMatchRequest) error {
	return s.queue.HandleCancelMatch(req)
}

// IsPlayerPendingMatch reports whether the player is in the pre-game game_match confirmation phase.
func (s *Service) IsPlayerPendingMatch(address types.PlayerAddress) bool {
	return s.queue.IsPlayerPendingMatch(address)
}
