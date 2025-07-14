package cron

import (
	"context"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
)

type Scheduler struct {
	tasks []Task
	mu    sync.Mutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) Register(name string, interval time.Duration, taskFunc TaskFunc, isPersistent bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = append(s.tasks, Task{Action: name, Interval: interval, Func: taskFunc, IsPersistent: isPersistent})
}

func (s *Scheduler) RegisterAllTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 注册匹配任务
	RegisterMatchmakingTask()

	// 注册对战处理任务
	RegisterBattleTask()

	// 从全局工厂获取所有已注册的任务
	s.tasks = append(s.tasks, GetAllTasks()...)
}

func (s *Scheduler) Start(ctx context.Context) {
	for i := range s.tasks {
		log.Debugf("run task %s", s.tasks[i].Action)
		go s.runTask(ctx, &s.tasks[i]) // 传递任务的指针
	}
}

func (s *Scheduler) runTask(ctx context.Context, task *Task) {
	if task.IsPersistent {
		task.Func(ctx)
		return
	}

	go func() {
		if !task.IsRunning {
			task.IsRunning = true
			defer func() {
				task.IsRunning = false
			}()
			task.Func(ctx)
		}
	}()

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go func() {
				if !task.IsRunning {
					task.IsRunning = true
					defer func() {
						task.IsRunning = false
					}()
					task.Func(ctx)
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}
