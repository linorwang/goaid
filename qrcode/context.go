package qrcode

import (
	"context"
	"time"
)

type invalidContext struct{}

func (invalidContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (invalidContext) Done() <-chan struct{}       { return nil }
func (invalidContext) Err() error                  { return ErrInvalidOption }
func (invalidContext) Value(any) any               { return nil }

func checkedContext(ctx context.Context) context.Context {
	if ctx == nil {
		return invalidContext{}
	}
	return ctx
}
