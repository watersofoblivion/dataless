package events

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/stretchr/testify/mock"
)

type AdTrafficTable interface {
}

type dynamoAdTrafficTable struct {
	tableName string
	ddb       dynamodbiface.DynamoDBAPI
}

func NewAdTrafficTable(tableName string, ddb dynamodbiface.DynamoDBAPI) AdTrafficTable {
	return &dynamoAdTrafficTable{
		tableName: tableName,
		ddb:       ddb,
	}
}

type MockAdTrafficTable struct {
	mock.Mock
}
