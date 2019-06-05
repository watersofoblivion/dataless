package amzmock

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDynamoDB(t *testing.T) {
	ctx := context.Background()
	err := fmt.Errorf("the-error")

	ddb := new(DynamoDB)

	t.Run("PutItemWithContext", func(t *testing.T) {
		input := new(dynamodb.PutItemInput)
		output := new(dynamodb.PutItemOutput)

		ddb.On("PutItemWithContext", ctx, input).Return(output, err)
		out, e := ddb.PutItemWithContext(ctx, input)

		ddb.AssertExpectations(t)
		assert.Equal(t, output, out)
		assert.Equal(t, err, e)

		t.Run("with nil output", func(t *testing.T) {
			ddb := new(DynamoDB)

			ddb.On("PutItemWithContext", mock.Anything, mock.Anything).Return(nil, nil)
			assert.NotPanics(t, func() { out, e = ddb.PutItemWithContext(ctx, input) })
			ddb.AssertExpectations(t)
			assert.Nil(t, out)
			assert.Nil(t, e)
		})
	})

	t.Run("GetItemWithContext", func(t *testing.T) {
		input := new(dynamodb.GetItemInput)
		output := new(dynamodb.GetItemOutput)

		ddb.On("GetItemWithContext", ctx, input).Return(output, err)
		out, e := ddb.GetItemWithContext(ctx, input)

		ddb.AssertExpectations(t)
		assert.Equal(t, output, out)
		assert.Equal(t, err, e)

		t.Run("with nil output", func(t *testing.T) {
			ddb := new(DynamoDB)

			ddb.On("GetItemWithContext", mock.Anything, mock.Anything).Return(nil, nil)
			assert.NotPanics(t, func() { out, e = ddb.GetItemWithContext(ctx, input) })
			ddb.AssertExpectations(t)
			assert.Nil(t, out)
			assert.Nil(t, e)
		})
	})

	t.Run("DeleteItemWithContext", func(t *testing.T) {
		input := new(dynamodb.DeleteItemInput)
		output := new(dynamodb.DeleteItemOutput)

		ddb.On("DeleteItemWithContext", ctx, input).Return(output, err)
		out, e := ddb.DeleteItemWithContext(ctx, input)

		ddb.AssertExpectations(t)
		assert.Equal(t, output, out)
		assert.Equal(t, err, e)

		t.Run("with nil output", func(t *testing.T) {
			ddb := new(DynamoDB)

			ddb.On("DeleteItemWithContext", mock.Anything, mock.Anything).Return(nil, nil)
			assert.NotPanics(t, func() { out, e = ddb.DeleteItemWithContext(ctx, input) })
			ddb.AssertExpectations(t)
			assert.Nil(t, out)
			assert.Nil(t, e)
		})
	})

	t.Run("QueryWithContext", func(t *testing.T) {
		input := new(dynamodb.QueryInput)
		output := new(dynamodb.QueryOutput)

		ddb.On("QueryWithContext", ctx, input).Return(output, err)
		out, e := ddb.QueryWithContext(ctx, input)

		ddb.AssertExpectations(t)
		assert.Equal(t, output, out)
		assert.Equal(t, err, e)

		t.Run("with nil output", func(t *testing.T) {
			ddb := new(DynamoDB)

			ddb.On("QueryWithContext", mock.Anything, mock.Anything).Return(nil, nil)
			assert.NotPanics(t, func() { out, e = ddb.QueryWithContext(ctx, input) })
			ddb.AssertExpectations(t)
			assert.Nil(t, out)
			assert.Nil(t, e)
		})
	})
}
