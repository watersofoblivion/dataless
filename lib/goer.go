package lib

import (
	"context"

	"github.com/watersofoblivion/dataless/lib/bang"
)

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

type GoErrCloser interface {
	GoCloser
	bang.Errorser
}
