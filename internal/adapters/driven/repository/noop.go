// Package repository provides TransactionRepository adapters.
// This file contains a no-op implementation that simply logs calls without
// performing any persistence. It is useful for demos, local development, or
// when you only care about notifications.
package repository

import (
	"log"

	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// NoopRepository is a TransactionRepository implementation that only logs.
type NoopRepository struct{}

// NewNoopRepository creates a new no-op repository.
func NewNoopRepository() *NoopRepository {
	return &NoopRepository{}
}

// SaveTransactions satisfies ports.TransactionRepository by logging the call.
func (r *NoopRepository) SaveTransactions(txns []domain.Transaction) error {
	log.Printf("[NoopRepository] SaveTransactions called with %d transaction(s)", len(txns))
	return nil
}

// SaveUserSummary satisfies ports.TransactionRepository by logging the call.
func (r *NoopRepository) SaveUserSummary(summary domain.UserSummary) error {
	log.Printf("[NoopRepository] SaveUserSummary called for user %d (totalBalance=%.2f, totalTxns=%d)",
		summary.UserID, summary.TotalBalance, summary.TotalTransactions)
	return nil
}

