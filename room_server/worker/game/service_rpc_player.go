package game

import (
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

func (s *Service) queueOrErr() (PlayerQueue, error) {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	if s.queue == nil {
		return nil, errors.New("queue not configured")
	}
	return s.queue, nil
}

// --- rpc/server.PlayerRequestHandler ---

func (s *Service) JoinQueue(req *proto.PlayerAddress) error {
	var address types.PlayerAddress
	address.FromProto(req)
	q, err := s.queueOrErr()
	if err != nil {
		return err
	}
	status := s.getPlayerStatusLocked(q, address)
	if status != proto.PlayerStatus_PLAYER_UNKNOWN {
		return fmt.Errorf("player cannot join queue, player status: %s", status)
	}
	return q.HandleJoinQueueEvent(req)
}

func (s *Service) ExitQueue(req *proto.PlayerAddress) error {
	q, err := s.queueOrErr()
	if err != nil {
		return err
	}
	return q.HandleExitQueueEvent(req)
}

func (s *Service) RefuseContinueGame(req *proto.RefuseContinueGameRequest) error {
	var address types.PlayerAddress
	address.FromProto(req.Player)
	q, err := s.queueOrErr()
	if err != nil {
		return err
	}
	return q.RefuseContinueGame(address, uint(req.LastGameID))
}

func (s *Service) ContinueGame(req *proto.ContinueGameRequest) error {
	q, err := s.queueOrErr()
	if err != nil {
		return err
	}
	return q.HandleContinueGameEvent(req)
}

func (s *Service) ConfirmBattle(req *proto.ConfirmBattleRequest) error {
	return s.gameManager.HandleConfirmBattle(req)
}

func (s *Service) IsPlayerInQueue(req *proto.PlayerAddress) (*proto.IsPlayerInQueueResponse, error) {
	var address types.PlayerAddress
	address.FromProto(req)
	s.playerMu.Lock()
	q := s.queue
	s.playerMu.Unlock()
	if q == nil {
		return &proto.IsPlayerInQueueResponse{IsInQueue: false}, nil
	}
	return &proto.IsPlayerInQueueResponse{IsInQueue: q.IsPlayerInQueue(address)}, nil
}

func (s *Service) Surrender(req *proto.SurrenderRequest) error {
	return s.gameManager.HandleSurrender(req)
}

func (s *Service) GetPlayerStatus(req *proto.PlayerAddress) (*proto.GetPlayerStatusResponse, error) {
	var address types.PlayerAddress
	address.FromProto(req)
	q, err := s.queueOrErr()
	if err != nil {
		return nil, err
	}
	return &proto.GetPlayerStatusResponse{
		Status: s.getPlayerStatusLocked(q, address),
	}, nil
}

func (s *Service) GetPlayerToken(req *proto.GetPlayerTokenRequest) (*proto.GetPlayerTokenResponse, error) {
	userToken, err := db.GetPlayerToken(s.ctx, req.Id)
	if err != nil {
		log.Error("GetPlayerToken failed, err: ", err)
		return nil, err
	}
	return conversion.DbUserTokenToProtoGetPlayerTokenResponse(userToken), nil
}

func (s *Service) GetTimeoutConfig() (*proto.TimeoutConfig, error) {
	cfg := &proto.TimeoutConfig{
		ConfirmationTimeout:         config.GameParams.ConfirmationTimeout,
		CommitmentSubmissionTimeout: config.GameParams.CommitmentSubmissionTimeout,
		CardSubmissionTimeout:       config.GameParams.CardSubmissionTimeout,
		GameContinueTimeout:         config.GameParams.GameContinueTimeout,
	}
	return cfg, nil
}

func (s *Service) SubmitPlayerCommitment(req *proto.SubmitPlayerCommitmentRequest) error {
	return s.gameManager.HandleSubmitPlayerCommitment(req)
}

func (s *Service) SubmitPlayerCard(req *proto.SubmitPlayerCardRequest) error {
	return s.gameManager.HandleSubmitPlayerCard(req)
}

// --- rpc/server.PubSub PlayerManager ---

func (s *Service) AddPlayer(address types.PlayerAddress) error {
	s.playerMu.Lock()
	if _, ok := s.connected[address]; ok {
		s.playerMu.Unlock()
		return errors.New("player already exists: " + (&address).String())
	}
	s.connected[address] = struct{}{}
	s.playerMu.Unlock()
	if err := s.gameManager.SyncGamePhase(address); err != nil {
		s.playerMu.Lock()
		delete(s.connected, address)
		s.playerMu.Unlock()
		return err
	}
	return nil
}

func (s *Service) RemovePlayer(address types.PlayerAddress) {
	s.playerMu.Lock()
	delete(s.connected, address)
	s.playerMu.Unlock()
	_ = s.ExitQueue(address.ToProto())
}

func (s *Service) AddBotPlayer(address types.PlayerAddress) error {
	if err := s.AddPlayer(address); err != nil {
		return err
	}
	q, err := s.queueOrErr()
	if err != nil {
		return err
	}
	return q.RegisterBots(&address)
}

func (s *Service) RemoveBotPlayer(address types.PlayerAddress) {
	s.playerMu.Lock()
	delete(s.connected, address)
	q := s.queue
	s.playerMu.Unlock()
	if q != nil {
		_ = q.UnregisterBots(&address)
	}
}
