SHELL := /bin/bash

.PHONY: test localstack-up lambda-build lambda-deploy-localstack lambda-update-code lambda-invoke-localstack lambda-rebuild-and-deploy lambda-logs s3-setup-localstack s3-list s3-cat


DOCKER_NETWORK = account-transaction-summary_default

LAMBDA_FUNCTION_NAME ?= account-transaction-summary
LOCALSTACK_HOST_ENDPOINT = http://localhost:4566
LOCALSTACK_DOCKER_ENDPOINT = http://localstack:4566
S3_BUCKET            ?= account-txns
S3_KEY               ?= txns.csv
LAMBDA_INVOKE_PAYLOAD ?= {"file_path":"s3://$(S3_BUCKET)/$(S3_KEY)","email":"lucho.juarez79@gmail.com", "name":"Luciano Juarez"	}
LAMBDA_INVOKE_TYPE ?= RequestResponse
AWS_CLI_CONNECT_TIMEOUT ?= 60
AWS_CLI_READ_TIMEOUT ?= 300

# AWS common environment variables for LocalStack
AWS_ENV = \
	-e AWS_ACCESS_KEY_ID=test \
	-e AWS_SECRET_ACCESS_KEY=test \
	-e AWS_DEFAULT_REGION=us-east-1

SMTP_HOST ?= smtp.example.com
SMTP_PORT ?= 587
SMTP_USERNAME ?= test@example.com
SMTP_PASSWORD ?= secret

test:
	go test ./...


## Start LocalStack with Lambda support
localstack-up:
	docker compose up -d localstack

## Stop LocalStack and clean persistent data
localstack-clean:
	docker compose down

## Wait for LocalStack to be ready
localstack-wait:
	@echo "Waiting for LocalStack to be ready..."
	@until curl -s $(LOCALSTACK_HOST_ENDPOINT)/_localstack/health | grep -q "\"lambda\": \"\(available\|running\)\""; do sleep 2; done
	@echo "LocalStack is ready."

## Build the AWS Lambda binary for Linux and package it as a zip
lambda-build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/bootstrap ./cmd/lambda
	rm -rf bin/data
	cp -R data bin/data
	cd bin && zip -q lambda.zip bootstrap && zip -qr lambda.zip data


## Deploy the Lambda to LocalStack using the official AWS CLI Docker image (no local aws/awslocal needed)
lambda-deploy-localstack: lambda-build
	@echo "Deploying Lambda function $(LAMBDA_FUNCTION_NAME)..."
	docker run --rm \
		--network $(DOCKER_NETWORK) \
		$(AWS_ENV) \
		amazon/aws-cli lambda delete-function \
			--endpoint-url=$(LOCALSTACK_DOCKER_ENDPOINT) \
			--function-name $(LAMBDA_FUNCTION_NAME) 2>/dev/null || true
	docker run --rm \
		--network $(DOCKER_NETWORK) \
		-v "$(PWD)/bin:/work/bin:ro" \
		$(AWS_ENV) \
		amazon/aws-cli lambda create-function \
			--endpoint-url=$(LOCALSTACK_DOCKER_ENDPOINT) \
			--function-name $(LAMBDA_FUNCTION_NAME) \
			--runtime provided.al2 \
			--zip-file fileb:///work/bin/lambda.zip \
			--environment "{\"Variables\":{\"SMTP_HOST\":\"$(SMTP_HOST)\",\"SMTP_PORT\":\"$(SMTP_PORT)\",\"SMTP_USERNAME\":\"$(SMTP_USERNAME)\",\"SMTP_PASSWORD\":\"$(SMTP_PASSWORD)\",\"AWS_ENDPOINT_URL\":\"$(LOCALSTACK_DOCKER_ENDPOINT)\",\"AWS_ACCESS_KEY_ID\":\"test\",\"AWS_SECRET_ACCESS_KEY\":\"test\",\"AWS_DEFAULT_REGION\":\"us-east-1\"}}" \
			--handler bootstrap \
			--role arn:aws:iam::000000000000:role/lambda-role

## Create the S3 bucket and upload txns.csv to LocalStack
s3-setup-localstack:
	docker run --rm \
		--network $(DOCKER_NETWORK) \
		$(AWS_ENV) \
		amazon/aws-cli s3 mb s3://$(S3_BUCKET) \
			--endpoint-url=$(LOCALSTACK_DOCKER_ENDPOINT) || true
	docker run --rm \
		--network $(DOCKER_NETWORK) \
		-v "$(PWD)/data:/data:ro" \
		$(AWS_ENV) \
		amazon/aws-cli s3 cp /data/$(S3_KEY) s3://$(S3_BUCKET)/$(S3_KEY) \
			--endpoint-url=$(LOCALSTACK_DOCKER_ENDPOINT)


## Update Lambda code without recreating the entire function
lambda-update-code: lambda-build
	docker run --rm \
		--network $(DOCKER_NETWORK) \
		-v "$(PWD)/bin:/work/bin:ro" \
		$(AWS_ENV) \
		amazon/aws-cli lambda update-function-code \
			--endpoint-url=$(LOCALSTACK_DOCKER_ENDPOINT) \
			--function-name $(LAMBDA_FUNCTION_NAME) \
			--zip-file fileb:///work/bin/lambda.zip

## Tail Lambda logs from CloudWatch
lambda-logs:
	docker run --rm \
		--network $(DOCKER_NETWORK) \
		$(AWS_ENV) \
		amazon/aws-cli logs tail \
			--endpoint-url=$(LOCALSTACK_DOCKER_ENDPOINT) \
			/aws/lambda/$(LAMBDA_FUNCTION_NAME) --follow

## Show the last invocation output
lambda-output:
	@cat lambda-output.json | jq . || cat lambda-output.json

## Invoke the Lambda on LocalStack using the official AWS CLI Docker image (no local aws/awslocal needed)
lambda-invoke-localstack:
	docker run --rm \
		--network $(DOCKER_NETWORK) \
		-v "$(PWD):/work" \
		$(AWS_ENV) \
		-e AWS_MAX_ATTEMPTS=10 \
		amazon/aws-cli lambda invoke \
			--cli-connect-timeout $(AWS_CLI_CONNECT_TIMEOUT) \
			--cli-read-timeout $(AWS_CLI_READ_TIMEOUT) \
			--endpoint-url=$(LOCALSTACK_DOCKER_ENDPOINT) \
			--invocation-type $(LAMBDA_INVOKE_TYPE) \
			--function-name $(LAMBDA_FUNCTION_NAME) \
			--payload "$$(echo -n '$(LAMBDA_INVOKE_PAYLOAD)' | base64)" \
			/work/lambda-output.json

## Fully rebuild and redeploy the Lambda to LocalStack (clean artifacts, restart docker-compose, rebuild, deploy via Docker AWS CLI)
lambda-rebuild-and-deploy:
	$(MAKE) localstack-clean
	rm -rf bin/bootstrap bin/lambda.zip bin/data
	$(MAKE) localstack-up
	$(MAKE) localstack-wait
	$(MAKE) lambda-deploy-localstack
	$(MAKE) s3-setup-localstack

## List objects in S3 bucket
s3-list:
	docker run --rm \
		--network $(DOCKER_NETWORK) \
		$(AWS_ENV) \
		amazon/aws-cli s3 ls s3://$(S3_BUCKET) \
			--endpoint-url=$(LOCALSTACK_DOCKER_ENDPOINT) --recursive --human-readable --summarize

## Show content of the CSV in S3
s3-cat:
	docker run --rm \
		--network $(DOCKER_NETWORK) \
		$(AWS_ENV) \
		amazon/aws-cli s3 cp s3://$(S3_BUCKET)/$(S3_KEY) - \
			--endpoint-url=$(LOCALSTACK_DOCKER_ENDPOINT)


