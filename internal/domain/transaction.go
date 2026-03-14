// Package domain contains the core business entities and rules.
// It has NO dependencies on infrastructure, frameworks, or I/O.
package domain

import (
	"fmt"
	"time"
)

// DateLayout is the canonical ISO date format used across the application.
const DateLayout = "2006-01-02"

// TransactionType classifies a movement as credit or debit.
type TransactionType string

const (
	Credit TransactionType = "credit"
	Debit  TransactionType = "debit"
)

// Transaction is the core business entity representing a single account movement.
type Transaction struct {
	ID     int
	UserID int
	Date   time.Time
	Amount float64
	Type   TransactionType
}

// NewTransaction creates a validated Transaction.
// The sign of amount determines the type: positive = credit, negative = debit.
func NewTransaction(id, userID int, date time.Time, amount float64) (Transaction, error) {
	if date.IsZero() {
		return Transaction{}, fmt.Errorf("transaction date cannot be zero")
	}
	if amount == 0 {
		return Transaction{}, fmt.Errorf("transaction amount cannot be zero")
	}

	txnType := Credit
	if amount < 0 {
		txnType = Debit
	}

	return Transaction{
		ID:     id,
		UserID: userID,
		Date:   date,
		Amount: amount,
		Type:   txnType,
	}, nil
}
