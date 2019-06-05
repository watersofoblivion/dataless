package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/watersofoblivion/dataless/svc/events"
)

func main() {
	controller := events.EnvController()
	lambda.Start(controller.Capture)
}
