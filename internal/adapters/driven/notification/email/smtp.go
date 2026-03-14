// Package email implements the NotificationSender driven port using SMTP.
// It reads SMTP credentials and defaults from environment variables so that
// operators can configure email delivery without changing application code.
//
// Expected env vars:
//
//   SMTP_HOST         e.g. "smtp.gmail.com"
//   SMTP_PORT         e.g. "587"
//   SMTP_USERNAME     e.g. "lucho.juarez79@gmail.com"
//   SMTP_PASSWORD     app-specific password or secret
//
// The adapter sends a per-user HTML summary.
package email

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	gomail "gopkg.in/gomail.v2"

	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// mailDialer abstracts the gomail dialer for testability.
//go:generate mockgen -destination=mock_maildialer_test.go -package=email . mailDialer
type mailDialer interface {
	DialAndSend(m ...*gomail.Message) error
}

// SMTPNotifier implements the NotificationSender port over SMTP.
type SMTPNotifier struct {
	host     string
	port     int
	username string
	password string
	dialer   mailDialer
}

// NewSMTPNotifierFromEnv constructs an SMTPNotifier from environment variables.
// It validates that required values are present and returns an error otherwise.
func NewSMTPNotifierFromEnv() (*SMTPNotifier, error) {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	portStr := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	username := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	password := os.Getenv("SMTP_PASSWORD")

	if host == "" {
		return nil, fmt.Errorf("SMTP_HOST is required")
	}
	if portStr == "" {
		return nil, fmt.Errorf("SMTP_PORT is required")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("invalid SMTP_PORT %q", portStr)
	}
	if username == "" {
		return nil, fmt.Errorf("SMTP_USERNAME is required")
	}
	if password == "" {
		return nil, fmt.Errorf("SMTP_PASSWORD is required")
	}

	n := &SMTPNotifier{
		host:     host,
		port:     port,
		username: username,
		password: password,
	}
	n.dialer = gomail.NewDialer(n.host, n.port, n.username, n.password)
	return n, nil
}

// SendSummary delivers the given UserSummary to a single recipient via SMTP.
func (n *SMTPNotifier) SendSummary(toEmail, toName string, summary domain.UserSummary) error {
	if strings.TrimSpace(toEmail) == "" {
		return fmt.Errorf("empty recipient email for user %d", summary.UserID)
	}

	subject := "Account summary"
	if toName != "" {
		subject = fmt.Sprintf("💳 Your account summary for %s", toName)
	}

	htmlBody, err := GenerateHTMLSummary(summary, toName)
	if err != nil {
		return fmt.Errorf("generate HTML summary: %w", err)
	}
	plainBody := GeneratePlainTextSummary(summary, toName)

	msg := gomail.NewMessage()
	msg.SetHeader("From", n.username)
	msg.SetHeader("To", toEmail)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", plainBody)
	msg.AddAlternative("text/html", htmlBody)

	if err := n.dialer.DialAndSend(msg); err != nil {
		return fmt.Errorf("send mail to %s: %w", toEmail, err)
	}
	return nil
}

