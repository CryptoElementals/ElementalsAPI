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
}

type Queuer interface {
	HandleJoinQueueEvent(event *types.JoinQueueEvent)
	HandleExitQueueEvent(event *types.ExitQueueEvent)
	IsPlayerInQueue(playerAddress types.PlayerAddress) bool
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
	if _, ok := s.players[address]; ok {
		return errors.New("player already exists: " + address.String())
	}

	player := NewPlayer(s.ctx, address, s.pub, s.workerManager, s.queue)
	s.players[address] = player
	return nil
}

func (s *Service) RemovePlayer(address types.PlayerAddress) {
	s.lock.Lock()
	defer s.lock.Unlock()
	player := s.players[address]
	if player == nil {
		return
	}
	s.workerManager.CloseWorker(player.address.String())
	delete(s.players, address)
}

func (s *Service) JoinQueue(address types.PlayerAddress) error {
	player := s.GetOrCreatePlayer(address)
	return player.joinQueue()
}

func (s *Service) ExitQueue(address types.PlayerAddress) error {
	player := s.GetOrCreatePlayer(address)
	return player.exitQueue()
}

func (s *Service) GetOrCreatePlayer(address types.PlayerAddress) *Player {
	s.lock.Lock()
	defer s.lock.Unlock()
	player, ok := s.players[address]
	if !ok {
		player = NewPlayer(s.ctx, address, s.pub, s.workerManager, s.queue)
		s.players[address] = player
	}
	return player
}

func (s *Service) SyncPlayerInfo(address types.PlayerAddress) error {
	player := s.GetOrCreatePlayer(address)

	// we need to lock player here for better consistency
	player.lock.Lock()
	defer player.lock.Unlock()
	gameInfo := s.gameInfoGetter.GetActiveGameInfo(player.address)
	if gameInfo == nil {
		return nil
	}
	return nil
}

func (s *Service) IsPlayerInQueue(address types.PlayerAddress) bool {
	return s.queue.IsPlayerInQueue(address)
}

func (s *Service) ConfirmBattle(address types.PlayerAddress, gameID uint, roundNum uint32) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	player, ok := s.players[address]
	if !ok {
		return errors.New("player not found")
	}
	s.workerManager.SendEvent(fmt.Sprint(gameID), types.NewEvent(player.address.String(), &types.PlayerReadyEvent{
		GameId:        gameID,
		RoundNumber:   roundNum,
		PlayerAddress: address,
	}))
	return nil
}

func (s *Service) ContinueGame(address types.PlayerAddress, gameID uint) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	player, ok := s.players[address]
	if !ok {
		return errors.New("player not found")
	}
	s.workerManager.SendEvent(fmt.Sprint(gameID), types.NewEvent(player.address.String(), &types.PlayerContinueEvent{
		GameId:        gameID,
		PlayerAddress: address,
	}))
	return nil
}

// GetGamePhase implements server.PlayerRequestHandler.
func (s *Service) GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	p, ok := s.players[address]
	if !ok {
		return nil, errors.New("player not found")
	}
	return p.ToGamePhase(), nil
}
