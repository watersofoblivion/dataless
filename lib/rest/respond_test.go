package rest

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/watersofoblivion/dataless/lib/rest/restassert"
)

func TestRespond(t *testing.T) {
	t.Run("Status Code", func(t *testing.T) {
		resp, err := Respond(http.StatusOK, nil, nil)

		assert.NoError(t, err)
		restassert.Response(t, resp, http.StatusOK, nil, nil)
	})

	t.Run("Headers", func(t *testing.T) {
		headers := map[string]string{
			"Foo": "Bar",
		}
		resp, err := Respond(http.StatusOK, nil, headers)

		assert.NoError(t, err)
		restassert.Response(t, resp, http.StatusOK, nil, headers)
	})

	t.Run("Body", func(t *testing.T) {
		body := "the body"
		resp, err := Respond(http.StatusOK, body, nil)

		assert.NoError(t, err)
		restassert.Response(t, resp, http.StatusOK, body, nil)

		t.Run("is an error", func(t *testing.T) {
			body := fmt.Errorf("the error")
			resp, err := Respond(http.StatusOK, body, nil)

			assert.NoError(t, err)
			restassert.Response(t, resp, http.StatusOK, body, nil)
		})

		t.Run("returns error", func(t *testing.T) {
			t.Run("on unmarshalable", func(t *testing.T) {
				_, err := Respond(http.StatusOK, make(chan int), nil)
				assert.Error(t, err)
			})
		})
	})
}
