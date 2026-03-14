// Package application contains the use-case logic.
// It depends only on domain and ports — never on concrete adapters.
package application

import (
	"fmt"
	"log"

	"github.com/luchojuarez/account-transaction-summary/internal/application/ports"
	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// ProcessAccountUseCase implements the primary use-case:
// read transactions → group by user → compute summaries → persist → notify each user.
type ProcessAccountUseCase struct {
	reader   ports.TransactionReader
	repo     ports.TransactionRepository
	notifier ports.NotificationSender
	email    string
	name     string
}

// NewProcessAccountUseCase constructs the use-case with its required driven ports.
func NewProcessAccountUseCase(
	reader ports.TransactionReader,
	repo ports.TransactionRepository,
	notifier ports.NotificationSender,
	email string,
	name string,
) *ProcessAccountUseCase {
	return &ProcessAccountUseCase{
		reader:   reader,
		repo:     repo,
		notifier: notifier,
		email:    email,
		name:     name,
	}
}

// Process reads all transactions, groups them by user, and sends each user their summary.
func (uc *ProcessAccountUseCase) Process() error {
	// 1. Read all transactions from the source
	txns, err := uc.reader.ReadTransactions()
	if err != nil {
		return fmt.Errorf("reading transactions: %w", err)
	}
	log.Printf("[UseCase] Loaded %d transactions across all users", len(txns))

	// 2. Persist raw transactions (non-fatal)
	if err := uc.repo.SaveTransactions(txns); err != nil {
		log.Printf("[UseCase] Warning — could not save transactions: %v", err)
	}

	// 3. Pure domain computation: group by user and compute each summary
	summaries := domain.NewUserSummaries(txns)
	log.Printf("[UseCase] Computed summaries for %d user(s)", len(summaries))

	// 4. For each user: persist summary + send notification
	var errs []error
	for _, summary := range summaries {
		if err := uc.repo.SaveUserSummary(summary); err != nil {
			log.Printf("[UseCase] Warning — could not save summary for user %d: %v", summary.UserID, err)
		}

		if err := uc.notifier.SendSummary(uc.email, uc.name, summary); err != nil {
			errs = append(errs, fmt.Errorf("notify user %d: %w", summary.UserID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}
	return nil
}
