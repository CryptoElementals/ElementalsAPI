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

// Scope selects a dedicated Asynq queue and handler registry so lobby and room
// processes never share the same delayed-task queue.
type Scope string

const (
	ScopeLobby Scope = "lobby"
	ScopeRoom  Scope = "room"
)

type TimerEvent interface {
	EventType() string
	Marshal() []byte
	Unmarshal([]byte) error
	String() string
}

type scopeRegistry struct {
	handlers   map[string]func(TimerEvent) error
	prototypes map[string]TimerEvent
}

func newScopeRegistry() *scopeRegistry {
	return &scopeRegistry{
		handlers:   map[string]func(TimerEvent) error{},
		prototypes: map[string]TimerEvent{},
	}
}

var (
	registries = map[Scope]*scopeRegistry{
		ScopeLobby: newScopeRegistry(),
		ScopeRoom:  newScopeRegistry(),
	}
	handlersLock sync.RWMutex

	client    *asynq.Client
	servers   = map[Scope]*asynq.Server{}
	serversMu sync.Mutex
)

func queueName(s Scope) string {
	return "timer:" + string(s)
}

// InitTimer starts an Asynq worker for the given scope's queue. Production binaries
// typically call this once with ScopeLobby or ScopeRoom; tests may call both scopes
// in-process (one worker per scope).
// When Redis is unavailable, delayed tasks fall back to in-process timers (no workers).
func InitTimer(scope Scope) {
	cfg := redis.GetConfig()
	if cfg == nil {
		log.Warn("timer: redis config not available, falling back to in-process timers")
		return
	}

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Address,
		Password: cfg.Password,
	}

	serversMu.Lock()
	defer serversMu.Unlock()

	if client == nil {
		client = asynq.NewClient(redisOpt)
	}

	if servers[scope] != nil {
		return
	}

	q := queueName(scope)
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues:      map[string]int{q: 1},
		Logger:      &asynqLogger{},
		LogLevel:    asynq.WarnLevel,
	})
	servers[scope] = srv

	go func(sc Scope, s *asynq.Server) {
		if err := s.Run(asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			return handleTaskForScope(sc, ctx, t)
		})); err != nil {
			log.Errorw("asynq server stopped", "scope", sc, "err", err)
		}
	}(scope, srv)
}

// StopTimer shuts down the Asynq worker for the given scope and closes the shared
// Redis client when no scope workers remain.
func StopTimer(scope Scope) {
	serversMu.Lock()
	defer serversMu.Unlock()
	if srv := servers[scope]; srv != nil {
		srv.Shutdown()
		delete(servers, scope)
	}
	if len(servers) == 0 && client != nil {
		client.Close()
		client = nil
	}
}

func RegisterHandler(scope Scope, evt TimerEvent, handler func(TimerEvent) error) error {
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

	reg := registries[scope]
	if reg == nil {
		return fmt.Errorf("register timer handler: unknown scope %q", scope)
	}

	handlersLock.Lock()
	defer handlersLock.Unlock()
	reg.handlers[eventType] = handler
	reg.prototypes[eventType] = evt
	return nil
}

func ProcessIn(scope Scope, duration time.Duration, evt TimerEvent) error {
	if evt == nil {
		return fmt.Errorf("process timer: event is nil")
	}
	eventType := evt.EventType()
	if eventType == "" {
		return fmt.Errorf("process timer: event type is empty")
	}

	reg := registries[scope]
	if reg == nil {
		return fmt.Errorf("process timer: unknown scope %q", scope)
	}

	handlersLock.RLock()
	_, ok := reg.handlers[eventType]
	handlersLock.RUnlock()
	if !ok {
		return fmt.Errorf("process timer: handler not found for event type %q in scope %q", eventType, scope)
	}

	if client != nil {
		task := asynq.NewTask(eventType, evt.Marshal())
		_, err := client.Enqueue(task,
			asynq.ProcessIn(duration),
			asynq.Queue(queueName(scope)),
			asynq.MaxRetry(0),
		)
		return err
	}

	handlersLock.RLock()
	handler := reg.handlers[eventType]
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

func handleTaskForScope(scope Scope, _ context.Context, t *asynq.Task) error {
	eventType := t.Type()

	reg := registries[scope]
	if reg == nil {
		return fmt.Errorf("no registry for scope %q", scope)
	}

	handlersLock.RLock()
	handler, ok := reg.handlers[eventType]
	proto, protoOk := reg.prototypes[eventType]
	handlersLock.RUnlock()

	if !ok || !protoOk {
		log.Errorw("asynq: no handler for task type", "scope", scope, "type", eventType)
		return fmt.Errorf("no handler for task type %q in scope %q", eventType, scope)
	}

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
