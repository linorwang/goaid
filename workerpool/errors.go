package workerpool

import (
	"errors"
	"fmt"
)

var (
	ErrClosed         = errors.New("workerpool: pool is closed")
	ErrInvalidOptions = errors.New("workerpool: invalid options")
	ErrNilTask        = errors.New("workerpool: nil task")
	ErrQueueFull      = errors.New("workerpool: queue is full")
)

type PanicKind string

const (
	TaskPanic   PanicKind = "task"
	WorkerPanic PanicKind = "worker"
)

type PanicError struct {
	Kind  PanicKind
	Value any
	Stack []byte
}

func (e *PanicError) Error() string {
	if e.Kind == "" {
		return fmt.Sprintf("workerpool: task panic: %v", e.Value)
	}
	return fmt.Sprintf("workerpool: %s panic: %v", e.Kind, e.Value)
}
