package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/google/uuid"
)

const DateFormat = "2006-01-02"

var (
	tableName = os.Getenv("TABLE_NAME")
	ddb       = dynamodb.New(session.New())
)

func init() {
	xray.AWS(ddb.Client)
}

func Handler(ctx context.Context, evt events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	req, err := RequestFromEvent(evt)
	if err != nil {
		return APIGWResponse(http.StatusBadRequest, err, nil)
	}

	input, err := req.Query(tableName)
	if err != nil {
		return APIGWResponse(http.StatusBadRequest, err, nil)
	}

	output, err := ddb.QueryWithContext(ctx, input)
	if err != nil {
		return APIGWResponse(http.StatusInternalServerError, err, nil)
	}

	response, err := ResponseFromOutput(output)
	if err != nil {
		return APIGWResponse(http.StatusInternalServerError, err, nil)
	}

	return APIGWResponse(http.StatusOK, response, nil)
}

type Request struct {
	Ad                uuid.UUID
	Start             time.Time
	End               time.Time
	ExclusiveStartKey map[string]*dynamodb.AttributeValue
	Limit             int64
}

func RequestFromEvent(evt events.APIGatewayProxyRequest) (request *Request, err error) {
	req := &Request{Start: time.Now(), End: time.Now()}

	if req.Ad, err = uuid.Parse(evt.PathParameters["id"]); err != nil {
		return
	}

	if startDate := evt.QueryStringParameters["start"]; startDate != "" {
		if req.Start, err = time.Parse(DateFormat, startDate); err != nil {
			return
		}
	}

	if endDate := evt.QueryStringParameters["end"]; endDate != "" {
		if req.End, err = time.Parse(DateFormat, endDate); err != nil {
			return
		}
	}

	if page := evt.QueryStringParameters["page"]; page != "" {
		req.ExclusiveStartKey = make(map[string]*dynamodb.AttributeValue)
		r := strings.NewReader(page)
		b64Decoder := base64.NewDecoder(base64.StdEncoding, r)
		if err = json.NewDecoder(b64Decoder).Decode(&req.ExclusiveStartKey); err != nil {
			return
		}
	}

	if lim := evt.QueryStringParameters["limit"]; lim != "" {
		if req.Limit, err = strconv.ParseInt(lim, 64, 0); err != nil {
			return
		}
	}

	req = request
	return
}

func (request *Request) Expression() (expression.Expression, error) {
	ad := expression.Key("ad_id").Equal(
		expression.Value(request.Ad),
	)
	date := expression.Key("date").Between(
		expression.Value(request.Start.Format(DateFormat)),
		expression.Value(request.End.Format(DateFormat)),
	)

	return expression.NewBuilder().
		WithKeyCondition(expression.KeyAnd(ad, date)).
		Build()
}

func (request *Request) Query(tableName string) (*dynamodb.QueryInput, error) {
	expr, err := request.Expression()
	if err != nil {
		return nil, err
	}

	query := new(dynamodb.QueryInput)
	query.SetTableName(tableName)
	query.SetExpressionAttributeNames(expr.Names())
	query.SetExpressionAttributeValues(expr.Values())
	query.SetKeyConditionExpression(aws.StringValue(expr.KeyCondition()))
	if request.ExclusiveStartKey != nil {
		query.SetExclusiveStartKey(request.ExclusiveStartKey)
	}
	if request.Limit > 0 {
		query.SetLimit(request.Limit)
	}

	return query, nil
}

type Response struct {
	Count int64  `json:"count"`
	Next  string `json:"next,omitempty"`
	Days  []*Day `json:"days"`
}

func ResponseFromOutput(output *dynamodb.QueryOutput) (*Response, error) {
	resp := &Response{
		Count: aws.Int64Value(output.Count),
		Days:  make([]*Day, aws.Int64Value(output.Count)),
	}

	var err error
	for i, item := range output.Items {
		resp.Days[i] = &Day{Day: aws.StringValue(item["day"].S)}

		if resp.Days[i].Impressions, err = strconv.ParseInt(aws.StringValue(item["impressions"].N), 64, 0); err != nil {
			return nil, err
		}
		if resp.Days[i].Clicks, err = strconv.ParseInt(aws.StringValue(item["clicks"].N), 64, 0); err != nil {
			return nil, err
		}
		if resp.Days[i].ClickthroughRate, err = strconv.ParseFloat(aws.StringValue(item["clickthrough_rate"].N), 64); err != nil {
			return nil, err
		}
	}

	if len(output.LastEvaluatedKey) > 0 {
		buf := new(bytes.Buffer)
		b64Encoder := base64.NewEncoder(base64.StdEncoding, buf)

		if err := json.NewEncoder(b64Encoder).Encode(output.LastEvaluatedKey); err != nil {
			return nil, err
		}
		b64Encoder.Close()

		resp.Next = buf.String()
	}

	return resp, nil
}

type Day struct {
	Day              string  `json:"day"`
	Impressions      int64   `json:"impressions"`
	Clicks           int64   `json:"clicks"`
	ClickthroughRate float64 `json:"clickthrough_rate"`
}
