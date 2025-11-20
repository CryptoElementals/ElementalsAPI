package player

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type GameInfoGetter interface {
	GetActiveGameInfo(playerAddress types.PlayerAddress) *proto.GameInfo
	GetPlayerGameInfo(playerAddress types.PlayerAddress) proto.PlayerStatus
	GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error)
	SyncGamePhase(address types.PlayerAddress) error
	GetBattleInfo(ctx context.Context, gameID uint32, roundNum uint32) (*proto.RoundResult, *proto.GameResult, error)
	HandleSubmitPlayerCommitment(evt *types.SubmitPlayerCommitment) error
	HandleSubmitPlayerCard(evt *types.SubmitPlayerCard) error
}

type Queuer interface {
	HandleJoinQueueEvent(event *types.JoinQueueEvent) error
	HandleExitQueueEvent(event *types.ExitQueueEvent) error
	HandleContinueGameEvent(event *types.PlayerContinueEvent) error
	IsPlayerInQueue(playerAddress types.PlayerAddress) bool
	GetPlayerContinueInfo(playerAddress types.PlayerAddress) *types.GameContinueInfo
	RefuseContinueGame(playerAddress types.PlayerAddress, gameID uint) error
	RegisterBots(addrs ...*types.PlayerAddress) error
	UnregisterBots(addrs ...*types.PlayerAddress) error
}

type Publisher interface {
	Publish(ctx context.Context, req *proto.PublishRequest) (*proto.PublishResponse, error)
}

type Service struct {
	ctx            context.Context
	lock           sync.RWMutex
	players        map[types.PlayerAddress]*Player
	pub            Publisher
	workerManager  *worker.WorkerManager
	gameInfoGetter GameInfoGetter
	queue          Queuer
}

func NewService(ctx context.Context,
	pub Publisher,
	workerManager *worker.WorkerManager,
	gameInfoGetter GameInfoGetter,
	queue Queuer) *Service {
	return &Service{
		ctx:            ctx,
		players:        make(map[types.PlayerAddress]*Player),
		pub:            pub,
		workerManager:  workerManager,
		gameInfoGetter: gameInfoGetter,
		queue:          queue,
	}
}

func (s *Service) AddPlayer(address types.PlayerAddress) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.addPlayer(address)
}

func (s *Service) RemovePlayer(address types.PlayerAddress) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.removePlayer(address)
}

func (s *Service) AddBotPlayer(address types.PlayerAddress) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	err := s.addPlayer(address)
	if err != nil {
		return err
	}
	s.queue.RegisterBots(&address)
	return nil
}

func (s *Service) RemoveBotPlayer(address types.PlayerAddress) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.removePlayer(address)
	s.queue.UnregisterBots(&address)
}

func (s *Service) JoinQueue(address types.PlayerAddress) error {
	status := s.getPlayerStatus(address)
	if status != proto.PlayerStatus_PLAYER_UNKNOWN {
		return fmt.Errorf("player cannot join queue, player status: %s", status)
	}
	return s.queue.HandleJoinQueueEvent(&types.JoinQueueEvent{
		PlayerAddress: address,
	})
}

func (s *Service) ExitQueue(address types.PlayerAddress) error {
	status := s.getPlayerStatus(address)
	if status != proto.PlayerStatus_PLAYER_IN_QUEUE {
		return nil
	}
	s.queue.HandleExitQueueEvent(&types.ExitQueueEvent{
		PlayerAddress: address,
	})
	return nil
}

func (s *Service) IsPlayerInQueue(address types.PlayerAddress) bool {
	return s.queue.IsPlayerInQueue(address)
}

func (s *Service) ConfirmBattle(address types.PlayerAddress, gameID uint, roundNum uint32, turnNum uint32) error {
	evt := types.NewEvent(address.String(), &types.PlayerReadyEvent{
		GameId:        gameID,
		RoundNumber:   roundNum,
		TurnNumber:    turnNum,
		PlayerAddress: address,
	}, true)
	s.workerManager.SendEvent(fmt.Sprint(gameID), evt)
	_, err := evt.Await()
	return err
}

func (s *Service) ContinueGame(address types.PlayerAddress, gameID uint) error {
	return s.queue.HandleContinueGameEvent(&types.PlayerContinueEvent{
		GameId:        gameID,
		PlayerAddress: address,
	})
}

func (s *Service) RefuseContinueGame(address types.PlayerAddress, gameID uint) error {
	return s.queue.RefuseContinueGame(address, gameID)
}

func (s *Service) Surrender(address types.PlayerAddress, gameID uint) error {
	evt := types.NewEvent(address.String(), &types.SurrenderEvent{
		GameID:  gameID,
		Address: address,
	}, true)
	s.workerManager.SendEvent(fmt.Sprint(gameID), evt)
	_, err := evt.Await()
	return err
}

func (s *Service) GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error) {
	status := s.getPlayerStatus(address)
	switch status {
	case proto.PlayerStatus_PLAYER_IN_GAME, proto.PlayerStatus_PLAYER_MATCHED:
		return s.gameInfoGetter.GetGamePhase(address)
	case proto.PlayerStatus_PLAYER_IN_QUEUE:
		return &proto.GamePhase{
			GameType: proto.GameType_PVP,
		}, nil
	case proto.PlayerStatus_PLAYER_UNKNOWN:
		// it might have a case that the player is not in any game, but in the continue queue
		continueInfo := s.queue.GetPlayerContinueInfo(address)
		if continueInfo != nil {
			players := make([]*proto.GamePhasePlayer, 0, len(continueInfo.Players))
			for _, playerInfo := range continueInfo.Players {
				addr := types.NewPlayerAddress(
					playerInfo.Id,
					playerInfo.TemporaryAddress,
				).ToProto()
				players = append(players, &proto.GamePhasePlayer{
					Address:    addr,
					TurnStatus: proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN,
				})
			}
			return &proto.GamePhase{
				GameType:    proto.GameType_PVP,
				GameID:      uint32(continueInfo.GameID),
				TurnStartAt: continueInfo.EndTime.Unix(),
				Players:     players,
			}, nil
		}
		return &proto.GamePhase{
			GameType: proto.GameType_PVP,
		}, nil
	}
	return nil, fmt.Errorf("unknonw player status, %s", status)
}

func (s *Service) GetBattleInfo(ctx context.Context, gameid uint32, roundNum uint32) (*proto.RoundResult, *proto.GameResult, error) {
	return s.gameInfoGetter.GetBattleInfo(ctx, gameid, roundNum)
}

func (s *Service) GetPlayerToken(playerId int64) (*proto.GetPlayerTokenResponse, error) {
	userToken, err := db.GetPlayerToken(s.ctx, playerId)
	if err != nil {
		log.Error("GetPlayerToken failed, err: ", err)
		return nil, err
	}
	return conversion.DbUserTokenToProtoGetPlayerTokenResponse(userToken), nil
}

func (s *Service) GetTimeoutConfig() (*proto.TimeoutConfig, error) {
	cfg := &proto.TimeoutConfig{
		GameMatchTimeout:    config.GameParams.GameMatchTimeout,
		RoundConfirmTimeout: config.GameParams.RoundConfirmTimeout,
		RoundTimeout:        config.GameParams.RoundTimeout,
		ContinueTimeout:     config.GameParams.ContinueTimeout,
	}
	return cfg, nil
}

// SubmitPlayerCommitment submits a player commitment
func (s *Service) SubmitPlayerCommitment(address types.PlayerAddress, roundNumber uint32, commitment []byte, commitmentIndex uint32, signature []byte, gameID uint) error {
	return s.gameInfoGetter.HandleSubmitPlayerCommitment(&types.SubmitPlayerCommitment{
		GameID:          gameID,
		Address:         address,
		RoundNumber:     roundNumber,
		Commitment:      commitment,
		CommitmentIndex: commitmentIndex,
		Signature:       signature,
	})
}

// SubmitPlayerCard submits a player card
func (s *Service) SubmitPlayerCard(address types.PlayerAddress, roundNumber uint32, salt []byte, card uint, cardIndex uint32, signature []byte, gameID uint) error {
	return s.gameInfoGetter.HandleSubmitPlayerCard(&types.SubmitPlayerCard{
		GameID:      gameID,
		Address:     address,
		RoundNumber: roundNumber,
		Salt:        salt,
		Card:        card,
		CardIndex:   cardIndex,
		Signature:   signature,
	})
}

func (s *Service) addPlayer(address types.PlayerAddress) error {
	if _, ok := s.players[address]; ok {
		return errors.New("player already exists: " + address.String())
	}

	player := NewPlayer(s.ctx, address, s.pub, s.workerManager)
	s.players[address] = player
	player.createSelf()
	// Sync game phase when player worker is created
	s.gameInfoGetter.SyncGamePhase(address)
	return nil
}

func (s *Service) removePlayer(address types.PlayerAddress) {
	player := s.players[address]
	if player == nil {
		return
	}
	s.workerManager.CloseWorker(player.address.String())
	delete(s.players, address)
}

func (s *Service) getPlayerStatus(address types.PlayerAddress) proto.PlayerStatus {
	if s.queue.IsPlayerInQueue(address) {
		return proto.PlayerStatus_PLAYER_IN_QUEUE
	} else {
		return s.gameInfoGetter.GetPlayerGameInfo(address)
	}
}
