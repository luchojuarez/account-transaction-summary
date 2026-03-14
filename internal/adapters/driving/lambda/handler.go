package lambda

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driving/cli"
)

// Request represents the expected JSON payload from the Lambda invocation.
type Request struct {
	FilePath string `json:"file_path"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

// Handler is the AWS Lambda entrypoint. It delegates to the same composition
// logic used by the CLI driving adapter so that configuration and behaviour
// stay consistent across environments.
//
// The handler reads the incoming event payload and runs the
// account-processing pipeline once per invocation.
func Handler(_ context.Context, req Request) (string, error) {
	log.Printf("[Lambda] Invoked account-transaction-summary handler with file_path: %q, email: %q, name: %q", req.FilePath, req.Email, req.Name)

	processor, err := cli.NewAccountProcessor(req.FilePath, req.Email, req.Name)
	if err != nil {
		log.Printf("[Lambda] Failed to initialise processor: %v", err)
		return "", err
	}

	if err := processor.Process(); err != nil {
		log.Printf("[Lambda] Processing failed: %v", err)
		return "", err
	}

	log.Println("[Lambda] Processing completed successfully")
	return "ok", nil
}

// Start is a convenience function to be used from a Lambda-specific main
// package. It wires the AWS Lambda runtime to the Handler above.
func Start() {
	lambda.Start(Handler)
}

