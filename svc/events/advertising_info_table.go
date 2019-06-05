package events

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/stretchr/testify/mock"
)

type AdvertisingInfoTable interface {
}

type dynamoAdvertisingInfoTable struct {
	tableName string
	ddb       dynamodbiface.DynamoDBAPI
}

func NewAdvertisingInfoTable(tableName string, ddb dynamodbiface.DynamoDBAPI) AdvertisingInfoTable {
	return &dynamoAdvertisingInfoTable{
		tableName: tableName,
		ddb:       ddb,
	}
}

type MockAdvertisingInfoTable struct {
	mock.Mock
}
