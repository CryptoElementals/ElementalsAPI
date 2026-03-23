package timer

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/hibiken/asynq"
)

type TimerEvent interface {
	EventType() string
	Marshal() []byte
	Unmarshal([]byte) error
	String() string
}

var (
	handlersMap  = map[string]func(TimerEvent) error{}
	// prototypes stores a zero-value instance of each registered event type
	// so the asynq handler can deserialize the payload back into the correct Go type.
	prototypes   = map[string]TimerEvent{}
	handlersLock sync.RWMutex

	client *asynq.Client
	server *asynq.Server
)

func InitTimer() {
	cfg := redis.GetConfig()
	if cfg == nil {
		log.Warn("timer: redis config not available, falling back to in-process timers")
		return
	}

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Address,
		Password: cfg.Password,
	}

	client = asynq.NewClient(redisOpt)

	server = asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues:      map[string]int{"timer": 1},
		Logger:      &asynqLogger{},
		LogLevel:    asynq.WarnLevel,
	})

	go func() {
		if err := server.Run(asynq.HandlerFunc(handleTask)); err != nil {
			log.Errorw("asynq server stopped", "err", err)
		}
	}()
}

func StopTimer() {
	if server != nil {
		server.Shutdown()
	}
	if client != nil {
		client.Close()
	}
}

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
	prototypes[eventType] = evt
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
	_, ok := handlersMap[eventType]
	handlersLock.RUnlock()
	if !ok {
		return fmt.Errorf("process timer: handler not found for event type %q", eventType)
	}

	// If asynq client is available, enqueue via Redis; otherwise fall back to in-process.
	if client != nil {
		task := asynq.NewTask(eventType, evt.Marshal())
		_, err := client.Enqueue(task,
			asynq.ProcessIn(duration),
			asynq.Queue("timer"),
			asynq.MaxRetry(0),
		)
		return err
	}

	// Fallback: in-process timer (no Redis)
	handlersLock.RLock()
	handler := handlersMap[eventType]
	handlersLock.RUnlock()

	time.AfterFunc(duration, func() {
		if err := handler(evt); err != nil {
			log.Errorw("handle timer event failed", "event", evt)
			return
		}
		log.Debugw("handle timer event success", "event", evt)
	})
	return nil
}

// handleTask is the catch-all asynq handler. It uses the task type (= EventType)
// to look up the registered handler and prototype, deserializes, and invokes.
func handleTask(_ context.Context, t *asynq.Task) error {
	eventType := t.Type()

	handlersLock.RLock()
	handler, ok := handlersMap[eventType]
	proto, protoOk := prototypes[eventType]
	handlersLock.RUnlock()

	if !ok || !protoOk {
		log.Errorw("asynq: no handler for task type", "type", eventType)
		return fmt.Errorf("no handler for task type %q", eventType)
	}

	// Create a fresh instance of the same concrete type via reflection.
	fresh := reflect.New(reflect.TypeOf(proto).Elem()).Interface().(TimerEvent)
	if err := fresh.Unmarshal(t.Payload()); err != nil {
		return fmt.Errorf("unmarshal timer event %q: %w", eventType, err)
	}

	if err := handler(fresh); err != nil {
		log.Errorw("handle timer event failed", "event", fresh)
		return err
	}
	log.Debugw("handle timer event success", "event", fresh)
	return nil
}

// asynqLogger adapts asynq logging to our log package.
type asynqLogger struct{}

func (l *asynqLogger) Debug(args ...interface{}) { log.Debug(args...) }
func (l *asynqLogger) Info(args ...interface{})  { log.Info(args...) }
func (l *asynqLogger) Warn(args ...interface{})  { log.Warn(args...) }
func (l *asynqLogger) Error(args ...interface{}) { log.Error(args...) }
func (l *asynqLogger) Fatal(args ...interface{}) { log.Fatal(args...) }
