package apigw

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

func Response(status int, body interface{}, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{StatusCode: status, Headers: headers}

	if body != nil {
		if err, ok := body.(error); ok {
			body = map[string]interface{}{"error": err.Error()}
		}

		bs, err := json.Marshal(body)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}

		resp.Headers["Content-Type"] = "application/json"
		resp.Body = string(bs)
	}

	return resp, nil
}
