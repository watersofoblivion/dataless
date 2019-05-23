package api

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type AdvertisingInfoRequest struct {
	ID    string
	Start string
	End   string
	Page  string
	Limit int
	expr  expression.Expression
}

func NewAdvertisingInfoRequest(req events.APIGatewayProxyRequest) (*AdvertisingInfoRequest, error) {
	request := &AdvertisingInfoRequest{
		ID:    req.PathParameters["id"],
		Start: req.QueryStringParameters["start"],
		End:   req.QueryStringParameters["end"],
		Page:  req.QueryStringParameters["page"],
	}

	if limit := req.QueryStringParameters["limit"]; limit != "" {
		l, err := strconv.ParseInt(limit, 10, 32)
		if err != nil {
			return nil, err
		}
		request.Limit = int(l)
	}

	return request, nil
}

func (request *AdvertisingInfoRequest) ExclusiveStartKey() (map[string]*dynamodb.AttributeValue, error) {
	if request.Page != "" {
		exclusiveStartKey := make(map[string]*dynamodb.AttributeValue)

		dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(request.Page))
		if err := json.NewDecoder(dec).Decode(&exclusiveStartKey); err != nil {
			return nil, err
		}

		return exclusiveStartKey, nil
	}

	return nil, nil
}

func (request *AdvertisingInfoRequest) Expression() (expression.Expression, error) {
	keyCondition := expression.Key("ad_id").Equal(expression.Value(request.ID)).
		And(expression.Key("date").Between(
			expression.Value(request.Start),
			expression.Value(request.End),
		))

	return expression.NewBuilder().
		WithKeyCondition(keyCondition).
		Build()
}

func (request *AdvertisingInfoRequest) Query(tableName string) (*dynamodb.QueryInput, error) {
	expr, err := request.Expression()
	if err != nil {
		return nil, err
	}

	exclusiveStartKey, err := request.ExclusiveStartKey()
	if err != nil {
		return nil, err
	}

	input := new(dynamodb.QueryInput)
	input.SetTableName(tableName)
	input.SetExclusiveStartKey(exclusiveStartKey)
	input.SetExpressionAttributeNames(expr.Names())
	input.SetExpressionAttributeValues(expr.Values())
	input.SetKeyConditionExpression(aws.StringValue(expr.KeyCondition()))

	return input, nil
}

type AdvertisingInfoResponse struct {
	Page  string            `json:"page"`
	Next  string            `json:"next"`
	Count int               `json:"count"`
	Days  []*AdvertisingDay `json:"ads"`
}

func NewAdvertisingInfoResponse(resp *dynamodb.QueryOutput) (*AdvertisingInfoResponse, error) {
	return nil, nil
}

type AdvertisingDay struct {
	Day              string  `json:"day"`
	Impressions      int64   `json:"impressions"`
	Clicks           int64   `json:"clicks"`
	ClickthroughRate float64 `json:"clickthrough_rate"`
}
