package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

var encoder = json.NewEncoder(os.Stdout)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, v interface{}) error {
	log.Printf("Events Output")
	if err := encoder.Encode(v); err != nil {
		log.Printf("Error encoding request: %s", err)
	}
	return nil
}
