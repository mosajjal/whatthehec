//go:build lambda

package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(HandleRequest)
}

type LogEvent struct {
	Timestamp int64  `json:"timestamp"`
	ID        string `json:"id"`
	Message   string `json:"message"`
}

type CloudwatchLogsWithLogEvents struct {
	LogEvents []LogEvent `json:"logEvents"`
}

func HandleRequest(ctx context.Context, event CloudwatchLogs) (string, error) {
	s, err := event.MarshalJSON()
	if err != nil {
		return "", err
	}

	// most likely, s looks like this (but compressed and not pretty printed)
	//
	// { [-]
	//    logEvents: [ [-]
	//      { [-]
	//        id: ID
	//        message: eni--- - - - -
	//        timestamp: EPOCH_TIME
	//      }
	//    ]
	//    logGroup: LOGGROUP_NAME
	//    logStream: LOGSTREAM_NAME
	//    messageType: DATA_MESSAGE
	//    owner: ACCOUNT_ID
	//    subscriptionFilters: [ [+]
	//    ]
	// }
	//
	// optionally, the logevents array can be extracted and sent to the runtime as individual events

	if args.ExtractLogEvents {
		logs := CloudwatchLogsWithLogEvents{}
		// extract log events and send them to the runtime
		err := json.Unmarshal(s, &logs)
		if err != nil {
			// send as a single event
			log.Printf("Failed to unmarshal log events: %s. Sending as a single event\n", err)
			hecRuntime.SendSingleEvent(string(s))
			return "OK", nil
		}
		// otherwise, send each log event as a separate event
		for _, logItem := range logs.LogEvents {
			logStr, err := json.Marshal(logItem)
			if err != nil {
				// we don't expect this to happen. but if it does, we'll just skip this log event
				log.Printf("Failed to marshal log event: %s. Skipping this log event\n", err)
				continue
			}
			hecRuntime.SendSingleEvent(string(logStr))
		}
		return "OK", nil
	}
	hecRuntime.SendSingleEvent(string(s))
	return "OK", nil
}
