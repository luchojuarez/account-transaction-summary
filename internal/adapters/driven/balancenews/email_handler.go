// Package balancenews provides BalanceNewsHandler implementations.
// This file contains the real handler that looks up user email and sends a notification.
package balancenews

import (
	"fmt"
	"log"

	"github.com/luchojuarez/account-transaction-summary/internal/application/ports"
	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// EmailHandler is a BalanceNewsHandler that looks up the user's email from UserRepository
// and sends the summary via NotificationSender.
type EmailHandler struct {
	notifier ports.NotificationSender
	userRepo ports.UserRepository
}

// NewEmailHandler builds a BalanceNewsHandler that sends summaries by email.
// It uses userRepo to resolve userID to email/name, then notifier to send.
func NewEmailHandler(notifier ports.NotificationSender, userRepo ports.UserRepository) *EmailHandler {
	return &EmailHandler{
		notifier: notifier,
		userRepo: userRepo,
	}
}

// Handle implements ports.BalanceNewsHandler: resolve user email, then send summary.
func (h *EmailHandler) Handle(summary domain.UserSummary) error {
	email, name, err := h.userRepo.GetUser(summary.UserID)
	if err != nil {
		return fmt.Errorf("get user %d: %w", summary.UserID, err)
	}
	if err := h.notifier.SendSummary(email, name, summary); err != nil {
		return fmt.Errorf("send summary to %s: %w", email, err)
	}
	log.Printf("[balanceNews] sent summary to user %d (%s)", summary.UserID, email)
	return nil
}
