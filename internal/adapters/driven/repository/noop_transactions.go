// Package repository provides TransactionRepository adapters.
// This file contains a no-op implementation that simply logs calls without
// performing any persistence. It is useful for demos, local development, or
// when you only care about notifications.
package repository

import (
	"log"

	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// LoggingTransactionRepository is a TransactionRepository implementation that only logs
// and does not persist. Useful for demos, local development, or when persistence is not needed.
type LoggingTransactionRepository struct{}

// NewLoggingTransactionRepository creates a TransactionRepository that logs calls without persisting.
func NewLoggingTransactionRepository() *LoggingTransactionRepository {
	return &LoggingTransactionRepository{}
}

// SaveTransactions satisfies ports.TransactionRepository by logging the call.
func (r *LoggingTransactionRepository) SaveTransactions(txns []domain.Transaction) error {
	log.Printf("[LoggingTransactionRepository] SaveTransactions called with %d transaction(s)", len(txns))
	return nil
}

// SaveUserSummary satisfies ports.TransactionRepository by logging the call.
func (r *LoggingTransactionRepository) SaveUserSummary(summary domain.UserSummary) error {
	log.Printf("[LoggingTransactionRepository] SaveUserSummary called for user %d (totalBalance=%.2f, totalTxns=%d)",
		summary.UserID, summary.TotalBalance, summary.TotalTransactions)
	return nil
}

