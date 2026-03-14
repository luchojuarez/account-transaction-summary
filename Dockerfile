FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o account-transaction-summary .

FROM gcr.io/distroless/base-debian12 AS final

WORKDIR /app

COPY --from=builder /app/account-transaction-summary /app/account-transaction-summary
COPY data ./data

ENV CSV_PATH=/app/data/txns.csv

ENTRYPOINT ["/app/account-transaction-summary"]

