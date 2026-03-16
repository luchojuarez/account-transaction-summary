// Package sqs provides SummaryNewsPublisher implementations.
// This file holds SQS configuration (queue name and env loader).
package sqs

import (
	"fmt"
	"os"
	"strings"
)

const defaultQueueName = "balanceNews"

// Config holds SQS publisher settings.
type Config struct {
	// QueueName is the SQS queue name (e.g. "balanceNews").
	// Used to resolve the queue URL when QueueURL is empty.
	QueueName string
	// QueueURL overrides QueueName when set (e.g. for LocalStack: http://localhost:4566/000000000000/balanceNews).
	QueueURL string
}

// FromEnv loads SQS config from the environment:
//   - SQS_QUEUE_NAME: queue name (default "balanceNews")
//   - SQS_QUEUE_URL: optional full queue URL; if set, QueueName is ignored for sending
func FromEnv() Config {
	c := Config{
		QueueName: defaultQueueName,
	}
	if n := strings.TrimSpace(os.Getenv("SQS_QUEUE_NAME")); n != "" {
		c.QueueName = n
	}
	if u := strings.TrimSpace(os.Getenv("SQS_QUEUE_URL")); u != "" {
		c.QueueURL = u
	}
	return c
}

// QueueURLOrName returns the queue URL if set, otherwise the queue name for resolution.
func (c Config) QueueURLOrName() string {
	if c.QueueURL != "" {
		return c.QueueURL
	}
	return c.QueueName
}

// Validate returns an error if config is invalid.
func (c Config) Validate() error {
	if c.QueueName == "" && c.QueueURL == "" {
		return fmt.Errorf("SQS config: either SQS_QUEUE_NAME or SQS_QUEUE_URL must be set")
	}
	return nil
}
