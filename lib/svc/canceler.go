package svc

import (
	"context"
	"time"
)

type Canceler struct {
	Timeout time.Duration
	cancel  func()
}

func NewCanceler(ctx context.Context) (context.Context, *Canceler) {
	ctx, cancel := context.WithCancel(ctx)
	return ctx, &Canceler{cancel: cancel}
}

func (canceler *Canceler) Go(ctx context.Context) {
	defer canceler.cancel()

	select {
	case <-ctx.Done():
	case <-time.After(canceler.Timeout):
	}
}
