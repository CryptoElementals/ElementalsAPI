package worker

import (
	"context"
	"testing"
	"time"

	"github.com/CryptoElementals/common/room_server/worker/types"
)

var globalTestWorkerManager *WorkerManager

type testEventHandler struct {
	evtChan chan *types.Event
	tt      *testing.T
}

func (h *testEventHandler) Handle(ctx context.Context, sender EventSender, event *types.Event) error {
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
	globalTestWorkerManager.SpwanWorker(context.Background(), "worker1", 1, h)

	// send event to worker1
	globalTestWorkerManager.SendEvent("worker1", types.NewEvent("sender", &types.PlayerReadyEvent{
		GameId:        1,
		RoundNumber:      1,
		PlayerAddress: types.PlayerAddress{WalletAddress: "player1", TemporaryAddress: "temp"},
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
	globalTestWorkerManager.SpwanWorker(context.Background(), "worker1", 1, h)
	// close worker1
	globalTestWorkerManager.CloseWorker("worker1")

	globalTestWorkerManager.SendEvent("worker1", types.NewEvent("sender", &types.PlayerReadyEvent{
		GameId:        1,
		RoundNumber:      1,
		PlayerAddress: types.PlayerAddress{WalletAddress: "player1", TemporaryAddress: "temp"},
	}))
	// check if worker1 is closed
	select {
	case evt := <-h.evtChan:
		t.Errorf("worker1 should be closed, but got event: %v", evt)
	default:
	}
}

func TestWorkerAckEventReceived(t *testing.T) {
	// setup two workers
	h1 := &testEventHandler{tt: t, evtChan: make(chan *types.Event, 1)}
	h2 := &testEventHandler{tt: t, evtChan: make(chan *types.Event, 1)}
	globalTestWorkerManager.SpwanWorker(context.Background(), "worker1", 1, h1)
	globalTestWorkerManager.SpwanWorker(context.Background(), "worker2", 1, h2)

	evt := types.NewEvent("worker1", &types.PlayerReadyEvent{
		GameId:        1,
		RoundNumber:      1,
		PlayerAddress: types.PlayerAddress{WalletAddress: "player1", TemporaryAddress: "temp"},
	}, true)
	id := evt.EventID
	// send a event and requires ack
	globalTestWorkerManager.SendEvent("worker2", evt)
	time.Sleep(1 * time.Millisecond)
	// check if worker2 receive the event
	select {
	case evt := <-h2.evtChan:
		if evt.Sender != "worker1" {
			t.Errorf("expected sender to be 'worker1', but got '%s'", evt.Sender)
		}
		_, ok := evt.Data.(*types.PlayerReadyEvent)
		if !ok {
			t.Errorf("expected data to be *types.PlayerReadyEvent, but got %T", evt.Data)
		}
	default:
		t.Errorf("cannot retrieve event from worker2")
	}
	time.Sleep(1 * time.Millisecond)
	// check if worker1 received the ack event
	select {
	case evt := <-h1.evtChan:
		if evt.Sender != "worker2" {
			t.Errorf("expected sender to be 'worker2', but got '%s'", evt.Sender)
		}
		ack, ok := evt.Data.(*types.AckEvent)
		if !ok {
			t.Errorf("expected data to be *types.PlayerReadyEvent, but got %T", evt.Data)
		}
		if ack.EventID != id {
			t.Errorf("expected EventID to be '%s', but got '%s'", id, ack.EventID)
		}
	default:
		t.Errorf("cannot retrieve event from worker1")
	}
}
