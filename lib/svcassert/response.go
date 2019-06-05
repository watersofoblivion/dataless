package svcassert

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Response(t *testing.T, resp events.APIGatewayProxyResponse, status int, body interface{}, headers map[string]string) {
	assert.Equal(t, status, resp.StatusCode)

	if body != nil {
		assert.NotEmpty(t, resp.Body)

		if headers == nil {
			headers = map[string]string{"Content-Type": "application/json"}
		}
		assert.Contains(t, headers, "Content-Type")
		assert.Equal(t, headers["Content-Type"], "application/json")

		if body != mock.Anything {
			var bs []byte
			var err error
			if e, ok := body.(error); ok {
				bs, err = json.Marshal(map[string]string{"error": e.Error()})
			} else {
				bs, err = json.Marshal(body)
			}
			assert.NoError(t, err)
			assert.JSONEq(t, string(bs), resp.Body)
		}
	} else {
		assert.Empty(t, resp.Body)
	}

	assert.Len(t, resp.Headers, len(headers))
	for k, v := range headers {
		assert.Contains(t, resp.Headers, k)
		assert.Equal(t, v, resp.Headers[k])
	}
}
