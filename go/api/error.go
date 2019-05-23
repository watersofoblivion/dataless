package api

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

func ErrorResponse(code int, err error) (events.APIGatewayProxyResponse, error) {
	bs, err := json.Marshal(map[string]interface{}{
		"code":    code,
		"message": err.Error(),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Body:       string(bs),
	}, nil
}
