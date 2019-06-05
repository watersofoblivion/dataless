package svc

import (
	"context"
	"sync"
)

type ErrorCollector struct {
	errors chan error
	close  chan struct{}
	done   chan struct{}
	wg     *sync.WaitGroup
}

func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make(chan error),
		close:  make(chan struct{}),
		done:   make(chan struct{}),
		wg:     new(sync.WaitGroup),
	}
}

func (collector *ErrorCollector) Go(ctx context.Context) {
	<-collector.close
	collector.wg.Wait()
	collector.done <- struct{}{}
}

func (collector *ErrorCollector) Collect(errors <-chan error) {
	collector.wg.Add(1)
	go func() {
		defer collector.wg.Done()

		for err := range errors {
			collector.errors <- err
		}
	}()
}

func (collector *ErrorCollector) Errors() <-chan error {
	return collector.errors
}

func (collector *ErrorCollector) Close(ctx context.Context) {
	close(collector.close)
	defer close(collector.errors)

	select {
	case <-ctx.Done():
	case <-collector.done:
	}
}
