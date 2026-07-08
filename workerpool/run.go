package workerpool

import (
	"context"
)

func Run(ctx context.Context, opts Options, tasks ...Task) error {
	p, err := New(opts)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if err = p.Submit(ctx, task); err != nil {
			p.Close()
			_ = p.Shutdown(ctx)
			return err
		}
	}

	p.Close()
	if err = p.Shutdown(ctx); err != nil {
		return err
	}
	return p.Err()
}

func Map[T any, R any](
	ctx context.Context,
	opts Options,
	items []T,
	fn func(context.Context, T) (R, error),
) ([]R, error) {
	if fn == nil {
		return nil, ErrNilTask
	}

	results := make([]R, len(items))
	tasks := make([]Task, 0, len(items))

	for i, item := range items {
		i, item := i, item
		tasks = append(tasks, func(ctx context.Context) error {
			result, err := fn(ctx, item)
			if err != nil {
				return err
			}
			results[i] = result
			return nil
		})
	}

	if err := Run(ctx, opts, tasks...); err != nil {
		return results, err
	}
	return results, nil
}
