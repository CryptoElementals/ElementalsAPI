package queue

import (
	"context"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type Service struct {
	ctx   context.Context
	queue *Queue
}

func NewService(ctx context.Context, workerManager *worker.WorkerManager, queueCache cache.Cache) *Service {
	return &Service{
		ctx:   ctx,
		queue: NewQueue(ctx, workerManager, queueCache),
	}
}

func (s *Service) Start() error {
	return s.queue.start()
}

func (s *Service) IsPlayerInQueue(address types.PlayerAddress) bool {
	return s.queue.isPlayerInQueue(address)
}
