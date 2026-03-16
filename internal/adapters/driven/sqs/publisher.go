// Package sqs provides a SummaryNewsPublisher that sends user summaries to AWS SQS.
package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// Publisher sends user summaries to an SQS queue.
type Publisher struct {
	client *sqs.Client
	url    string
}

// NewPublisher builds an SQS SummaryNewsPublisher using the given queue URL.
func NewPublisher(cfg aws.Config, queueURL string) *Publisher {
	return &Publisher{
		client: sqs.NewFromConfig(cfg),
		url:    queueURL,
	}
}

// ResolveQueueURL returns the queue URL from Config. If Config.QueueURL is set it is returned;
// otherwise the URL is resolved from Config.QueueName via GetQueueUrl.
func ResolveQueueURL(ctx context.Context, client *sqs.Client, c Config) (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	if c.QueueURL != "" {
		return c.QueueURL, nil
	}
	out, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(c.QueueName),
	})
	if err != nil {
		return "", fmt.Errorf("get SQS queue URL for %q: %w", c.QueueName, err)
	}
	return aws.ToString(out.QueueUrl), nil
}

// NewPublisherFromConfig builds an SQS SummaryNewsPublisher from Config.
// If Config.QueueURL is set it is used; otherwise the queue URL is resolved from Config.QueueName via GetQueueUrl.
func NewPublisherFromConfig(awsCfg aws.Config, c Config) (*Publisher, error) {
	ctx := context.Background()
	client := sqs.NewFromConfig(awsCfg)
	queueURL, err := ResolveQueueURL(ctx, client, c)
	if err != nil {
		return nil, err
	}
	return &Publisher{client: client, url: queueURL}, nil
}

// PublishSummary implements ports.SummaryNewsPublisher by sending one summary as a message to SQS.
func (p *Publisher) PublishSummary(summary domain.UserSummary) error {
	body, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(p.url),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return err
	}
	log.Printf("[SQS] Published summary for user %d", summary.UserID)
	return nil
}
