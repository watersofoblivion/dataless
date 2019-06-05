package bang

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorCollector(t *testing.T) {
	ctx := context.Background()

	collector := NewErrorCollector()
	go collector.Go(ctx)

	errorsOne := make(chan error)
	errorsTwo := make(chan error)

	errOne := fmt.Errorf("error-one")
	go func() {
		defer close(errorsOne)
		errorsOne <- errOne
	}()

	errTwo := fmt.Errorf("error-two")
	go func() {
		defer close(errorsTwo)
		errorsTwo <- errTwo
	}()

	collector.Collect(errorsOne)
	collector.Collect(errorsTwo)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		collector.Close(ctx)
	}()

	errors := make([]error, 0, 2)
	for err := range collector.Errors() {
		errors = append(errors, err)
	}
	assert.Len(t, errors, 2)
	assert.Contains(t, errors, errOne)
	assert.Contains(t, errors, errTwo)

	wg.Wait()
}
