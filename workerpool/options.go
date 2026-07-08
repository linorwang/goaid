package workerpool

import (
	"fmt"
	"runtime"
	"time"
)

type SubmitMode int

const (
	// SubmitBlock blocks until the task is accepted, the submit context is done,
	// or the pool is closed.
	SubmitBlock SubmitMode = iota
	// SubmitReject rejects immediately when the queue is full.
	SubmitReject
	// SubmitTimeout waits for SubmitTimeout before rejecting a full queue.
	SubmitTimeout
)

type Options struct {
	// Name is optional and only used for diagnostics exposed by callers.
	Name string

	// Workers is kept for simple fixed-size pools. When MinWorkers is not set,
	// Workers is used as MinWorkers.
	Workers int
	// MinWorkers is the minimum number of live workers.
	MinWorkers int
	// MaxWorkers is the maximum number of live workers. When MaxWorkers equals
	// MinWorkers, dynamic scaling is disabled.
	MaxWorkers int
	// QueueSize is the maximum number of waiting tasks.
	QueueSize int
	// IdleTimeout controls when extra idle workers above MinWorkers retire.
	IdleTimeout time.Duration

	SubmitMode    SubmitMode
	SubmitTimeout time.Duration
	TaskTimeout   time.Duration

	// StopOnError cancels tasks that have not started yet after the first task
	// returns an error or panics.
	StopOnError bool

	// RecoverPanic controls task panic recovery. It defaults to true unless
	// DisablePanicRecovery is set.
	RecoverPanic         bool
	DisablePanicRecovery bool

	OnError       func(error)
	OnPanic       func(any, []byte)
	OnTaskPanic   func(any, []byte)
	OnWorkerPanic func(any, []byte)
	OnReject      func(error)
}

func normalizeOptions(opts Options) (Options, error) {
	minWorkers := opts.MinWorkers
	if minWorkers == 0 {
		minWorkers = opts.Workers
	}
	if minWorkers == 0 {
		minWorkers = runtime.NumCPU()
	}
	if minWorkers < 0 {
		return Options{}, fmt.Errorf("%w: workers must be greater than or equal to 0", ErrInvalidOptions)
	}

	maxWorkers := opts.MaxWorkers
	if maxWorkers == 0 {
		maxWorkers = minWorkers
	}
	if maxWorkers < minWorkers {
		return Options{}, fmt.Errorf("%w: max workers must be greater than or equal to min workers", ErrInvalidOptions)
	}

	opts.MinWorkers = minWorkers
	opts.MaxWorkers = maxWorkers
	if opts.QueueSize < 0 {
		return Options{}, fmt.Errorf("%w: queue size must be greater than or equal to 0", ErrInvalidOptions)
	}
	if opts.QueueSize == 0 {
		opts.QueueSize = opts.MaxWorkers
	}
	if opts.MaxWorkers > opts.MinWorkers && opts.IdleTimeout <= 0 {
		opts.IdleTimeout = time.Minute
	}
	if opts.IdleTimeout < 0 {
		return Options{}, fmt.Errorf("%w: idle timeout must be greater than or equal to 0", ErrInvalidOptions)
	}
	switch opts.SubmitMode {
	case SubmitBlock, SubmitReject, SubmitTimeout:
	default:
		return Options{}, fmt.Errorf("%w: unsupported submit mode %d", ErrInvalidOptions, opts.SubmitMode)
	}
	if opts.SubmitMode == SubmitTimeout && opts.SubmitTimeout <= 0 {
		return Options{}, fmt.Errorf("%w: submit timeout must be greater than 0", ErrInvalidOptions)
	}
	if opts.TaskTimeout < 0 {
		return Options{}, fmt.Errorf("%w: task timeout must be greater than or equal to 0", ErrInvalidOptions)
	}
	if opts.DisablePanicRecovery {
		opts.RecoverPanic = false
	} else {
		opts.RecoverPanic = true
	}
	opts.Workers = opts.MinWorkers
	return opts, nil
}
