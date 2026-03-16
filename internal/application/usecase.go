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
// read transactions → save transactions → compute summaries by user → publish to SQS.
type ProcessAccountUseCase struct {
	reader               ports.TransactionReader
	repo                 ports.TransactionRepository
	summaryNewsPublisher ports.SummaryNewsPublisher
}

// NewProcessAccountUseCase constructs the use-case with its required driven ports.
func NewProcessAccountUseCase(
	reader ports.TransactionReader,
	repo ports.TransactionRepository,
	summaryNewsPublisher ports.SummaryNewsPublisher,
) *ProcessAccountUseCase {
	return &ProcessAccountUseCase{
		reader:               reader,
		repo:                 repo,
		summaryNewsPublisher: summaryNewsPublisher,
	}
}

// Process reads transactions, saves them, computes summaries by user, and publishes to SQS.
func (uc *ProcessAccountUseCase) Process() error {
	// 1. Read transactions
	txns, err := uc.reader.ReadTransactions()
	if err != nil {
		return fmt.Errorf("reading transactions: %w", err)
	}
	log.Printf("[UseCase] Loaded %d transactions across all users", len(txns))

	// 2. Save transactions (non-fatal)
	if err := uc.repo.SaveTransactions(txns); err != nil {
		log.Printf("[UseCase] Warning — could not save transactions: %v", err)
	}

	// 3. Calculate summaries by user
	summaries := domain.NewUserSummaries(txns, 0.0)
	log.Printf("[UseCase] Computed summaries for %d user(s)", len(summaries))

	// 4. Publish each summary to SQS
	for _, summary := range summaries {
		if err := uc.summaryNewsPublisher.PublishSummary(summary); err != nil {
			return fmt.Errorf("publish summary for user %d to SQS: %w", summary.UserID, err)
		}
	}
	return nil
}
