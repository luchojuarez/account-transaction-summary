package cli

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	csvadapter "github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/csv"
	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/notification/email"
	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/repository"
	"github.com/luchojuarez/account-transaction-summary/internal/application"
	"github.com/luchojuarez/account-transaction-summary/internal/application/ports"
)

// NewAccountProcessor composes the concrete adapters and returns an
// AccountProcessor that can be used by any driving adapter (CLI, Lambda, etc).
//
// csvPath can be:
//   - a local file path  (e.g. "data/txns.csv")
//   - an S3 URI          (e.g. "s3://my-bucket/path/txns.csv")
func NewAccountProcessor(csvPath, userEmail, defaultName string) (ports.AccountProcessor, error) {
	log.Printf("[CLI Adapter] Initialising account processor with CSV: %s, Email: %s, DefaultName: %s", csvPath, userEmail, defaultName)

	if csvPath == "" {
		csvPath = "./data/txns.csv"
		log.Printf("[CLI Adapter] CSV_PATH not set, defaulting to %q", csvPath)
	}

	var reader ports.TransactionReader

	if strings.HasPrefix(csvPath, "s3://") {
		bucket, key, err := parseS3URI(csvPath)
		if err != nil {
			return nil, fmt.Errorf("invalid S3 URI %q: %w", csvPath, err)
		}
		endpoint := os.Getenv("AWS_ENDPOINT_URL") // empty in real AWS; set for LocalStack
		log.Printf("[CLI Adapter] Using S3Reader — bucket=%s key=%s endpoint=%q", bucket, key, endpoint)

		r, err := csvadapter.NewS3Reader(context.Background(), bucket, key, endpoint)
		if err != nil {
			return nil, fmt.Errorf("create S3 reader: %w", err)
		}
		reader = r
	} else {
		log.Printf("[CLI Adapter] Using FileReader — path=%q", csvPath)
		reader = csvadapter.NewFileReader(csvPath)
	}

	repo := repository.NewNoopRepository()

	if userEmail == "" {
		log.Println("[CLI Adapter] userEmail not set — notifications may fail")
	}
	if defaultName == "" {
		defaultName = "Valued Customer"
	}

	notifier, err := email.NewSMTPNotifierFromEnv()
	if err != nil {
		return nil, err
	}

	uc := application.NewProcessAccountUseCase(reader, repo, notifier, userEmail, defaultName)
	return uc, nil
}

// parseS3URI parses an "s3://bucket/key" URI into its components.
func parseS3URI(rawURI string) (bucket, key string, err error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "s3" {
		return "", "", fmt.Errorf("expected s3 scheme, got %q", u.Scheme)
	}
	bucket = u.Host
	key = strings.TrimPrefix(u.Path, "/")
	if bucket == "" {
		return "", "", fmt.Errorf("URI must have a bucket (got bucket=%q)", bucket)
	}
	return bucket, key, nil
}
