package logging

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/watersofoblivion/dataless/lib"
	"github.com/watersofoblivion/dataless/lib/bang"
)

type Logger interface {
	lib.GoErrCloser
	Log(v interface{})
}

type MockLogger struct {
	mock.Mock
}

func (mock *MockLogger) Go(ctx context.Context) {
	mock.Called(ctx)
}

func (mock *MockLogger) Errors() bang.Errors {
	args := mock.Called()
	return args.Get(0).(bang.Errors)
}

func (mock *MockLogger) Close(ctx context.Context) {
	mock.Called(ctx)
}

func (mock *MockLogger) Log(v interface{}) {
	mock.Called(v)
}
