package svc

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type Publisher interface {
	GoErrCloser
	Publish(v interface{})
}

type MockPublisher struct {
	mock.Mock
}

func (mock *MockPublisher) Go(ctx context.Context) {
	mock.Called(ctx)
}

func (mock *MockPublisher) Errors() <-chan error {
	args := mock.Called()
	return args.Get(0).(<-chan error)
}

func (mock *MockPublisher) Close(ctx context.Context) {
	mock.Called(ctx)
}

func (mock *MockPublisher) Publish(v interface{}) {
	mock.Called(v)
}
