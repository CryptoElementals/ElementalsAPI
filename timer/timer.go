package timer

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/DillLabs/asynq"
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

	client  *asynq.Client
	servers = map[Scope]*asynq.Server{}
	// asynqWorkerRunning records scopes for which StartTimer has already launched s.Run.
	asynqWorkerRunning = map[Scope]bool{}
	serversMu          sync.Mutex
)

func queueName(s Scope) string {
	return "timer:" + string(s)
}

// InitTimer creates the Asynq client, periodic scheduler, and Asynq Server for the scope
// but does not run the worker. Call StartTimer after handlers are registered and before
// opening your listener. When Redis is unavailable, it returns after logging (no client).
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

	initPeriodicAsynqScheduler(redisOpt)
	if servers[scope] != nil {
		return
	}

	q := queueName(scope)
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues:      map[string]int{q: 1},
		Logger:      &asynqLogger{},
		LogLevel:    asynq.WarnLevel,
		// Speed up retry handoff for crash simulation demos.
		RetryDelayFunc: func(n int, err error, t *asynq.Task) time.Duration {
			return 1 * time.Second
		},
		DelayedTaskCheckInterval: 1 * time.Second,
		LeaseDuration:            5 * time.Second,
		HeartbeatInterval:        1500 * time.Millisecond,
		RecovererInterval:        1 * time.Second,
		RecovererCutoff:          2 * time.Second,
	})
	servers[scope] = srv
}

// StartTimer runs the Asynq worker for the given scope in a goroutine (s.Run). It is
// a no-op if InitTimer was skipped (no Redis) or the worker for that scope is already
// running. Call once per started process after InitTimer for that scope.
func StartTimer(scope Scope) {
	serversMu.Lock()
	srv, ok := servers[scope]
	if !ok {
		serversMu.Unlock()
		if client != nil {
			log.Warnw("start timer: no asynq server; InitTimer for scope was not run", "scope", scope)
		}
		return
	}
	if asynqWorkerRunning[scope] {
		serversMu.Unlock()
		return
	}
	asynqWorkerRunning[scope] = true
	serversMu.Unlock()

	go func(sc Scope, s *asynq.Server) {
		if err := s.Run(asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			return handleTaskForScope(sc, ctx, t)
		})); err != nil {
			log.Errorw("asynq server stopped", "scope", sc, "err", err)
		}
		serversMu.Lock()
		delete(asynqWorkerRunning, sc)
		serversMu.Unlock()
	}(scope, srv)
}

// StopTimer shuts down the Asynq worker for the given scope and closes the shared
// Redis client when no scope workers remain.
func StopTimer(scope Scope) {
	serversMu.Lock()
	defer serversMu.Unlock()
	if scope == ScopeRoom {
		if err := UnregisterRoomChainTxPoolRecurring(); err != nil {
			log.Errorw("stop timer: unregister room chain tx pool cron", "err", err)
		}
	}
	if scope == ScopeLobby {
		shutdownPeriodicAsynqScheduler()
	}
	if srv := servers[scope]; srv != nil {
		srv.Shutdown()
		delete(servers, scope)
		delete(asynqWorkerRunning, scope)
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

// ProcessIn schedules evt to run after duration. When unique is true and the Redis
// asynq client is active, the task is enqueued with asynq.Unique so duplicate
// type+payload+queue within the uniqueness window are rejected (see asynq docs).
// Uniqueness TTL is max(duration, 1s) as required by asynq. ErrDuplicateTask is
// treated as success when unique is true. The in-process timer fallback does not
// enforce uniqueness.
func ProcessIn(scope Scope, duration time.Duration, evt TimerEvent, unique bool) error {
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
		opts := []asynq.Option{
			asynq.ProcessIn(duration),
			asynq.Queue(queueName(scope)),
			asynq.MaxRetry(0),
		}
		if unique {
			ttl := duration
			if ttl < time.Second {
				ttl = time.Second
			}
			opts = append(opts, asynq.Unique(ttl))
		}
		_, err := client.Enqueue(task, opts...)
		if err != nil && unique && errors.Is(err, asynq.ErrDuplicateTask) {
			return nil
		}
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
