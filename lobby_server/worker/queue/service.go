package queue

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/bot_manager"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// Service is the queue / matchmaking façade: join/exit, post-game continue rematch, token checks, and stat callbacks.
type Service struct {
	ctx   context.Context
	queue *Queue
}

func NewService(ctx context.Context,
	pub EventPublisher,
	botStore *bot_manager.RedisStore,
	gameCreator GameCreator,
	minTokenToJoinQueue int32,
	matchConfirmationTimeout int64,
	continueTimeout int64,
	continueTimeoutRedundancy int64,
	botWaitTime int64,
	botFreshnessSec int64,
	statServiceEndpoint string,
) (*Service, error) {
	q, err := NewQueue(ctx, pub, botStore, gameCreator, matchConfirmationTimeout, continueTimeout, continueTimeoutRedundancy, botWaitTime, botFreshnessSec, minTokenToJoinQueue, statServiceEndpoint)
	if err != nil {
		return nil, fmt.Errorf("queue: %w", err)
	}
	return &Service{ctx: ctx, queue: q}, nil
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

func (s *Service) IsPlayerInGame(address types.PlayerAddress) bool {
	return s.queue.isPlayerInGame(address)
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

// GetPlayerStatusResponse returns lobby queue/match/game status and optional details (Since = queue join ms, MatchID, GameID).
func (s *Service) GetPlayerStatusResponse(addr types.PlayerAddress) *proto.GetPlayerStatusResponse {
	if s.IsPlayerInQueue(addr) {
		out := &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_IN_QUEUE}
		if ms, ok := s.queue.queueJoinedAtMs(addr); ok {
			out.Since = &ms
		}
		return out
	}
	if s.IsPlayerPendingMatch(addr) {
		out := &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_PENDING_QUEUE_MATCH}
		if mid, ok := s.queue.pendingMatchID(addr); ok {
			out.MatchID = &mid
		}
		return out
	}
	if s.IsPlayerInGame(addr) {
		out := &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_IN_GAME}
		if gid, err := db.GetQueueLockedGameID(s.ctx, addr.Id, addr.TemporaryAddress); err == nil && gid != 0 {
			out.GameID = &gid
		}
		return out
	}
	return &proto.GetPlayerStatusResponse{Status: proto.PlayerStatus_PLAYER_UNKNOWN}
}
