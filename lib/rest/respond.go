package rest

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

func Respond(status int, body interface{}, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers:    headers,
	}

	if body != nil {
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["Content-Type"] = "application/json"

		switch b := body.(type) {
		case error:
			body = map[string]string{"error": b.Error()}
		}

		bs, err := json.Marshal(body)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}

		resp.Body = string(bs)
	}

	return resp, nil
}
