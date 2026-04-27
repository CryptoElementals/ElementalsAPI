package timer

import (
	"fmt"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/hibiken/asynq"
)

var (
	// periodicAsynqScheduler is the shared hibiken/asynq.Scheduler used to run cron specs
	// that periodically enqueue work onto timer queues (one queue per scope).
	periodicAsynqScheduler  *asynq.Scheduler
	lobbyBotDispatchEntryID string
	lobbyTournamentEntryID  string
	roomChainTxPoolEntryID  string
)

// initPeriodicAsynqScheduler starts the shared scheduler. Invoked from InitTimer when Redis is
// available. Uses a separate Redis client from the timer Client.
func initPeriodicAsynqScheduler(redisOpt asynq.RedisClientOpt) {
	if periodicAsynqScheduler != nil {
		return
	}
	s := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
		Logger:   &asynqLogger{},
		LogLevel: asynq.WarnLevel,
	})
	if err := s.Start(); err != nil {
		log.Fatalw("asynq periodic scheduler failed to start", "err", err)
		return
	}
	periodicAsynqScheduler = s
}

func shutdownPeriodicAsynqScheduler() {
	if err := UnregisterBotDispatchRecurring(); err != nil {
		log.Errorw("unregister bot dispatch cron on scheduler shutdown", "err", err)
	}
	if err := UnregisterTournamentRecurring(); err != nil {
		log.Errorw("unregister tournament cron on scheduler shutdown", "err", err)
	}
	if err := UnregisterRoomChainTxPoolRecurring(); err != nil {
		log.Errorw("unregister room chain tx pool cron on scheduler shutdown", "err", err)
	}
	s := periodicAsynqScheduler
	if s == nil {
		return
	}
	periodicAsynqScheduler = nil
	s.Shutdown()
}

// registerRecurring enqueues a task to the given scope queue on @every <period>. Replaces
// a previous entry when entryID is non-empty (same logical job reregistered).
func registerRecurring(period time.Duration, evt TimerEvent, scope Scope, previousEntryID *string) error {
	if period <= 0 || evt == nil {
		return nil
	}
	s := periodicAsynqScheduler
	if s == nil {
		return fmt.Errorf("periodic asynq scheduler is not started")
	}
	cronSpec := fmt.Sprintf("@every %s", period.String())
	task := asynq.NewTask(evt.EventType(), evt.Marshal())
	uniqueTTL := period
	if uniqueTTL < time.Second {
		uniqueTTL = time.Second
	}
	opts := []asynq.Option{
		asynq.Queue(queueName(scope)),
		asynq.MaxRetry(0),
		asynq.Unique(uniqueTTL),
	}
	if *previousEntryID != "" {
		if err := s.Unregister(*previousEntryID); err != nil {
			log.Errorw("unregister previous periodic cron", "err", err)
		}
		*previousEntryID = ""
	}
	eid, err := s.Register(cronSpec, task, opts...)
	if err != nil {
		return err
	}
	*previousEntryID = eid
	return nil
}

func unregisterRecurring(entryID *string) error {
	eid := *entryID
	*entryID = ""
	if eid == "" {
		return nil
	}
	s := periodicAsynqScheduler
	if s == nil {
		return nil
	}
	if err := s.Unregister(eid); err != nil {
		return err
	}
	return nil
}

// RegisterBotDispatchRecurring registers a single repeating cron job that enqueues
// queue bot-dispatch work on @every <period> (using robfig/cron "every" syntax).
// It replaces any previous bot-dispatch registration. Call after InitTimer(ScopeLobby).
func RegisterBotDispatchRecurring(period time.Duration, evt TimerEvent) error {
	return registerRecurring(period, evt, ScopeLobby, &lobbyBotDispatchEntryID)
}

// UnregisterBotDispatchRecurring removes the scheduled bot-dispatch job if any
// (e.g. on queue stop).
func UnregisterBotDispatchRecurring() error {
	return unregisterRecurring(&lobbyBotDispatchEntryID)
}

// RegisterTournamentRecurring registers a repeating job that enqueues the given
// event on @every <period> to the lobby timer queue. Replaces the previous
// tournament registration.
func RegisterTournamentRecurring(period time.Duration, evt TimerEvent) error {
	return registerRecurring(period, evt, ScopeLobby, &lobbyTournamentEntryID)
}

// UnregisterTournamentRecurring removes the tournament cron, if any.
func UnregisterTournamentRecurring() error {
	return unregisterRecurring(&lobbyTournamentEntryID)
}

// RegisterRoomChainTxPoolRecurring registers a repeating job that enqueues the given event
// on @every <period> to the room timer queue. Replaces a previous room chain tx pool
// registration. Call after InitTimer(ScopeRoom) and RegisterHandler for the same event type.
func RegisterRoomChainTxPoolRecurring(period time.Duration, evt TimerEvent) error {
	return registerRecurring(period, evt, ScopeRoom, &roomChainTxPoolEntryID)
}

// UnregisterRoomChainTxPoolRecurring removes the room chain tx pool cron, if any.
func UnregisterRoomChainTxPoolRecurring() error {
	return unregisterRecurring(&roomChainTxPoolEntryID)
}
