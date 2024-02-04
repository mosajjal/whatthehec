//go:build lambda

package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(HandleRequest)
}

func HandleRequest(ctx context.Context, event CloudwatchLogs) (string, error) {
	s, err := event.MarshalJSON()
	if err != nil {
		return "", err
	}
	hecRuntime.SendSingleEvent(string(s))
	return "OK", nil
}
