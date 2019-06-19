package advertising

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// DateFormatDay is the format for dates in the table.
const DateFormatDay = "2006-01-02"

// An AdTrafficTable is a table containing summary information about traffic to
// ads.
type AdTrafficTable interface {
	// Days queries the table and fetches a page of daily summaries between
	// "start" and "end".  The "page" and "limit" arguments should control paging.
	//
	// The returned values should be the zero or more days in the page of results,
	// and the key to fetch the next page.  If there is a problem fetching the
	// summaries start the table, this method should return an error.
	Days(ctx context.Context, ad uuid.UUID, start, end time.Time, page map[string]*dynamodb.AttributeValue, limit int64) ([]*AdDay, map[string]*dynamodb.AttributeValue, error)
}

type dynamoAdTrafficTable struct {
	tableName string
	ddb       dynamodbiface.DynamoDBAPI
}

// NewAdTrafficTable constructs an AdTrafficTable backed by DynamoDB.
func NewAdTrafficTable(tableName string, ddb dynamodbiface.DynamoDBAPI) AdTrafficTable {
	return &dynamoAdTrafficTable{
		tableName: tableName,
		ddb:       ddb,
	}
}

// Days implements the AdTrafficTable interface by querying the DynamoDB table.
func (table *dynamoAdTrafficTable) Days(ctx context.Context, ad uuid.UUID, start, end time.Time, page map[string]*dynamodb.AttributeValue, limit int64) ([]*AdDay, map[string]*dynamodb.AttributeValue, error) {
	keyCondition := expression.Key("ad_id").Equal(expression.Value(ad.String())).
		And(expression.Key("day").Between(
			expression.Value(start.Format(DateFormatDay)),
			expression.Value(end.Format(DateFormatDay))))

	expr, err := expression.NewBuilder().
		WithKeyCondition(keyCondition).
		Build()
	if err != nil {
		return nil, nil, err
	}

	input := new(dynamodb.QueryInput)
	input.SetTableName(table.tableName)
	input.SetExpressionAttributeNames(expr.Names())
	input.SetExpressionAttributeValues(expr.Values())
	input.SetKeyConditionExpression(aws.StringValue(expr.KeyCondition()))
	if len(page) > 0 {
		input.SetExclusiveStartKey(page)
	}
	if limit > 0 {
		input.SetLimit(limit)
	}

	output, err := table.ddb.QueryWithContext(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	items := make([]*AdDay, aws.Int64Value(output.Count))
	for i, item := range output.Items {
		if err := dynamodbattribute.UnmarshalMap(item, &items[i]); err != nil {
			return nil, nil, err
		}
	}

	return items, output.LastEvaluatedKey, nil
}

// MockAdTrafficTable is a mock for testing.
type MockAdTrafficTable struct {
	mock.Mock
}

// Days implements the AdTrafficTable interface.
func (mock *MockAdTrafficTable) Days(ctx context.Context, ad uuid.UUID, start, end time.Time, page map[string]*dynamodb.AttributeValue, limit int64) (days []*AdDay, next map[string]*dynamodb.AttributeValue, err error) {
	args := mock.Called(ctx, ad, start, end, page, limit)

	if daysArg := args.Get(0); daysArg != nil {
		days = daysArg.([]*AdDay)
	}
	if nextArg := args.Get(1); nextArg != nil {
		next = nextArg.(map[string]*dynamodb.AttributeValue)
	}

	return days, next, args.Error(2)
}

type AdDay struct {
	Day         string `json:"day" dynamodbav:"day"`
	Impressions int64  `json:"impressions" dynamodbav:"impressions"`
	Clicks      int64  `json:"clicks" dynamodbav:"clicks"`
}
