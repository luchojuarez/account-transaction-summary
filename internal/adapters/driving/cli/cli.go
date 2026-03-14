package cli

import (
	"log"

	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/csv"
	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/notification/email"
	"github.com/luchojuarez/account-transaction-summary/internal/adapters/driven/repository"
	"github.com/luchojuarez/account-transaction-summary/internal/application"
	"github.com/luchojuarez/account-transaction-summary/internal/application/ports"
)

// NewAccountProcessor composes the concrete adapters and returns an
// AccountProcessor that can be used by any driving adapter (CLI, Lambda, etc).
func NewAccountProcessor(csvPath, userEmail, defaultName string) (ports.AccountProcessor, error) {
	log.Printf("[CLI Adapter] Initialising account processor with CSV: %s, Email: %s, DefaultName: %s", csvPath, userEmail, defaultName)

	if csvPath == "" {
		csvPath = "./data/txns.csv"
		log.Printf("[CLI Adapter] CSV_PATH not set, defaulting to %q", csvPath)
	}

	reader := csv.NewFileReader(csvPath)
	repo := repository.NewNoopRepository()

	if userEmail == "" {
		log.Println("[CLI Adapter] userEmail not set — notifications may fail")
	}
	if defaultName == "" {
		defaultName = "Valued Customer"
	}

	notifier, err := email.NewSMTPNotifierFromEnv()
	if err != nil {
		return nil, err
	}

	uc := application.NewProcessAccountUseCase(reader, repo, notifier, userEmail, defaultName)
	return uc, nil
}
