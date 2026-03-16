SHELL := /bin/bash

.PHONY: test localstack-up build-all warmup lambda-build lambda-deploy-localstack lambda-invoke-localstack lambda-rebuild-and-deploy sqs-build worker-build worker-up worker-build-and-restart

LAMBDA_FUNCTION_NAME ?= account-transaction-summary
LOCALSTACK_ENDPOINT ?= http://localhost:4566
LAMBDA_INVOKE_PAYLOAD ?= {"file_path":"data/txns.csv"}
LAMBDA_INVOKE_TYPE ?= RequestResponse
AWS_CLI_CONNECT_TIMEOUT ?= 60
AWS_CLI_READ_TIMEOUT ?= 300
SMTP_HOST ?= smtp.example.com
SMTP_PORT ?= 587
SMTP_USERNAME ?= test@example.com
SMTP_PASSWORD ?= secret

test:
	go test ./...


## Warm up entire system: LocalStack, SQS queue, Lambda deploy, worker build and run
warmup: localstack-up
	sleep 5 && $(MAKE) sqs-build
	$(MAKE) lambda-deploy-localstack
	$(MAKE) worker-build
	$(MAKE) worker-up

## Start LocalStack with Lambda support
localstack-up:
	docker compose up -d localstack

## Build the balance-news worker Docker image
worker-build:
	docker compose build worker

## Rebuild the worker image and restart the worker container (SMTP_* from Makefile or env)
worker-build-and-restart:
	SMTP_HOST="$(SMTP_HOST)" SMTP_PORT="$(SMTP_PORT)" SMTP_USERNAME="$(SMTP_USERNAME)" SMTP_PASSWORD="$(SMTP_PASSWORD)" \
	docker compose build worker && \
	SMTP_HOST="$(SMTP_HOST)" SMTP_PORT="$(SMTP_PORT)" SMTP_USERNAME="$(SMTP_USERNAME)" SMTP_PASSWORD="$(SMTP_PASSWORD)" \
	docker compose up -d --force-recreate worker

## Create SQS queue in LocalStack (requires localstack-up and SQS in SERVICES)
SQS_QUEUE_NAME ?= balanceNews
sqs-build:
	docker run --rm --network host \
		-e AWS_ACCESS_KEY_ID=test \
		-e AWS_SECRET_ACCESS_KEY=test \
		-e AWS_DEFAULT_REGION=us-east-1 \
		amazon/aws-cli sqs create-queue \
			--endpoint-url=$(LOCALSTACK_ENDPOINT) \
			--queue-name $(SQS_QUEUE_NAME)

## Build the AWS Lambda binary for Linux and package it as a zip (Docker)
lambda-build:
	docker run --rm -v "$(PWD):/app" -w /app -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 golang:1.25 sh -c "mkdir -p bin && go build -buildvcs=false -ldflags='-s -w' -o bin/bootstrap ./cmd/lambda"
	docker run --rm -v "$(PWD):/app" -w /app alpine sh -c "apk add --no-cache zip > /dev/null && rm -rf /app/bin/data && cp -R /app/data /app/bin/data && cd /app/bin && zip -q lambda.zip bootstrap && zip -qr lambda.zip data"


## Deploy the Lambda to LocalStack using the official AWS CLI Docker image (no local aws/awslocal needed)
lambda-deploy-localstack: lambda-build
	docker run --rm --network host \
		-v "$(PWD)/bin:/work/bin:ro" \
		-e AWS_ACCESS_KEY_ID=test \
		-e AWS_SECRET_ACCESS_KEY=test \
		-e AWS_DEFAULT_REGION=us-east-1 \
		amazon/aws-cli lambda create-function \
			--endpoint-url=$(LOCALSTACK_ENDPOINT) \
			--function-name $(LAMBDA_FUNCTION_NAME) \
			--runtime provided.al2 \
			--zip-file fileb:///work/bin/lambda.zip \
			--environment "{\"Variables\":{\"SMTP_HOST\":\"$(SMTP_HOST)\",\"SMTP_PORT\":\"$(SMTP_PORT)\",\"SMTP_USERNAME\":\"$(SMTP_USERNAME)\",\"SMTP_PASSWORD\":\"$(SMTP_PASSWORD)\"}}" \
			--handler bootstrap \
			--role arn:aws:iam::000000000000:role/lambda-role


## Invoke the Lambda on LocalStack using the official AWS CLI Docker image (no local aws/awslocal needed)
lambda-invoke-localstack:
	docker run --rm --network host \
		-v "$(PWD):/work" \
		-e AWS_ACCESS_KEY_ID=test \
		-e AWS_SECRET_ACCESS_KEY=test \
		-e AWS_DEFAULT_REGION=us-east-1 \
		-e AWS_MAX_ATTEMPTS=10 \
		amazon/aws-cli lambda invoke \
			--cli-connect-timeout $(AWS_CLI_CONNECT_TIMEOUT) \
			--cli-read-timeout $(AWS_CLI_READ_TIMEOUT) \
			--endpoint-url=$(LOCALSTACK_ENDPOINT) \
			--invocation-type $(LAMBDA_INVOKE_TYPE) \
			--function-name $(LAMBDA_FUNCTION_NAME) \
			--payload "$$(echo -n '$(LAMBDA_INVOKE_PAYLOAD)' | base64)" \
			/work/lambda-output.json

## Fully rebuild and redeploy the Lambda to LocalStack (clean artifacts, restart docker-compose, rebuild, deploy via Docker)
lambda-rebuild-and-deploy:
	docker compose down
	docker run --rm -v "$(PWD):/work" -w /work alpine rm -rf bin/bootstrap bin/lambda.zip bin/data
	docker compose up -d localstack
	$(MAKE) lambda-deploy-localstack


