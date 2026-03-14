SHELL := /bin/bash

.PHONY: test localstack-up lambda-build lambda-deploy-localstack lambda-invoke-localstack lambda-rebuild-and-deploy

LAMBDA_FUNCTION_NAME ?= account-transaction-summary
LOCALSTACK_ENDPOINT ?= http://localhost:4566
LAMBDA_INVOKE_PAYLOAD ?= {"file_path":"data/txns.csv","email":"lucho.juarez79@gmail.com"}
LAMBDA_INVOKE_TYPE ?= RequestResponse
AWS_CLI_CONNECT_TIMEOUT ?= 60
AWS_CLI_READ_TIMEOUT ?= 300
SMTP_HOST ?= smtp.example.com
SMTP_PORT ?= 587
SMTP_USERNAME ?= test@example.com
SMTP_PASSWORD ?= secret

test:
	go test ./...


## Start LocalStack with Lambda support
localstack-up:
	docker compose up -d localstack

## Build the AWS Lambda binary for Linux and package it as a zip
lambda-build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/bootstrap ./cmd/lambda
	rm -rf bin/data
	cp -R data bin/data
	cd bin && zip -q lambda.zip bootstrap && zip -qr lambda.zip data


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

## Fully rebuild and redeploy the Lambda to LocalStack (clean artifacts, restart docker-compose, rebuild, deploy via Docker AWS CLI)
lambda-rebuild-and-deploy:
	docker compose down
	rm -rf bin/bootstrap bin/lambda.zip bin/data
	docker compose up -d localstack
	$(MAKE) lambda-deploy-localstack


