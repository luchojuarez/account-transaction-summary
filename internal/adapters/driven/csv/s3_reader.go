// Package csv implements the TransactionReader port backed by a CSV file or S3 object.
package csv

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// S3Reader is a TransactionReader that downloads a CSV object from S3 and
// parses it using the same logic as FileReader.
type S3Reader struct {
	bucket string
	prefix string
	client *s3.Client
}

// NewS3Reader creates an S3Reader.
// prefix is an optional base path (e.g. "txns/") for subsequent reads.
// endpoint is optional: when non-empty it overrides the S3 endpoint URL,
// which is useful for LocalStack (e.g. "http://localhost:4566").
func NewS3Reader(ctx context.Context, bucket, prefix, endpoint string) (*S3Reader, error) {
	optFns := []func(*config.LoadOptions) error{}

	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("s3reader: load AWS config: %w", err)
	}

	clientOpts := []func(*s3.Options){}
	if endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // LocalStack requires path-style URLs
		})
	}

	client := s3.NewFromConfig(cfg, clientOpts...)
	return &S3Reader{bucket: bucket, prefix: prefix, client: client}, nil
}

// ReadTransactions satisfies ports.TransactionReader.
// It fetches the S3 object and delegates CSV parsing to parseRecords.
func (r *S3Reader) ReadTransactions(key string) ([]domain.Transaction, error) {
	resolvedKey := key
	if strings.HasPrefix(key, "s3://") {
		u, err := url.Parse(key)
		if err == nil && u.Scheme == "s3" {
			resolvedKey = strings.TrimPrefix(u.Path, "/")
		}
	} else if r.prefix != "" {
		// If it's a relative path, prepend the prefix.
		// Ensure single slash between prefix and key.
		p := strings.TrimSuffix(r.prefix, "/")
		k := strings.TrimPrefix(key, "/")
		resolvedKey = p + "/" + k
	}

	log.Printf("[S3Reader] Fetching s3://%s/%s", r.bucket, resolvedKey)

	out, err := r.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(resolvedKey),
	})
	if err != nil {
		return nil, fmt.Errorf("s3reader: get object s3://%s/%s: %w", r.bucket, resolvedKey, err)
	}
	defer out.Body.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(out.Body); err != nil {
		return nil, fmt.Errorf("s3reader: read body: %w", err)
	}

	cr := csv.NewReader(&buf)
	cr.TrimLeadingSpace = true

	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("s3reader: parse csv: %w", err)
	}

	return parseRecords(records)
}
