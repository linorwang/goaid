# workerpool

`workerpool` 是一个面向业务任务的协程池组件，核心目标不是复用 goroutine，而是限制并发、保护资源、隔离 panic、统一收集错误并提供可观测状态。

适合场景：

- 批量 HTTP / RPC / 数据库 / K8s API 调用
- 批量文件处理、对象存储上传、通知发送
- 后台服务中的异步任务消费
- 需要控制并发、队列背压和优雅关闭的业务任务

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/linorwang/goaid/workerpool"
)

func main() {
	ctx := context.Background()

	pool := workerpool.MustNew(workerpool.Options{
		Name:        "example",
		MinWorkers: 2,
		MaxWorkers: 8,
		QueueSize:  100,
		IdleTimeout: time.Minute,
	})
	defer pool.Shutdown(ctx)

	err := pool.Submit(ctx, func(ctx context.Context) error {
		fmt.Println("task running")
		return nil
	})
	if err != nil {
		panic(err)
	}

	pool.Wait()
}
```

## 核心设计

### 并发控制

`MinWorkers` 表示最小 worker 数，`MaxWorkers` 表示最大 worker 数。

```go
pool := workerpool.MustNew(workerpool.Options{
	MinWorkers: 4,
	MaxWorkers: 16,
	QueueSize:  1000,
})
```

如果只想使用固定大小的池，可以只配置 `Workers`：

```go
pool := workerpool.MustNew(workerpool.Options{
	Workers:   8,
	QueueSize: 1000,
})
```

`Workers` 会被当作 `MinWorkers`，并且默认 `MaxWorkers = MinWorkers`，此时不会动态伸缩。

### 动态伸缩

动态伸缩只在配置了 `MaxWorkers > MinWorkers` 时启用。

扩容规则：

```text
队列长度 > 当前 worker 数 && 当前 worker 数 < MaxWorkers
```

缩容规则：

```text
队列为空 && worker 空闲超过 IdleTimeout && 当前 worker 数 > MinWorkers
```

扩缩容是轻量启发式控制，不追求每一瞬间都精确等于最优 worker 数。并发提交时可能短暂多创建 worker，但不会超过 `MaxWorkers`。

示例：

```go
pool := workerpool.MustNew(workerpool.Options{
	MinWorkers:  2,
	MaxWorkers:  32,
	QueueSize:   5000,
	IdleTimeout: 30 * time.Second,
})
```

### 任务类型

所有任务都接收 `context.Context`：

```go
type Task func(ctx context.Context) error
```

任务应主动监听 `ctx.Done()`，这样才能响应取消、超时和 `StopOnError`。

```go
err := pool.Submit(ctx, func(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return doWork(ctx)
	}
})
```

## 提交模式

### Submit

`Submit` 按 `Options.SubmitMode` 工作，默认是阻塞提交。

```go
err := pool.Submit(ctx, task)
```

### TrySubmit

队列满时立即返回 `ErrQueueFull`。

```go
err := pool.TrySubmit(ctx, task)
if errors.Is(err, workerpool.ErrQueueFull) {
	// 降级、重试或返回上游限流
}
```

### SubmitWithTimeout

队列满时最多等待指定时间。

```go
err := pool.SubmitWithTimeout(ctx, task, 200*time.Millisecond)
```

也可以通过配置让 `Submit` 默认使用超时模式：

```go
pool := workerpool.MustNew(workerpool.Options{
	Workers:       8,
	QueueSize:     100,
	SubmitMode:    workerpool.SubmitTimeout,
	SubmitTimeout: 200 * time.Millisecond,
})
```

## Panic 隔离

默认开启 panic recover。业务任务 panic 不会导致服务崩溃，会被记录成 `*workerpool.PanicError`。

```go
err := pool.Submit(ctx, func(ctx context.Context) error {
	panic("array index out of range")
})
```

panic 分两类：

```go
workerpool.TaskPanic   // 业务任务 panic
workerpool.WorkerPanic // worker 内部逻辑或 hook panic
```

任务 panic 时会先记录为 `TaskPanic`，然后调用 `OnTaskPanic`。如果同时配置了兼容 hook `OnPanic`，它也会被调用。hook 自身 panic 会被隔离为 `WorkerPanic`。

可以通过 `PanicError.Kind` 判断：

```go
for _, err := range pool.Errors() {
	var panicErr *workerpool.PanicError
	if errors.As(err, &panicErr) {
		switch panicErr.Kind {
		case workerpool.TaskPanic:
			// 业务任务 panic
		case workerpool.WorkerPanic:
			// worker 或 hook panic
		}
	}
}
```

如果确实希望关闭 panic recover：

```go
pool := workerpool.MustNew(workerpool.Options{
	Workers:              8,
	DisablePanicRecovery: true,
})
```

生产环境不建议关闭。

## 错误处理

任务返回 error 后，协程池会记录错误并增加失败计数。

```go
err := pool.Submit(ctx, func(ctx context.Context) error {
	return errors.New("business failed")
})
```

获取全部错误：

```go
errs := pool.Errors()
```

获取合并后的错误：

```go
err := pool.Err()
```

遇到第一个错误后取消后续任务：

```go
pool := workerpool.MustNew(workerpool.Options{
	Workers:     8,
	QueueSize:   100,
	StopOnError: true,
})
```

注意：`StopOnError` 是协作式取消，已经运行中的任务需要自己监听 `ctx.Done()`。

## 日志接入

`workerpool` 不直接依赖项目内的 `logger` 包，而是通过 hook 接入日志，避免底层并发组件强绑定具体日志实现。

```go
pool := workerpool.MustNew(workerpool.Options{
	Workers:   8,
	QueueSize: 1000,
	OnError: func(err error) {
		logger.Error("workerpool task error", logger.Field{Key: "error", Value: err})
	},
	OnTaskPanic: func(v any, stack []byte) {
		logger.Error(
			"workerpool task panic",
			logger.Field{Key: "panic", Value: v},
			logger.Field{Key: "stack", Value: string(stack)},
		)
	},
	OnWorkerPanic: func(v any, stack []byte) {
		logger.Error(
			"workerpool worker panic",
			logger.Field{Key: "panic", Value: v},
			logger.Field{Key: "stack", Value: string(stack)},
		)
	},
	OnReject: func(err error) {
		logger.Warn("workerpool task rejected", logger.Field{Key: "error", Value: err})
	},
})
```

hook 自身发生 panic 时，也会被协程池隔离并记录为 `WorkerPanic`。

## 统计指标

```go
stats := pool.Stats()
```

字段说明：

| 字段 | 含义 |
| --- | --- |
| `Name` | 池名称 |
| `Workers` | 当前 worker 数 |
| `MinWorkers` | 最小 worker 数 |
| `MaxWorkers` | 最大 worker 数 |
| `Running` | 当前正在执行的任务数 |
| `Queued` | 当前队列中的任务数 |
| `Completed` | 已完成任务数 |
| `Failed` | 失败任务数 |
| `Rejected` | 被拒绝任务数 |
| `PanicCount` | panic 总数 |
| `TaskPanicCount` | 业务任务 panic 数 |
| `WorkerPanicCount` | worker 或 hook panic 数 |

这些指标可以很方便地接入 Prometheus、日志或内部监控平台。

## 优雅关闭

等待当前已提交任务完成：

```go
pool.Wait()
```

带超时或取消的等待：

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

err := pool.WaitContext(ctx)
```

停止接收新任务并等待已提交任务完成：

```go
pool.Close()

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := pool.Shutdown(ctx)
```

常见写法：

```go
defer pool.Shutdown(context.Background())
```

`Shutdown` 会等待已提交到池中的任务执行完，或者等待传入的 `ctx` 超时。

## 批量执行

### Run

```go
err := workerpool.Run(ctx, workerpool.Options{
	Workers: 8,
}, task1, task2, task3)
```

### Map

`Map` 适合输入集合到输出集合的批处理，返回结果顺序与输入顺序一致。

```go
results, err := workerpool.Map(ctx, workerpool.Options{
	Workers: 8,
}, []int{1, 2, 3}, func(ctx context.Context, item int) (int, error) {
	return item * 2, nil
})
```

## 选项说明

| 选项 | 说明 |
| --- | --- |
| `Name` | 池名称，用于统计和日志 |
| `Workers` | 固定大小池的简化配置 |
| `MinWorkers` | 最小 worker 数 |
| `MaxWorkers` | 最大 worker 数 |
| `QueueSize` | 等待队列大小 |
| `IdleTimeout` | 动态缩容的空闲时间 |
| `SubmitMode` | `Submit` 的默认提交模式 |
| `SubmitTimeout` | `SubmitTimeout` 模式的默认等待时间 |
| `TaskTimeout` | 单个任务最大执行时间 |
| `StopOnError` | 遇到错误后取消后续任务 |
| `DisablePanicRecovery` | 是否关闭 panic recover |
| `OnError` | 任务错误 hook |
| `OnTaskPanic` | 业务任务 panic hook |
| `OnWorkerPanic` | worker 或 hook panic hook |
| `OnReject` | 提交被拒绝 hook |

## 错误类型

```go
var (
	ErrClosed         = errors.New("workerpool: pool is closed")
	ErrInvalidOptions = errors.New("workerpool: invalid options")
	ErrNilTask        = errors.New("workerpool: nil task")
	ErrQueueFull      = errors.New("workerpool: queue is full")
)
```

建议使用 `errors.Is` / `errors.As` 判断错误。

```go
if errors.Is(err, workerpool.ErrQueueFull) {
	// 队列满
}
```

## 使用建议

- `QueueSize` 不要无限大，队列也是内存压力来源。
- IO 型任务可以配置较高的 `MaxWorkers`，CPU 型任务应更保守。
- 生产环境建议保持 panic recover 开启。
- `StopOnError` 依赖任务主动响应 `context`。
- 日志、监控、告警通过 hook 接入，不建议在通用组件中硬编码日志实现。
