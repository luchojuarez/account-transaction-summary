// Package ports defines the interfaces (ports) of the hexagonal architecture.
//
// Driven ports are interfaces that the application core CALLS.
// Concrete implementations (adapters) live in adapters/driven/.
package ports

import "github.com/luchojuarez/account-transaction-summary/internal/domain"

// TransactionReader is a driven port for loading transactions from any source.
type TransactionReader interface {
	ReadTransactions() ([]domain.Transaction, error)
}

// TransactionRepository is a driven port for persisting account data.
type TransactionRepository interface {
	SaveTransactions(txns []domain.Transaction) error
	SaveUserSummary(summary domain.UserSummary) error
}

// UserRepository is a driven port for looking up user contact info by user ID.
type UserRepository interface {
	GetUser(userID int) (email, name string, err error)
}

// NotificationSender is a driven port for delivering a summary to one user.
type NotificationSender interface {
	SendSummary(toEmail, toName string, summary domain.UserSummary) error
}

// SummaryNewsPublisher is a driven port for publishing a single user summary to SQS.
type SummaryNewsPublisher interface {
	PublishSummary(summary domain.UserSummary) error
}

// BalanceNewsHandler is a driven port for handling one balance-news message (e.g. after consuming from SQS).
type BalanceNewsHandler interface {
	Handle(summary domain.UserSummary) error
}

