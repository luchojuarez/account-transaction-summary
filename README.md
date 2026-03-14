## Account Transaction Summary

`account-transaction-summary` is a small Go application packaged as an AWS Lambda function. It reads account transactions from a CSV file, computes per-user summaries, optionally persists them, and sends an email notification with each user’s summary. The project is wired in a hexagonal / ports-and-adapters style and is designed to run locally via LocalStack.

### Features

- **CSV transaction processing**: Reads transactions from a CSV file (see `data/txns.csv`).
- **User summaries**: Groups transactions per user and computes summary information in the domain layer.
- **Notification sending**: Uses an email-driven adapter to send summaries.
- **AWS Lambda ready**: Exposed as an AWS Lambda handler and built as a custom runtime binary (`bootstrap`).
- **Local development with LocalStack**: Run and invoke the Lambda locally using Docker and LocalStack.

### Quick setup

From the project root, with Docker running:

```bash
# 1) Start LocalStack
make localstack-up

# 2) Build and deploy the Lambda
make lambda-deploy-localstack

# 3) Invoke the Lambda with the default payload
make lambda-invoke-localstack
```

**Required environment variables (overridable when calling `make`):**

- `LAMBDA_FUNCTION_NAME` (default: `account-transaction-summary`)
- `LOCALSTACK_ENDPOINT` (default: `http://localhost:4566`)
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD` (used by the email adapter)

### Project Layout (high level)

- `cmd/lambda`: Lambda entrypoint (`main.go`) which simply calls the lambda driving adapter.
- `internal/adapters/driving/lambda`: AWS Lambda adapter (`Handler`, `Start`).
- `internal/adapters/driving/cli`: CLI adapter used by the Lambda to build and run the account processor.
- `internal/application`: Core use case (`ProcessAccountUseCase`) and ports.
- `internal/domain`: Domain entities (`Transaction`, `UserSummary`, etc.) and pure business logic.
- `internal/adapters/driven`: CSV reader, repository implementation(s), email sender, and user resolver.
- `data/`: Example CSV transaction data used for local runs and tests.

### Requirements

- **Go**: version `1.25` or later (see `go.mod`).
- **Docker** and **docker compose**: for running LocalStack and the AWS CLI image (via the Makefile targets).

### Getting Started

#### 1. Run tests

From the project root:

```bash
make test
```

This runs `go test ./...` across all packages.

#### 2. Start LocalStack

Start the LocalStack container (Lambda service only) using the Makefile:

```bash
make localstack-up
```

LocalStack will expose its edge endpoint on `http://localhost:4566`.

#### 3. Build the Lambda package

The Lambda is built as a Linux binary named `bootstrap` and zipped along with the example `data/` directory into `bin/lambda.zip`:

```bash
make lambda-build
```

This will:

- Compile the Lambda binary for `linux/amd64` into `bin/bootstrap`.
- Copy `data/` into `bin/data`.
- Create `bin/lambda.zip` containing `bootstrap` and `data/`.

#### 4. Deploy the Lambda to LocalStack

Use the Makefile target to deploy the function to LocalStack using the official `amazon/aws-cli` Docker image:

```bash
make lambda-deploy-localstack
```

For a full rebuild and deploy, including cleaning build artifacts and restarting LocalStack:

```bash
make lambda-rebuild-and-deploy
```

#### 5. Invoke the Lambda on LocalStack

Once deployed, invoke the Lambda using the Makefile target, which again uses the official AWS CLI Docker image:

```bash
make lambda-invoke-localstack
```

The default invocation payload (defined in `Makefile` as `LAMBDA_INVOKE_PAYLOAD`) is:

```json
{"file_path":"data/txns.csv","email":"lucho.juarez79@gmail.com","name":""}
```

You can override `LAMBDA_INVOKE_PAYLOAD` when calling `make` to use a different CSV path or email, e.g.:

```bash
make lambda-invoke-localstack LAMBDA_INVOKE_PAYLOAD='{"file_path":"data/txns.csv","email":"user@example.com","name":"Jane"}'
```

The Lambda response body will be written to `lambda-output.json` in the project root.

### Lambda Event Shape

The Lambda handler expects a JSON event with the following shape:

```json
{
  "file_path": "data/txns.csv",
  "email": "user@example.com",
  "name": "User Name"
}
```

- **file_path**: Path to the CSV file containing transactions, relative to the Lambda’s working directory.
- **email**: Destination email address for the summary.
- **name**: Display name for the user in the email content.

### SMTP Configuration

When deploying to LocalStack via `make lambda-deploy-localstack`, the following environment variables are injected into the Lambda:

- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USERNAME`
- `SMTP_PASSWORD`

Set these variables when invoking `make` to point to your SMTP server (or a local mail catcher).

### Development Notes

- The code is structured around **ports and adapters** to keep the domain and application logic independent of concrete infrastructure details.
- The **Lambda** and **CLI** driving adapters share the same composition logic, so behaviour remains consistent across environments.

### License

This project is provided as-is. Add your preferred license text here if you intend to distribute it.

