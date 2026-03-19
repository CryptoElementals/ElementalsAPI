package timer

import (
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
)

type Timer struct{}

func InitTimer() {}

type TimerEvent interface {
	EventType() string
	Marshal() []byte
	Unmarshal([]byte) error
	String() string
}

var handlersMap map[string]func(TimerEvent) error = map[string]func(TimerEvent) error{}
var handlersLock sync.RWMutex

func RegisterHandler(evt TimerEvent, handler func(TimerEvent) error) error {
	if evt == nil {
		return fmt.Errorf("register timer handler: event is nil")
	}
	if handler == nil {
		return fmt.Errorf("register timer handler: handler is nil")
	}
	eventType := evt.EventType()
	if eventType == "" {
		return fmt.Errorf("register timer handler: event type is empty")
	}

	handlersLock.Lock()
	defer handlersLock.Unlock()
	handlersMap[eventType] = handler
	return nil
}

func ProcessIn(duration time.Duration, evt TimerEvent) error {
	if evt == nil {
		return fmt.Errorf("process timer: event is nil")
	}
	eventType := evt.EventType()
	if eventType == "" {
		return fmt.Errorf("process timer: event type is empty")
	}

	handlersLock.RLock()
	handler, ok := handlersMap[eventType]
	handlersLock.RUnlock()
	if !ok {
		return fmt.Errorf("process timer: handler not found for event type %q", eventType)
	}

	time.AfterFunc(duration, func() {
		err := handler(evt)
		if err != nil {
			log.Errorw("handle timer event failed", "event", evt)
			return
		}
		log.Debugw("handle timer event success", "event", evt)
	})
	return nil
}
