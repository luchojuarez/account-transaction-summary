// Package ports defines the interfaces (ports) of the hexagonal architecture.
//
// Driven ports are interfaces that the application core CALLS.
// Concrete implementations (adapters) live in adapters/driven/.
package ports

import "github.com/luchojuarez/account-transaction-summary/internal/domain"

// TransactionReader is a driven port for loading transactions from any source.
type TransactionReader interface {
	ReadTransactions(key string) ([]domain.Transaction, error)
}

// TransactionRepository is a driven port for persisting account data.
type TransactionRepository interface {
	SaveTransactions(txns []domain.Transaction) error
	SaveUserSummary(summary domain.UserSummary) error
}

// NotificationSender is a driven port for delivering a summary to one user.
type NotificationSender interface {
	SendSummary(toEmail, toName string, summary domain.UserSummary) error
}
