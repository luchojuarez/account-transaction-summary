package csv_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	csvadapter "github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/csv"
)

// fakeS3Server returns a minimal HTTP server that responds to S3 GetObject
// requests with the given body.
func fakeS3Server(t *testing.T, bucket, key, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/%s/%s", bucket, key)
		if r.Method != http.MethodGet || r.URL.Path != expectedPath {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		fmt.Fprint(w, body)
	}))
}

const validCSV = `id,date,transaction
1,2021-07-15,+60.5
2,2021-07-28,-10.0
`

// Note: these tests use t.Setenv so they CANNOT use t.Parallel().

func TestS3Reader_ReadTransactions_HappyPath(t *testing.T) {
	srv := fakeS3Server(t, "test-bucket", "txns.csv", validCSV)
	defer srv.Close()

	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")

	reader, err := csvadapter.NewS3Reader(context.Background(), "test-bucket", "", srv.URL)
	if err != nil {
		t.Fatalf("NewS3Reader: %v", err)
	}

	txns, err := reader.ReadTransactions("txns.csv")
	if err != nil {
		t.Fatalf("ReadTransactions: %v", err)
	}

	if len(txns) != 2 {
		t.Errorf("want 2 transactions, got %d", len(txns))
	}
}

func TestS3Reader_ReadTransactions_S3Error(t *testing.T) {
	// Server always returns 404
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "NoSuchKey", http.StatusNotFound)
	}))
	defer srv.Close()

	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")

	reader, err := csvadapter.NewS3Reader(context.Background(), "test-bucket", "", srv.URL)
	if err != nil {
		t.Fatalf("NewS3Reader: %v", err)
	}

	_, err = reader.ReadTransactions("missing.csv")
	if err == nil {
		t.Fatal("expected error for missing object, got nil")
	}
}

func TestS3Reader_ReadTransactions_InvalidCSV(t *testing.T) {
	body := strings.Repeat("bad data\n", 3) // no header, no valid columns

	srv := fakeS3Server(t, "test-bucket", "bad.csv", body)
	defer srv.Close()

	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")

	reader, err := csvadapter.NewS3Reader(context.Background(), "test-bucket", "", srv.URL)
	if err != nil {
		t.Fatalf("NewS3Reader: %v", err)
	}

	_, err = reader.ReadTransactions("bad.csv")
	if err == nil {
		t.Fatal("expected parse error for invalid CSV, got nil")
	}
}
