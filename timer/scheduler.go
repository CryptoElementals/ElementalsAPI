package timer

import (
	"fmt"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/hibiken/asynq"
)

var (
	lobbyAsynqScheduler *asynq.Scheduler
	botDispatchCronID   string
)

// initAsynqScheduler starts the shared hibiken/asynq process scheduler for
// periodic lobby jobs. It is invoked from InitTimer(ScopeLobby) when Redis is
// available. The scheduler uses a separate Redis client from the timer Client.
func initAsynqScheduler(redisOpt asynq.RedisClientOpt) {
	if lobbyAsynqScheduler != nil {
		return
	}
	s := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
		Logger:   &asynqLogger{},
		LogLevel: asynq.WarnLevel,
	})
	if err := s.Start(); err != nil {
		log.Fatalw("asynq lobby scheduler failed to start", "err", err)
		return
	}
	lobbyAsynqScheduler = s
}

func shutdownLobbyAsynqScheduler() {
	s := lobbyAsynqScheduler
	if s == nil {
		return
	}
	if err := UnregisterBotDispatchRecurring(); err != nil {
		log.Errorw("unregister bot dispatch cron on scheduler shutdown", "err", err)
	}
	lobbyAsynqScheduler = nil
	s.Shutdown()
}

// RegisterBotDispatchRecurring registers a single repeating cron job that enqueues
// queue bot-dispatch work on @every <period> (using robfig/cron "every" syntax).
// It replaces any previous bot-dispatch registration. Call after InitTimer(ScopeLobby).
func RegisterBotDispatchRecurring(period time.Duration, evt TimerEvent) error {
	if period <= 0 || evt == nil {
		return nil
	}
	s := lobbyAsynqScheduler
	if s == nil {
		return fmt.Errorf("register bot dispatch: lobby asynq scheduler is not started")
	}
	cronSpec := fmt.Sprintf("@every %s", period.String())
	task := asynq.NewTask(evt.EventType(), evt.Marshal())
	uniqueTTL := period
	if uniqueTTL < time.Second {
		uniqueTTL = time.Second
	}
	opts := []asynq.Option{
		asynq.Queue(queueName(ScopeLobby)),
		asynq.MaxRetry(0),
		asynq.Unique(uniqueTTL),
	}
	if botDispatchCronID != "" {
		if err := s.Unregister(botDispatchCronID); err != nil {
			log.Errorw("unregister previous bot dispatch cron", "err", err)
		}
		botDispatchCronID = ""
	}
	eid, err := s.Register(cronSpec, task, opts...)
	if err != nil {
		return err
	}
	botDispatchCronID = eid
	return nil
}

// UnregisterBotDispatchRecurring removes the scheduled bot-dispatch job if any
// (e.g. on queue stop).
func UnregisterBotDispatchRecurring() error {
	eid := botDispatchCronID
	botDispatchCronID = ""
	if eid == "" {
		return nil
	}
	s := lobbyAsynqScheduler
	if s == nil {
		return nil
	}
	if err := s.Unregister(eid); err != nil {
		return err
	}
	return nil
}
