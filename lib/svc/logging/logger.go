package logging

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/watersofoblivion/dataless/lib/svc"
)

type Logger interface {
	svc.GoErrCloser
	Log(v interface{})
}

type MockLogger struct {
	mock.Mock
}

func (mock *MockLogger) Go(ctx context.Context) {
	mock.Called(ctx)
}

func (mock *MockLogger) Errors() <-chan error {
	args := mock.Called()
	return args.Get(0).(<-chan error)
}

func (mock *MockLogger) Close(ctx context.Context) {
	mock.Called(ctx)
}

func (mock *MockLogger) Log(v interface{}) {
	mock.Called(v)
}
