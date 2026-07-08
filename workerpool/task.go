package workerpool

import "context"

// Task is a unit of work executed by Pool.
type Task func(ctx context.Context) error
