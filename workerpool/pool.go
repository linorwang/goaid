package workerpool

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type job struct {
	ctx  context.Context
	task Task
}

type Pool struct {
	opts Options

	tasks chan job
	done  chan struct{}

	closeOnce sync.Once
	closed    atomic.Bool

	workerMu   sync.Mutex
	workerCond *sync.Cond
	workers    int

	taskMu      sync.Mutex
	taskCond    *sync.Cond
	pendingTask int64

	stopCtx    context.Context
	stopCancel context.CancelFunc

	running          atomic.Int64
	completed        atomic.Int64
	failed           atomic.Int64
	rejected         atomic.Int64
	panicCount       atomic.Int64
	taskPanicCount   atomic.Int64
	workerPanicCount atomic.Int64

	errMu sync.Mutex
	errs  []error
}

func New(opts Options) (*Pool, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}

	stopCtx, stopCancel := context.WithCancel(context.Background())
	p := &Pool{
		opts:       normalized,
		tasks:      make(chan job, normalized.QueueSize),
		done:       make(chan struct{}),
		stopCtx:    stopCtx,
		stopCancel: stopCancel,
	}
	p.taskCond = sync.NewCond(&p.taskMu)
	p.workerCond = sync.NewCond(&p.workerMu)

	for i := 0; i < normalized.MinWorkers; i++ {
		p.startWorker()
	}
	return p, nil
}

func MustNew(opts Options) *Pool {
	p, err := New(opts)
	if err != nil {
		panic(err)
	}
	return p
}

func (p *Pool) Submit(ctx context.Context, task Task) error {
	switch p.opts.SubmitMode {
	case SubmitReject:
		return p.TrySubmit(ctx, task)
	case SubmitTimeout:
		return p.SubmitWithTimeout(ctx, task, p.opts.SubmitTimeout)
	default:
		return p.submit(ctx, task, 0, false)
	}
}

func (p *Pool) TrySubmit(ctx context.Context, task Task) error {
	return p.submit(ctx, task, 0, true)
}

func (p *Pool) SubmitWithTimeout(ctx context.Context, task Task, timeout time.Duration) error {
	if timeout <= 0 {
		return p.TrySubmit(ctx, task)
	}
	return p.submit(ctx, task, timeout, false)
}

func (p *Pool) Wait() {
	_ = p.WaitContext(context.Background())
}

func (p *Pool) WaitContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	p.taskMu.Lock()
	defer p.taskMu.Unlock()

	stop := context.AfterFunc(ctx, func() {
		p.taskMu.Lock()
		p.taskCond.Broadcast()
		p.taskMu.Unlock()
	})
	defer stop()

	for p.pendingTask > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		p.taskCond.Wait()
	}
	return nil
}

func (p *Pool) Close() {
	p.closeOnce.Do(func() {
		p.workerMu.Lock()
		p.closed.Store(true)
		p.workerMu.Unlock()
		close(p.done)
	})
}

func (p *Pool) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	p.Close()

	return p.waitWorkers(ctx)
}

func (p *Pool) Errors() []error {
	p.errMu.Lock()
	defer p.errMu.Unlock()

	errs := make([]error, len(p.errs))
	copy(errs, p.errs)
	return errs
}

func (p *Pool) Err() error {
	return errors.Join(p.Errors()...)
}

func (p *Pool) Stats() Stats {
	p.workerMu.Lock()
	workers := p.workers
	p.workerMu.Unlock()

	return Stats{
		Name:             p.opts.Name,
		Workers:          workers,
		MinWorkers:       p.opts.MinWorkers,
		MaxWorkers:       p.opts.MaxWorkers,
		Running:          p.running.Load(),
		Queued:           len(p.tasks),
		Completed:        p.completed.Load(),
		Failed:           p.failed.Load(),
		Rejected:         p.rejected.Load(),
		PanicCount:       p.panicCount.Load(),
		TaskPanicCount:   p.taskPanicCount.Load(),
		WorkerPanicCount: p.workerPanicCount.Load(),
	}
}

func (p *Pool) submit(ctx context.Context, task Task, timeout time.Duration, rejectWhenFull bool) error {
	if task == nil {
		return ErrNilTask
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if p.closed.Load() {
		return p.reject(ErrClosed)
	}

	j := job{ctx: ctx, task: task}
	p.addTask()

	var err error
	if rejectWhenFull {
		select {
		case p.tasks <- j:
			p.maybeGrow()
			return nil
		case <-ctx.Done():
			err = ctx.Err()
		case <-p.done:
			err = ErrClosed
		default:
			err = ErrQueueFull
		}
		p.finishTask()
		return p.reject(err)
	}

	var timer <-chan time.Time
	if timeout > 0 {
		t := time.NewTimer(timeout)
		defer t.Stop()
		timer = t.C
	}

	select {
	case p.tasks <- j:
		p.maybeGrow()
		return nil
	case <-ctx.Done():
		err = ctx.Err()
	case <-p.done:
		err = ErrClosed
	case <-timer:
		err = ErrQueueFull
	}

	p.finishTask()
	return p.reject(err)
}

func (p *Pool) reject(err error) error {
	p.rejected.Add(1)
	if p.opts.OnReject != nil {
		p.callHook(func() {
			p.opts.OnReject(err)
		}, true)
	}
	return err
}

func (p *Pool) worker() {
	counted := true
	defer func() {
		if v := recover(); v != nil {
			p.recordWorkerPanic(v, debug.Stack(), true)
		}
		if counted {
			p.finishWorker()
		}
		if !p.closed.Load() {
			p.ensureMinWorkers()
			p.maybeGrow()
		}
	}()

	for {
		idleTimer := p.idleTimer()
		select {
		case j := <-p.tasks:
			stopTimer(idleTimer)
			p.run(j)
		case <-p.done:
			stopTimer(idleTimer)
			for {
				select {
				case j := <-p.tasks:
					p.run(j)
				default:
					return
				}
			}
		case <-timerC(idleTimer):
			if p.tryRetireWorker() {
				counted = false
				return
			}
		}
	}
}

func (p *Pool) run(j job) {
	p.running.Add(1)
	defer p.running.Add(-1)
	defer p.finishTask()

	ctx, cancel := p.taskContext(j.ctx)
	defer cancel()

	if err := ctx.Err(); err != nil {
		p.recordError(err)
		return
	}

	if err := p.executeTask(ctx, j.task); err != nil {
		p.recordError(err)
		p.notifyTaskPanic(err)
		return
	}
	p.completed.Add(1)
}

func (p *Pool) executeTask(ctx context.Context, task Task) (err error) {
	if p.opts.RecoverPanic {
		defer func() {
			if v := recover(); v != nil {
				err = p.newTaskPanicError(v, debug.Stack())
			}
		}()
	}
	return task(ctx)
}

func (p *Pool) taskContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	cancel := context.CancelFunc(func() {})
	if p.opts.StopOnError {
		combined, combinedCancel := context.WithCancel(ctx)
		stop := context.AfterFunc(p.stopCtx, combinedCancel)
		ctx = combined
		cancel = func() {
			stop()
			combinedCancel()
		}
	}

	if p.opts.TaskTimeout > 0 {
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, p.opts.TaskTimeout)
		prevCancel := cancel
		ctx = timeoutCtx
		cancel = func() {
			timeoutCancel()
			prevCancel()
		}
	}
	return ctx, cancel
}

func (p *Pool) newTaskPanicError(value any, stack []byte) error {
	p.panicCount.Add(1)
	p.taskPanicCount.Add(1)
	return &PanicError{Kind: TaskPanic, Value: value, Stack: stack}
}

func (p *Pool) notifyTaskPanic(err error) {
	panicErr, ok := err.(*PanicError)
	if !ok || panicErr.Kind != TaskPanic {
		return
	}
	if p.opts.OnTaskPanic != nil {
		p.callHook(func() {
			p.opts.OnTaskPanic(panicErr.Value, panicErr.Stack)
		}, true)
	}
	if p.opts.OnPanic != nil {
		p.callHook(func() {
			p.opts.OnPanic(panicErr.Value, panicErr.Stack)
		}, true)
	}
}

func (p *Pool) recordWorkerPanic(value any, stack []byte, notify bool) {
	p.panicCount.Add(1)
	p.workerPanicCount.Add(1)
	err := &PanicError{Kind: WorkerPanic, Value: value, Stack: stack}

	p.errMu.Lock()
	p.errs = append(p.errs, err)
	p.errMu.Unlock()

	if notify && p.opts.OnWorkerPanic != nil {
		p.callHook(func() {
			p.opts.OnWorkerPanic(value, stack)
		}, false)
	}
	if notify && p.opts.OnError != nil {
		p.callHook(func() {
			p.opts.OnError(err)
		}, false)
	}
	if p.opts.StopOnError {
		p.stopCancel()
	}
}

func (p *Pool) recordError(err error) {
	if err == nil {
		return
	}
	p.failed.Add(1)
	p.errMu.Lock()
	p.errs = append(p.errs, err)
	p.errMu.Unlock()

	if p.opts.OnError != nil {
		p.callHook(func() {
			p.opts.OnError(err)
		}, true)
	}
	if p.opts.StopOnError {
		p.stopCancel()
	}
}

func (p *Pool) callHook(fn func(), notifyWorker bool) {
	defer func() {
		if v := recover(); v != nil {
			p.recordWorkerPanic(v, debug.Stack(), notifyWorker)
		}
	}()
	fn()
}

func (p *Pool) addTask() {
	p.taskMu.Lock()
	p.pendingTask++
	p.taskMu.Unlock()
}

func (p *Pool) finishTask() {
	p.taskMu.Lock()
	p.pendingTask--
	if p.pendingTask == 0 {
		p.taskCond.Broadcast()
	}
	p.taskMu.Unlock()
}

func (p *Pool) startWorker() bool {
	p.workerMu.Lock()
	defer p.workerMu.Unlock()

	if p.closed.Load() || p.workers >= p.opts.MaxWorkers {
		return false
	}
	p.workers++
	go p.worker()
	return true
}

func (p *Pool) ensureMinWorkers() {
	p.workerMu.Lock()
	defer p.workerMu.Unlock()

	for !p.closed.Load() && p.workers < p.opts.MinWorkers && p.workers < p.opts.MaxWorkers {
		p.workers++
		go p.worker()
	}
}

func (p *Pool) maybeGrow() {
	p.workerMu.Lock()
	defer p.workerMu.Unlock()

	for !p.closed.Load() && len(p.tasks) > p.workers && p.workers < p.opts.MaxWorkers {
		p.workers++
		go p.worker()
	}
}

func (p *Pool) tryRetireWorker() bool {
	p.workerMu.Lock()
	defer p.workerMu.Unlock()

	if len(p.tasks) != 0 || p.workers <= p.opts.MinWorkers {
		return false
	}
	p.workers--
	if len(p.tasks) > 0 {
		p.workers++
		return false
	}
	if p.workers == 0 {
		p.workerCond.Broadcast()
	}
	return true
}

func (p *Pool) finishWorker() {
	p.workerMu.Lock()
	p.workers--
	if p.workers == 0 {
		p.workerCond.Broadcast()
	}
	p.workerMu.Unlock()
}

func (p *Pool) waitWorkers(ctx context.Context) error {
	p.workerMu.Lock()
	defer p.workerMu.Unlock()

	stop := context.AfterFunc(ctx, func() {
		p.workerMu.Lock()
		p.workerCond.Broadcast()
		p.workerMu.Unlock()
	})
	defer stop()

	for p.workers > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		p.workerCond.Wait()
	}
	return nil
}

func (p *Pool) idleTimer() *time.Timer {
	if p.opts.MaxWorkers <= p.opts.MinWorkers || p.opts.IdleTimeout <= 0 {
		return nil
	}
	return time.NewTimer(p.opts.IdleTimeout)
}

func timerC(timer *time.Timer) <-chan time.Time {
	if timer == nil {
		return nil
	}
	return timer.C
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
