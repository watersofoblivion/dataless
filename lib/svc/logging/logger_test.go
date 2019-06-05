package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockLogger(t *testing.T) {
	ctx := context.Background()

	t.Run("Go", func(t *testing.T) {
		mock := new(MockLogger)
		mock.On("Go", ctx).Return()

		mock.Go(ctx)

		mock.AssertExpectations(t)
	})

	t.Run("Errors", func(t *testing.T) {
		errors := (<-chan error)(make(chan error))
		mock := new(MockLogger)
		mock.On("Errors").Return(errors)

		returned := mock.Errors()

		mock.AssertExpectations(t)
		assert.Equal(t, errors, returned)
	})

	t.Run("Close", func(t *testing.T) {
		mock := new(MockLogger)
		mock.On("Close", ctx).Return()

		mock.Close(ctx)

		mock.AssertExpectations(t)
	})

	t.Run("Publish", func(t *testing.T) {
		v := map[string]string{"foo": "bar"}

		mock := new(MockLogger)
		mock.On("Log", v).Return(nil)

		mock.Log(v)

		mock.AssertExpectations(t)
	})
}
