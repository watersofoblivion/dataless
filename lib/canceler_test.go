package lib

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCanceler(t *testing.T) {
	ctx, canceler := NewCanceler(context.Background())

	canceler.Timeout = 50 * time.Millisecond
	go canceler.Go(context.Background())

	select {
	case <-ctx.Done():
		assert.Fail(t, "context canceled before timeout elapsed")
	case <-time.After(canceler.Timeout / 2):
	}

	select {
	case <-time.After(canceler.Timeout * 2):
		assert.Fail(t, "context not canceled after timeout")
	case <-ctx.Done():
	}

	t.Run("respects cancel of parent", func(t *testing.T) {
		parent, cancel := context.WithCancel(context.Background())
		ctx, canceler := NewCanceler(context.Background())

		canceler.Timeout = 100 * time.Millisecond
		go canceler.Go(parent)

		cancel()
		select {
		case <-time.After(canceler.Timeout / 2):
			assert.Fail(t, "canceler not canceling on parent cancel")
		case <-ctx.Done():
		}
	})
}
