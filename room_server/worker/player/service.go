package player

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type GameInfoGetter interface {
	GetActiveGameInfo(playerAddress types.PlayerAddress) *proto.GameInfo
	GetPlayerGameInfo(playerAddress types.PlayerAddress) proto.PlayerStatus
	GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error)
}

type Queuer interface {
	HandleJoinQueueEvent(event *types.JoinQueueEvent) error
	HandleExitQueueEvent(event *types.ExitQueueEvent) error
	HandleContinueGameEvent(event *types.PlayerContinueEvent) error
	IsPlayerInQueue(playerAddress types.PlayerAddress) bool
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

func (s *Service) addPlayer(address types.PlayerAddress) error {
	if _, ok := s.players[address]; ok {
		return errors.New("player already exists: " + address.String())
	}

	player := NewPlayer(s.ctx, address, s.pub, s.workerManager, s.queue)
	if s.queue.IsPlayerInQueue(address) {
		player.status = proto.PlayerStatus_PLAYER_IN_QUEUE
	} else {
		player.status = s.gameInfoGetter.GetPlayerGameInfo(address)
	}
	s.players[address] = player
	player.createSelf()
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

func (s *Service) JoinQueue(address types.PlayerAddress) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	player, ok := s.players[address]
	if !ok {
		return errors.New("player is not subscribing")
	}
	return player.joinQueue()
}

func (s *Service) ExitQueue(address types.PlayerAddress) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	player, ok := s.players[address]
	if !ok {
		return errors.New("player is not subscribing")
	}
	return player.exitQueue()
}

func (s *Service) IsPlayerInQueue(address types.PlayerAddress) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.queue.IsPlayerInQueue(address)
}

func (s *Service) ConfirmBattle(address types.PlayerAddress, gameID uint, roundNum uint32) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	player, ok := s.players[address]
	if !ok {
		return errors.New("player is not subscribing")
	}
	evt := types.NewEvent(player.address.String(), &types.PlayerReadyEvent{
		GameId:        gameID,
		RoundNumber:   roundNum,
		PlayerAddress: address,
	}, true)
	s.workerManager.SendEvent(fmt.Sprint(gameID), evt)
	err := evt.Await()
	return err
}

func (s *Service) ContinueGame(address types.PlayerAddress, gameID uint) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok := s.players[address]
	if !ok {
		return errors.New("player is not subscribing")
	}
	return s.queue.HandleContinueGameEvent(&types.PlayerContinueEvent{
		GameId:        gameID,
		PlayerAddress: address,
	})
}

func (s *Service) RefuseContinueGame(address types.PlayerAddress, gameID uint) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok := s.players[address]
	if !ok {
		return errors.New("player is not subscribing")
	}
	return s.queue.RefuseContinueGame(address, gameID)
}

func (s *Service) Surrender(address types.PlayerAddress, gameID uint) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	player, ok := s.players[address]
	if !ok {
		return errors.New("player is not subscribing")
	}
	if player.status != proto.PlayerStatus_PLAYER_IN_GAME {
		return errors.New("player not in game")
	}
	evt := types.NewEvent(player.address.String(), &types.SurrenderEvent{
		GameID:  gameID,
		Address: address,
	}, true)
	s.workerManager.SendEvent(fmt.Sprint(gameID), evt)
	err := evt.Await()
	return err
}

func (s *Service) GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var status proto.PlayerStatus
	player, ok := s.players[address]
	if ok {
		status = player.status
	} else {
		if s.queue.IsPlayerInQueue(address) {
			status = proto.PlayerStatus_PLAYER_IN_QUEUE
		} else {
			status = s.gameInfoGetter.GetPlayerGameInfo(address)
		}
	}
	switch status {
	case proto.PlayerStatus_PLAYER_IN_GAME, proto.PlayerStatus_PLAYER_MATCHED:
		return s.gameInfoGetter.GetGamePhase(address)
	case proto.PlayerStatus_PLAYER_IN_QUEUE:
		return &proto.GamePhase{
			GameType: proto.GameType_PVP,
			PvPInfo: &proto.PvPInfo{
				Status: proto.PlayerStatus_PLAYER_IN_QUEUE,
			},
		}, nil
	case proto.PlayerStatus_PLAYER_UNKNOWN:
		return &proto.GamePhase{
			GameType: proto.GameType_PVP,
			PvPInfo: &proto.PvPInfo{
				Status: proto.PlayerStatus_PLAYER_UNKNOWN,
			},
		}, nil
	}
	return nil, fmt.Errorf("unknonw player status, %s", status)
}
