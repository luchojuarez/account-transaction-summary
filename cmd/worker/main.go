package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/balancenews"
	sqsdriven "github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/sqs"
	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driving/worker"
	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/notification/email"
	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/repository"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var configOpts []func(*config.LoadOptions) error
	if u := os.Getenv("AWS_ENDPOINT_URL"); u != "" {
		configOpts = append(configOpts, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: u}, nil
			}),
		))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		log.Fatalf("[worker] load AWS config: %v", err)
	}
	client := sqs.NewFromConfig(awsCfg)
	sqsCfg := sqsdriven.FromEnv()
	queueURL, err := sqsdriven.ResolveQueueURL(ctx, client, sqsCfg)
	if err != nil {
		log.Fatalf("[worker] resolve queue URL: %v", err)
	}
	notifier, err := email.NewSMTPNotifierFromEnv()
	if err != nil {
		log.Fatalf("[worker] create email notifier: %v", err)
	}

	handler := balancenews.NewEmailHandler(notifier,repository.NewInMemoryUserRepository())


	w := worker.New(client, queueURL, handler)
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("[worker] run: %v", err)
	}
}
