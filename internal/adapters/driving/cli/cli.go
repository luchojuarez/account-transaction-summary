package cli

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/csv"
	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/repository"
	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/sqs"
	"github.com/luchojuarez/account-transaction-summary/internal/application"
	"github.com/luchojuarez/account-transaction-summary/internal/application/ports"
)

// NewAccountProcessor composes the concrete adapters and returns an
// AccountProcessor that can be used by any driving adapter (CLI, Lambda, etc).
// summaryPublisher can be nil to create a real SQS publisher from env (AWS config + SQS_QUEUE_NAME/SQS_QUEUE_URL).
func NewAccountProcessor(csvPath string, summaryPublisher ports.SummaryNewsPublisher) (ports.AccountProcessor, error) {
	log.Printf("[CLI Adapter] Initialising account processor with CSV: %s", csvPath)

	if csvPath == "" {
		csvPath = "./data/txns.csv"
		log.Printf("[CLI Adapter] CSV_PATH not set, defaulting to %q", csvPath)
	}

	reader := csv.NewFileReader(csvPath)
	repo := repository.NewLoggingTransactionRepository()
	if summaryPublisher == nil {
		ctx := context.Background()
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
			return nil, err
		}
		summaryPublisher, err = sqs.NewPublisherFromConfig(awsCfg, sqs.FromEnv())
		if err != nil {
			return nil, err
		}
	}

	uc := application.NewProcessAccountUseCase(reader, repo, summaryPublisher)
	return uc, nil
}
