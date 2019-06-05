package svc

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

type Handler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// Example
// func (handler Handler) WithLogf(format string, v ...interface{}) Handler {
// 	return Handler(func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
// 		log.Printf(format, v...)
// 		return handler(ctx, req)
// 	})
// }
