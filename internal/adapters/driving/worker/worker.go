// Package worker provides a driving adapter that polls the balanceNews SQS queue
// and processes each message with a BalanceNewsHandler.
package worker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/luchojuarez/account-transaction-summary/internal/application/ports"
	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

const (
	longPollWaitSeconds = 20
	maxMessages         = 10
)

// Worker pulls messages from an SQS queue and processes them with a BalanceNewsHandler.
type Worker struct {
	client    *sqs.Client
	queueURL  string
	handler   ports.BalanceNewsHandler
}

// New creates a Worker that receives from the given queue and passes each message to handler.
func New(client *sqs.Client, queueURL string, handler ports.BalanceNewsHandler) *Worker {
	return &Worker{
		client:   client,
		queueURL: queueURL,
		handler:  handler,
	}
}

// Run polls the queue until ctx is cancelled. Each received message body is decoded
// as domain.UserSummary, passed to the handler, and deleted on success.
func (w *Worker) Run(ctx context.Context) error {
	log.Printf("[worker] started polling queue %q", w.queueURL)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[worker] context cancelled, stopping")
			return ctx.Err()
		default:
			if err := w.receiveAndProcess(ctx); err != nil && ctx.Err() == nil {
				log.Printf("[worker] receive/process error: %v", err)
			}
		}
	}
}

func (w *Worker) receiveAndProcess(ctx context.Context) error {
	out, err := w.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(w.queueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     longPollWaitSeconds,
	})
	if err != nil {
		return err
	}
	for _, msg := range out.Messages {
		if msg.Body == nil {
			continue
		}
		var summary domain.UserSummary
		if err := json.Unmarshal([]byte(*msg.Body), &summary); err != nil {
			log.Printf("[worker] invalid message body (skip, not deleted): %v", err)
			continue
		}
		if err := w.handler.Handle(summary); err != nil {
			log.Printf("[worker] handle error for user %d (message not deleted): %v", summary.UserID, err)
			continue
		}
		_, _ = w.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(w.queueURL),
			ReceiptHandle: msg.ReceiptHandle,
		})
	}
	return nil
}
