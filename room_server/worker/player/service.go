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
	HandleJoinQueueEvent(event *types.JoinQueueEvent) error
	HandleExitQueueEvent(event *types.ExitQueueEvent) error
	HandleContinueGameEvent(event *types.PlayerContinueEvent) error
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
	if s.queue.IsPlayerInQueue(address) {
		player.status = proto.PlayerStatus_PLAYER_IN_QUEUE
	} else {
		player.status = s.gameInfoGetter.GetPlayerGameInfo(address)
	}
	s.players[address] = player
	player.createSelf()
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
	s.lock.RLock()
	defer s.lock.RUnlock()
	player, ok := s.players[address]
	if !ok {
		return errors.New("player not found")
	}
	return player.joinQueue()
}

func (s *Service) ExitQueue(address types.PlayerAddress) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	player, ok := s.players[address]
	if !ok {
		return errors.New("player not found")
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
		return errors.New("player not found")
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
		return errors.New("player not found")
	}
	return s.queue.HandleContinueGameEvent(&types.PlayerContinueEvent{
		GameId:        gameID,
		PlayerAddress: address,
	})
}
