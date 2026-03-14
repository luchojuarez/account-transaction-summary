// Package csv implements the TransactionReader port backed by a CSV file.
package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// Expected CSV header columns (case-insensitive, spaces stripped).
// Supported formats:
//
//	1) id,date,transaction
//	2) userID,id,date,transaction
const (
	colUserID      = "userid"
	colID          = "id"
	colDate        = "date"
	colTransaction = "transaction"
)

// FileReader is the CSV-file adapter for the TransactionReader driven port.
type FileReader struct {
	path string
}

// NewFileReader creates a FileReader that reads from the given file path.
func NewFileReader(path string) *FileReader {
	return &FileReader{path: path}
}

// ReadTransactions satisfies ports.TransactionReader.
// It supports both legacy and new CSV formats:
//
//	1) id,date,transaction
//	2) userID,id,date,transaction
func (r *FileReader) ReadTransactions() ([]domain.Transaction, error) {
	f, err := os.Open(r.path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", r.path, err)
	}
	defer f.Close()

	cr := csv.NewReader(f)
	cr.TrimLeadingSpace = true

	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv %q: %w", r.path, err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv %q: no data rows found", r.path)
	}

	// Parse header to find column indices — tolerant of ordering and spacing.
	colIdx, err := parseHeader(records[0])
	if err != nil {
		return nil, fmt.Errorf("csv %q header: %w", r.path, err)
	}

	txns := make([]domain.Transaction, 0, len(records)-1)
	for lineNum, row := range records[1:] {
		t, err := parseRow(lineNum+2, row, colIdx) // +2: 1-indexed + header row
		if err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, nil
}

// colIndices maps the expected column names to their positions.
type colIndices struct {
	userID      int
	id          int
	date        int
	transaction int
}

func parseHeader(header []string) (colIndices, error) {
	idx := colIndices{userID: -1, id: -1, date: -1, transaction: -1}
	for i, h := range header {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case colUserID:
			idx.userID = i
		case colID:
			idx.id = i
		case colDate:
			idx.date = i
		case colTransaction:
			idx.transaction = i
		}
	}
	missing := []string{}
	if idx.id < 0 {
		missing = append(missing, colID)
	}
	if idx.date < 0 {
		missing = append(missing, colDate)
	}
	if idx.transaction < 0 {
		missing = append(missing, colTransaction)
	}
	if len(missing) > 0 {
		return idx, fmt.Errorf("missing columns: %s", strings.Join(missing, ", "))
	}
	return idx, nil
}

func parseRow(lineNum int, row []string, idx colIndices) (domain.Transaction, error) {
	maxIdx := max3(idx.id, idx.date, idx.transaction)
	if idx.userID >= 0 && idx.userID > maxIdx {
		maxIdx = idx.userID
	}
	if len(row) <= maxIdx {
		return domain.Transaction{}, fmt.Errorf("line %d: expected at least %d columns, got %d",
			lineNum, maxIdx+1, len(row))
	}

	// When the CSV omits a userID column, we treat all transactions as
	// belonging to a single (default) user with ID = 0 to preserve the
	// original challenge behaviour.
	userID := 0
	if idx.userID >= 0 {
		parsed, err := strconv.Atoi(strings.TrimSpace(row[idx.userID]))
		if err != nil {
			return domain.Transaction{}, fmt.Errorf("line %d: invalid userID %q: %w", lineNum, row[idx.userID], err)
		}
		userID = parsed
	}

	id, err := strconv.Atoi(strings.TrimSpace(row[idx.id]))
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("line %d: invalid id %q: %w", lineNum, row[idx.id], err)
	}

	// Use time.Parse with the domain's canonical layout (2006-01-02).
	date, err := time.Parse(domain.DateLayout, strings.TrimSpace(row[idx.date]))
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("line %d: invalid date %q (expected %s): %w",
			lineNum, row[idx.date], domain.DateLayout, err)
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(row[idx.transaction]), 64)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("line %d: invalid transaction amount %q: %w",
			lineNum, row[idx.transaction], err)
	}

	return domain.NewTransaction(id, userID, date, amount)
}

func max3(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}
