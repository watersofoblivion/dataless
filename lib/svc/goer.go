package svc

import "context"

type Goer interface {
	Go(ctx context.Context)
}

type Closer interface {
	Close(ctx context.Context)
}

type GoCloser interface {
	Goer
	Closer
}

type Errorer interface {
	Errors() <-chan error
}

type GoErrCloser interface {
	GoCloser
	Errorer
}
