package cron

import (
	"context"
	"time"
)

type TaskFunc func(ctx context.Context)

type Task struct {
	Action       string
	Func         TaskFunc
	IsPersistent bool
	Interval     time.Duration
	IsRunning    bool
}

var _factory = make(map[string]Task)

func RegisterTask(action string, task Task) {
	_factory[action] = task
}

func Register(action string, taskFunc TaskFunc, persistent bool, interval time.Duration) {
	task := Task{
		Action:       action,
		Func:         taskFunc,
		IsPersistent: persistent,
		Interval:     interval,
		IsRunning:    false,
	}
	_factory[action] = task
}

func GetAllTasks() []Task {
	tasks := make([]Task, 0)
	for _, t := range _factory {
		tasks = append(tasks, t)
	}
	return tasks
}

func Exist(action string) bool {
	_, ok := _factory[action]
	return ok
}
