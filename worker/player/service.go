package player

import (
	"context"
	"errors"
	"sync"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type GameInfoGetter interface {
	GetActiveGameInfo(playerAddress *types.PlayerAddress) (*proto.GameInfo, error)
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

func NewService(ctx context.Context) *Service {
	return &Service{
		ctx:     ctx,
		players: make(map[types.PlayerAddress]*Player),
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
	gameInfo, err := s.gameInfoGetter.GetActiveGameInfo(&player.address)
	if err != nil {
		return err
	}
	return player.sync(gameInfo)
}
