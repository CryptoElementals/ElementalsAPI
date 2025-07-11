package worker

import (
	"context"
	"math"
)

type Status uint16
type WorkerType uint16

const (
	EXIT Status = math.MaxUint16
)

type Worker struct {
	ctx      context.Context
	Id       string
	Type     WorkerType
	handlers map[Status]func(*Worker) error
}

func (w *Worker) Run() {
	
}
