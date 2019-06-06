package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/watersofoblivion/dataless/svc/advertising"
)

func main() {
	controller := advertising.EnvController()
	lambda.Start(controller.PublishToCloudWatch)
}
