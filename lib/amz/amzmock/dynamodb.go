package amzmock

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/stretchr/testify/mock"
)

type DynamoDB struct {
	mock.Mock
	dynamodbiface.DynamoDBAPI
}

func (mock *DynamoDB) GetItemWithContext(ctx aws.Context, input *dynamodb.GetItemInput, options ...request.Option) (*dynamodb.GetItemOutput, error) {
	args := mock.Called(ctx, input)
	if output := args.Get(0); output != nil {
		return output.(*dynamodb.GetItemOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

func (mock *DynamoDB) PutItemWithContext(ctx aws.Context, input *dynamodb.PutItemInput, options ...request.Option) (*dynamodb.PutItemOutput, error) {
	args := mock.Called(ctx, input)
	if output := args.Get(0); output != nil {
		return output.(*dynamodb.PutItemOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

func (mock *DynamoDB) DeleteItemWithContext(ctx aws.Context, input *dynamodb.DeleteItemInput, options ...request.Option) (*dynamodb.DeleteItemOutput, error) {
	args := mock.Called(ctx, input)
	if output := args.Get(0); output != nil {
		return output.(*dynamodb.DeleteItemOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

func (mock *DynamoDB) QueryWithContext(ctx aws.Context, input *dynamodb.QueryInput, options ...request.Option) (*dynamodb.QueryOutput, error) {
	args := mock.Called(ctx, input)
	if output := args.Get(0); output != nil {
		return output.(*dynamodb.QueryOutput), args.Error(1)
	}
	return nil, args.Error(1)
}
