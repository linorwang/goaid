package workerpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolSubmitAndWait(t *testing.T) {
	p := MustNew(Options{Workers: 2, QueueSize: 4})
	defer func() {
		if err := p.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
	}()

	var count atomic.Int64
	for i := 0; i < 10; i++ {
		if err := p.Submit(context.Background(), func(context.Context) error {
			count.Add(1)
			return nil
		}); err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	p.Wait()
	if got := count.Load(); got != 10 {
		t.Fatalf("count = %d, want 10", got)
	}
	if stats := p.Stats(); stats.Completed != 10 || stats.Failed != 0 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestTrySubmitRejectsWhenQueueFull(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{})
	p := MustNew(Options{Workers: 1, QueueSize: 1})
	defer func() {
		_ = p.Shutdown(context.Background())
	}()
	defer close(block)

	if err := p.Submit(context.Background(), func(context.Context) error {
		close(started)
		<-block
		return nil
	}); err != nil {
		t.Fatalf("submit running task: %v", err)
	}
	<-started
	if err := p.TrySubmit(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("submit queued task: %v", err)
	}
	if err := p.TrySubmit(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("third submit error = %v, want %v", err, ErrQueueFull)
	}
	if stats := p.Stats(); stats.Rejected != 1 {
		t.Fatalf("rejected = %d, want 1", stats.Rejected)
	}
}

func TestSubmitWithTimeout(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{})
	p := MustNew(Options{Workers: 1, QueueSize: 1})
	defer func() {
		_ = p.Shutdown(context.Background())
	}()
	defer close(block)

	_ = p.Submit(context.Background(), func(context.Context) error {
		close(started)
		<-block
		return nil
	})
	<-started
	_ = p.Submit(context.Background(), func(context.Context) error { return nil })

	err := p.SubmitWithTimeout(context.Background(), func(context.Context) error { return nil }, 10*time.Millisecond)
	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("submit error = %v, want %v", err, ErrQueueFull)
	}
}

func TestPanicRecovery(t *testing.T) {
	var seen atomic.Bool
	p := MustNew(Options{
		Workers:   1,
		QueueSize: 1,
		OnPanic: func(any, []byte) {
			seen.Store(true)
		},
	})
	defer func() {
		_ = p.Shutdown(context.Background())
	}()

	if err := p.Submit(context.Background(), func(context.Context) error {
		panic("boom")
	}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	p.Wait()

	if !seen.Load() {
		t.Fatal("panic hook was not called")
	}
	stats := p.Stats()
	if stats.PanicCount != 1 || stats.TaskPanicCount != 1 || stats.WorkerPanicCount != 0 || stats.Failed != 1 {
		t.Fatalf("stats = %+v", stats)
	}
	panicErr, ok := p.Errors()[0].(*PanicError)
	if !ok {
		t.Fatalf("error type = %T, want *PanicError", p.Errors()[0])
	}
	if panicErr.Kind != TaskPanic {
		t.Fatalf("panic kind = %s, want %s", panicErr.Kind, TaskPanic)
	}
}

func TestWorkerPanicIsTrackedSeparately(t *testing.T) {
	var seen atomic.Bool
	p := MustNew(Options{
		Workers:   1,
		QueueSize: 1,
		OnPanic: func(any, []byte) {
			panic("panic hook failed")
		},
		OnWorkerPanic: func(any, []byte) {
			seen.Store(true)
		},
	})
	defer func() {
		_ = p.Shutdown(context.Background())
	}()

	if err := p.Submit(context.Background(), func(context.Context) error {
		panic("business panic")
	}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	p.Wait()
	waitUntil(t, time.Second, func() bool {
		return seen.Load()
	})

	stats := p.Stats()
	if stats.TaskPanicCount != 1 || stats.WorkerPanicCount != 1 || stats.PanicCount != 2 {
		t.Fatalf("stats = %+v", stats)
	}
	var hasTaskPanic, hasWorkerPanic bool
	for _, err := range p.Errors() {
		panicErr, ok := err.(*PanicError)
		if !ok {
			continue
		}
		hasTaskPanic = hasTaskPanic || panicErr.Kind == TaskPanic
		hasWorkerPanic = hasWorkerPanic || panicErr.Kind == WorkerPanic
	}
	if !hasTaskPanic || !hasWorkerPanic {
		t.Fatalf("panic errors = %#v", p.Errors())
	}
}

func TestTaskTimeout(t *testing.T) {
	p := MustNew(Options{Workers: 1, QueueSize: 1, TaskTimeout: 10 * time.Millisecond})
	defer func() {
		_ = p.Shutdown(context.Background())
	}()

	if err := p.Submit(context.Background(), func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	p.Wait()
	if !errors.Is(p.Err(), context.DeadlineExceeded) {
		t.Fatalf("pool err = %v, want deadline exceeded", p.Err())
	}
}

func TestWaitContextTimeout(t *testing.T) {
	block := make(chan struct{})
	p := MustNew(Options{Workers: 1, QueueSize: 1})
	defer func() {
		close(block)
		_ = p.Shutdown(context.Background())
	}()

	if err := p.Submit(context.Background(), func(context.Context) error {
		<-block
		return nil
	}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := p.WaitContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait context error = %v, want deadline exceeded", err)
	}
}

func TestStopOnErrorCancelsQueuedTaskContext(t *testing.T) {
	p := MustNew(Options{Workers: 1, QueueSize: 2, StopOnError: true})
	defer func() {
		_ = p.Shutdown(context.Background())
	}()

	want := errors.New("first task failed")
	if err := p.Submit(context.Background(), func(context.Context) error {
		return want
	}); err != nil {
		t.Fatalf("submit first task: %v", err)
	}
	if err := p.Submit(context.Background(), func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatalf("submit second task: %v", err)
	}

	p.Wait()
	if !errors.Is(p.Err(), want) {
		t.Fatalf("pool err = %v, want first task error", p.Err())
	}
	if !errors.Is(p.Err(), context.Canceled) {
		t.Fatalf("pool err = %v, want context canceled", p.Err())
	}
}

func TestShutdownWaitsForQueuedTasks(t *testing.T) {
	p := MustNew(Options{Workers: 2, QueueSize: 4})
	var count atomic.Int64

	for i := 0; i < 6; i++ {
		if err := p.Submit(context.Background(), func(context.Context) error {
			time.Sleep(time.Millisecond)
			count.Add(1)
			return nil
		}); err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
	if got := count.Load(); got != 6 {
		t.Fatalf("count = %d, want 6", got)
	}
	if err := p.Submit(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrClosed) {
		t.Fatalf("submit after close error = %v, want %v", err, ErrClosed)
	}
}

func TestShutdownTimeoutCanBeRetried(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{})
	p := MustNew(Options{Workers: 1, QueueSize: 1})

	if err := p.Submit(context.Background(), func(context.Context) error {
		close(started)
		<-block
		return nil
	}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := p.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown error = %v, want deadline exceeded", err)
	}

	close(block)
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("retry shutdown failed: %v", err)
	}
}

func TestRunCollectsErrors(t *testing.T) {
	want := errors.New("bad task")
	err := Run(context.Background(), Options{Workers: 2}, func(context.Context) error {
		return nil
	}, func(context.Context) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("run error = %v, want %v", err, want)
	}
}

func TestMapKeepsOrder(t *testing.T) {
	got, err := Map(context.Background(), Options{Workers: 3}, []int{1, 2, 3}, func(_ context.Context, item int) (int, error) {
		return item * 2, nil
	})
	if err != nil {
		t.Fatalf("map failed: %v", err)
	}
	want := []int{2, 4, 6}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestConcurrentSubmit(t *testing.T) {
	p := MustNew(Options{Workers: 4, QueueSize: 16})
	defer func() {
		_ = p.Shutdown(context.Background())
	}()

	var count atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.Submit(context.Background(), func(context.Context) error {
				count.Add(1)
				return nil
			}); err != nil {
				t.Errorf("submit failed: %v", err)
			}
		}()
	}

	wg.Wait()
	p.Wait()
	if got := count.Load(); got != 64 {
		t.Fatalf("count = %d, want 64", got)
	}
}

func TestConcurrentSubmitAndShutdown(t *testing.T) {
	p := MustNew(Options{MinWorkers: 1, MaxWorkers: 4, QueueSize: 16})
	var wg sync.WaitGroup

	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = p.Submit(context.Background(), func(context.Context) error {
				time.Sleep(time.Millisecond)
				return nil
			})
		}()
	}

	time.Sleep(time.Millisecond)
	shutdownErr := p.Shutdown(context.Background())
	wg.Wait()

	if shutdownErr != nil {
		t.Fatalf("shutdown failed: %v", shutdownErr)
	}
	if got := p.Stats().Workers; got != 0 {
		t.Fatalf("workers = %d, want 0 after shutdown", got)
	}
}

func TestDynamicScalingGrowAndShrink(t *testing.T) {
	block := make(chan struct{})
	p := MustNew(Options{
		MinWorkers:  1,
		MaxWorkers:  4,
		QueueSize:   32,
		IdleTimeout: 20 * time.Millisecond,
	})
	defer func() {
		_ = p.Shutdown(context.Background())
	}()

	for i := 0; i < 12; i++ {
		if err := p.Submit(context.Background(), func(context.Context) error {
			<-block
			return nil
		}); err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	waitUntil(t, time.Second, func() bool {
		stats := p.Stats()
		return stats.Workers > stats.MinWorkers && stats.Workers <= stats.MaxWorkers
	})

	close(block)
	p.Wait()

	waitUntil(t, time.Second, func() bool {
		return p.Stats().Workers == 1
	})
}

func TestDynamicScalingDoesNotExceedMaxWorkers(t *testing.T) {
	block := make(chan struct{})
	p := MustNew(Options{
		MinWorkers:  1,
		MaxWorkers:  3,
		QueueSize:   128,
		IdleTimeout: time.Second,
	})
	defer func() {
		close(block)
		_ = p.Shutdown(context.Background())
	}()

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = p.Submit(context.Background(), func(context.Context) error {
				<-block
				return nil
			})
		}()
	}
	wg.Wait()

	if got := p.Stats().Workers; got > 3 {
		t.Fatalf("workers = %d, want <= 3", got)
	}
}

func waitUntil(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("condition was not met within %s", timeout)
}
