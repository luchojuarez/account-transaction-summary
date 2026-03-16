package csv_test

import (
	"os"
	"testing"
	"time"

	csvadapter "github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/csv"
	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	// Create a 'folder' subdirectory to match the test cases
	subDir := dir + "/folder"
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(subDir, "txns-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestCSVFileReader_ValidFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		wantLen   int
		wantFirst domain.Transaction
	}{
		{
			name: "challenge example CSV (single user inferred, no userID column)",
			content: "id,date,transaction\n" +
				"1,2021-07-15,+60.5\n" +
				"2,2021-07-28,-20.46\n" +
				"3,2021-08-02,+10\n" +
				"4,2021-08-10,-10.30\n",
			wantLen: 4,
			wantFirst: domain.Transaction{
				ID:     1,
				UserID: 0,
				Date:   time.Date(2021, time.July, 15, 0, 0, 0, 0, time.UTC),
				Amount: 60.5,
				Type:   domain.Credit,
			},
		},
		{
			name: "whitespace around values is trimmed (no userID column)",
			content: "id,date,transaction\n" +
				" 3 , 2021-08-02 , +10 \n",
			wantLen: 1,
			wantFirst: domain.Transaction{
				ID:     3,
				UserID: 0,
				Date:   time.Date(2021, time.August, 2, 0, 0, 0, 0, time.UTC),
				Amount: 10,
				Type:   domain.Credit,
			},
		},
		{
			name: "negative transaction is debit (no userID column)",
			content: "id,date,transaction\n" +
				"2,2021-07-28,-20.46\n",
			wantLen: 1,
			wantFirst: domain.Transaction{
				ID:     2,
				UserID: 0,
				Date:   time.Date(2021, time.July, 28, 0, 0, 0, 0, time.UTC),
				Amount: -20.46,
				Type:   domain.Debit,
			},
		},
		{
			name: "file with explicit userID column is parsed correctly",
			content: "userID,id,date,transaction\n" +
				"10,1,2021-07-15,+60.5\n" +
				"20,2,2021-07-28,-20.46\n",
			wantLen: 2,
			wantFirst: domain.Transaction{
				ID:     1,
				UserID: 10,
				Date:   time.Date(2021, time.July, 15, 0, 0, 0, 0, time.UTC),
				Amount: 60.5,
				Type:   domain.Credit,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := writeTempCSV(t, tc.content)
			txns, err := csvadapter.NewFileReader("folder").ReadTransactions(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(txns) != tc.wantLen {
				t.Fatalf("count: want %d, got %d", tc.wantLen, len(txns))
			}

			got := txns[0]
			w := tc.wantFirst
			if got.ID != w.ID || got.UserID != w.UserID ||
				!got.Date.Equal(w.Date) || got.Amount != w.Amount || got.Type != w.Type {
				t.Errorf("first row:\n  want %+v\n  got  %+v", w, got)
			}
		})
	}
}

func TestCSVFileReader_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "completely empty file",
			content: "",
		},
		{
			name:    "header only — no data rows",
			content: "id,date,transaction\n",
		},
		{
			name:    "wrong date format (M/D instead of YYYY-MM-DD)",
			content: "id,date,transaction\n1,7/15,+60.5\n",
		},
		{
			name:    "invalid amount",
			content: "id,date,transaction\n1,2021-07-15,abc\n",
		},
		{
			name:    "missing column",
			content: "id,date,transaction\n1,2021-07-15\n",
		},
		{
			name:    "non-numeric id",
			content: "id,date,transaction\nabc,2021-07-15,+10\n",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := writeTempCSV(t, tc.content)
			_, err := csvadapter.NewFileReader("").ReadTransactions(path)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestCSVFileReader_FileNotFound(t *testing.T) {
	_, err := csvadapter.NewFileReader("").ReadTransactions("/nonexistent/path/txns.csv")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
