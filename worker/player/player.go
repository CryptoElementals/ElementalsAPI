package player

import (
	"context"
	"fmt"
	"sync"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type Publisher interface {
	Publish(ctx context.Context, req *proto.PublishRequest) (*proto.PublishResponse, error)
}

type Player struct {
	ctx          context.Context
	address      types.PlayerAddress
	publisher    Publisher
	workerManger *worker.WorkerManager
	status       proto.PlayerStatus
	lock         sync.RWMutex
}

func NewPlayer(ctx context.Context, address types.PlayerAddress, publisher Publisher, workerManger *worker.WorkerManager) *Player {
	p := &Player{
		address:      address,
		publisher:    publisher,
		workerManger: workerManger,
	}
	p.createSelf()
	return p
}

func (p *Player) Handle(ctx context.Context, event *types.Event) error {
	switch event.EventType {
	case types.EVENT_TYPE_NEW_GAME:
		p.publisher.Publish(ctx, &proto.PublishRequest{
			Topic: p.address.String(),
			Event: &proto.Event{},
		})
		p.status = proto.PlayerStatus_PLAYER_IN_GAME
	}
	return nil
}

func (p *Player) createSelf() {
	p.workerManger.SpwanWorker(p.address.String(), types.WORKER_TYPE_PLAYER, p)
}

func (p *Player) joinQueue() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.status != proto.PlayerStatus_PLAYER_KNOWN {
		return fmt.Errorf("join queue failed, player status %s", p.status)
	}
	p.workerManger.SendEvent(types.QUEUE_MANAGER_ID, types.NewEvent(p.address.String(), types.EVENT_TYPE_JOIN_QUEUE, &types.JoinQueueEvent{
		PlayerAddress: p.address,
	}))
	p.status = proto.PlayerStatus_PLAYER_IN_QUEUE
	return nil
}

func (p *Player) exitQueue() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.status != proto.PlayerStatus_PLAYER_IN_QUEUE {
		return fmt.Errorf("join queue failed, player status %s", p.status)
	}
	p.workerManger.SendEvent(types.QUEUE_MANAGER_ID, types.NewEvent(p.address.String(), types.EVENT_TYPE_EXIT_QUEUE, &types.ExitQueueEvent{
		PlayerAddress: p.address,
	}))
	p.status = proto.PlayerStatus_PLAYER_KNOWN
	return nil
}
