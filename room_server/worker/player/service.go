package player

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

type GameInfoGetter interface {
	GetActiveGameInfo(playerAddress *types.PlayerAddress) *proto.GameInfo
	GetPlayerGameInfo(playerAddress types.PlayerAddress) proto.PlayerStatus
}

type QueueInfoGetter interface {
	IsPlayerInQueue(playerAddress types.PlayerAddress) bool
}

type Service struct {
	ctx             context.Context
	lock            sync.RWMutex
	players         map[types.PlayerAddress]*Player
	pub             *server.PubSubServer
	workerManager   *worker.WorkerManager
	gameInfoGetter  GameInfoGetter
	queueInfoGetter QueueInfoGetter
}

func NewService(ctx context.Context,
	pub *server.PubSubServer,
	workerManager *worker.WorkerManager,
	gameInfoGetter GameInfoGetter,
	queueInfoGetter QueueInfoGetter) *Service {
	return &Service{
		ctx:             ctx,
		players:         make(map[types.PlayerAddress]*Player),
		pub:             pub,
		workerManager:   workerManager,
		gameInfoGetter:  gameInfoGetter,
		queueInfoGetter: queueInfoGetter,
	}
}

func (s *Service) AddPlayer(address types.PlayerAddress) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.players[address]; ok {
		return errors.New("player already exists")
	}

	player := NewPlayer(s.ctx, address, s.pub, s.workerManager)
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
		player = NewPlayer(s.ctx, address, s.pub, s.workerManager)
		s.players[address] = player
	}
	return player
}

func (s *Service) SyncPlayerInfo(address types.PlayerAddress) error {
	player := s.GetOrCreatePlayer(address)

	// we need to lock player here for better consistency
	player.lock.Lock()
	defer player.lock.Unlock()
	gameInfo := s.gameInfoGetter.GetActiveGameInfo(&player.address)
	if gameInfo == nil {
		return nil
	}
	return player.sync(gameInfo)
}

func (s *Service) IsPlayerInQueue(address types.PlayerAddress) bool {
	return s.queueInfoGetter.IsPlayerInQueue(address)
}

func (s *Service) SendPlayerReady(address types.PlayerAddress, gameID uint, roundNum uint32) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	player, ok := s.players[address]
	if !ok {
		return
	}
	s.workerManager.SendEvent(fmt.Sprint(gameID), types.NewEvent(player.address.String(), &types.PlayerReadyEvent{
		GameId:        gameID,
		RoundNumber:   roundNum,
		PlayerAddress: address,
	}))
}
