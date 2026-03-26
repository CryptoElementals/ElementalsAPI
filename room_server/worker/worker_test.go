package worker

import (
	"context"
	"testing"
	"time"

	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

var globalTestWorkerManager *WorkerManager

type testEventHandler struct {
	evtChan chan *types.Event
	tt      *testing.T
}

func (h *testEventHandler) Handle(ctx context.Context, event *types.Event) error {
	h.evtChan <- event
	h.tt.Log(*event)
	return nil
}

func TestMain(m *testing.M) {
	globalTestWorkerManager = NewWorkerManager(context.Background())
	m.Run()
}

func TestWorkerManager_SendEvent(t *testing.T) {
	h := &testEventHandler{tt: t, evtChan: make(chan *types.Event, 1)}
	// setup a test worker
	globalTestWorkerManager.SpawnWorker(context.Background(), "worker1", 1, h)

	// send event to worker1
	globalTestWorkerManager.SendEvent("worker1", types.NewEvent("sender", &proto.SurrenderRequest{
		GameID:  1,
		Address: &proto.PlayerAddress{Id: 1, TemporaryAddress: "temp"},
	}))
	time.Sleep(1 * time.Millisecond)
	// check if worker1 received the event
	select {
	case evt := <-h.evtChan:
		if evt.Sender != "sender" {
			t.Errorf("expected sender to be 'sender', but got '%s'", evt.Sender)
		}
	default:
		t.Errorf("cannot retrieve event from worker1")
	}
}

func TestWorkerManager_CloseWorker(t *testing.T) {
	h := &testEventHandler{tt: t, evtChan: make(chan *types.Event, 1)}
	// setup a test worker
	globalTestWorkerManager.SpawnWorker(context.Background(), "worker1", 1, h)
	// close worker1
	globalTestWorkerManager.CloseWorker("worker1")

	globalTestWorkerManager.SendEvent("worker1", types.NewEvent("sender", &proto.SurrenderRequest{
		GameID:  1,
		Address: &proto.PlayerAddress{Id: 1, TemporaryAddress: "temp"},
	}))
	// check if worker1 is closed
	select {
	case evt := <-h.evtChan:
		t.Errorf("worker1 should be closed, but got event: %v", evt)
	default:
	}
}
