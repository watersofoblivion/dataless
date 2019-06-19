package advertising

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/watersofoblivion/sam/amz/amzmock"
)

func TestAdTrafficTable(t *testing.T) {
	ctx := context.Background()
	ad := uuid.New()
	start := time.Now()
	end := time.Now()
	page := map[string]*dynamodb.AttributeValue{
		"ad_id": new(dynamodb.AttributeValue).SetS(ad.String()),
		"day":   new(dynamodb.AttributeValue).SetS(time.Now().Format(DateFormatDay)),
	}
	limit := int64(10)

	tableName := "table-name"

	keyCondition := expression.Key("ad_id").Equal(expression.Value(ad.String())).
		And(expression.Key("day").Between(
			expression.Value(start.Format(DateFormatDay)),
			expression.Value(end.Format(DateFormatDay))))

	expr, err := expression.NewBuilder().
		WithKeyCondition(keyCondition).
		Build()
	require.NoError(t, err)

	input := new(dynamodb.QueryInput)
	input.SetTableName(tableName)
	input.SetExpressionAttributeNames(expr.Names())
	input.SetExpressionAttributeValues(expr.Values())
	input.SetKeyConditionExpression(aws.StringValue(expr.KeyCondition()))
	input.SetExclusiveStartKey(page)
	input.SetLimit(limit)

	items := []map[string]*dynamodb.AttributeValue{
		{
			"day":         new(dynamodb.AttributeValue).SetS(time.Now().Format(DateFormatDay)),
			"impressions": new(dynamodb.AttributeValue).SetN(fmt.Sprintf("%d", 4)),
			"clicks":      new(dynamodb.AttributeValue).SetN(fmt.Sprintf("%d", 3)),
		},
		{
			"day":         new(dynamodb.AttributeValue).SetS(time.Now().Add(24 * time.Hour).Format(DateFormatDay)),
			"impressions": new(dynamodb.AttributeValue).SetN(fmt.Sprintf("%d", 2)),
			"clicks":      new(dynamodb.AttributeValue).SetN(fmt.Sprintf("%d", 1)),
		},
	}

	lastEvaluatedKey := map[string]*dynamodb.AttributeValue{
		"ad_id": new(dynamodb.AttributeValue).SetS(ad.String()),
		"day":   new(dynamodb.AttributeValue).SetS(end.Format(DateFormatDay)),
	}

	output := new(dynamodb.QueryOutput)
	output.SetCount(2)
	output.SetItems(items)
	output.SetLastEvaluatedKey(lastEvaluatedKey)

	days := []*AdDay{
		{
			Day:         time.Now().Format(DateFormatDay),
			Impressions: 4,
			Clicks:      3,
		},
		{
			Day:         time.Now().Add(24 * time.Hour).Format(DateFormatDay),
			Impressions: 2,
			Clicks:      1,
		},
	}

	ddb := new(amzmock.DynamoDB)
	ddb.On("QueryWithContext", ctx, input).Return(output, nil)

	table := NewAdTrafficTable(tableName, ddb)

	returnedDays, returnedNext, err := table.Days(ctx, ad, start, end, page, limit)

	ddb.AssertExpectations(t)
	assert.Equal(t, days, returnedDays)
	assert.Equal(t, lastEvaluatedKey, returnedNext)
	assert.NoError(t, err)

	t.Run("returns error", func(t *testing.T) {
		t.Run("on SDK error", func(t *testing.T) {
			expectedErr := fmt.Errorf("the-error")

			ddb := new(amzmock.DynamoDB)
			ddb.On("QueryWithContext", ctx, input).Return(nil, expectedErr)

			table := NewAdTrafficTable(tableName, ddb)
			returnedDays, returnedNext, err := table.Days(ctx, ad, start, end, page, limit)

			ddb.AssertExpectations(t)
			assert.Empty(t, returnedDays)
			assert.Nil(t, returnedNext)
			assert.Error(t, err)
			assert.Equal(t, expectedErr, err)
		})
	})
}

func TestMockAdTrafficTable(t *testing.T) {
	ctx := context.Background()
	ad := uuid.New()
	start := time.Now()
	end := time.Now()
	page := map[string]*dynamodb.AttributeValue{
		"ad_id": new(dynamodb.AttributeValue).SetS(ad.String()),
		"day":   new(dynamodb.AttributeValue).SetS(time.Now().Format(DateFormatDay)),
	}
	limit := int64(10)

	days := []*AdDay{
		{
			Day:         start.Format(DateFormatDay),
			Impressions: 2,
			Clicks:      1,
		},
	}
	next := map[string]*dynamodb.AttributeValue{
		"ad_id": page["ad_id"],
		"day":   new(dynamodb.AttributeValue).SetS(time.Now().Add(24 * time.Hour).Format(DateFormatDay)),
	}

	table := new(MockAdTrafficTable)
	table.On("Days", ctx, ad, start, end, page, limit).Return(days, next, nil)

	returnedDays, returnedNext, err := table.Days(ctx, ad, start, end, page, limit)

	table.AssertExpectations(t)
	assert.Equal(t, days, returnedDays)
	assert.Equal(t, next, returnedNext)
	assert.NoError(t, err)

	t.Run("with no next", func(t *testing.T) {
		table := new(MockAdTrafficTable)
		table.On("Days", ctx, ad, start, end, page, limit).Return(days, nil, nil)

		returnedDays, returnedNext, err := table.Days(ctx, ad, start, end, page, limit)

		table.AssertExpectations(t)
		assert.Equal(t, days, returnedDays)
		assert.Nil(t, returnedNext)
		assert.NoError(t, err)
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := fmt.Errorf("the-error")

		table := new(MockAdTrafficTable)
		table.On("Days", ctx, ad, start, end, page, limit).Return(nil, nil, expectedErr)

		returnedDays, returnedNext, err := table.Days(ctx, ad, start, end, page, limit)

		table.AssertExpectations(t)
		assert.Empty(t, returnedDays)
		assert.Nil(t, returnedNext)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}
