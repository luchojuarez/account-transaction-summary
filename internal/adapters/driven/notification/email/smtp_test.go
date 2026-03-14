package email

import (
	"fmt"
	"mime"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	gomail "gopkg.in/gomail.v2"

	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

func TestNewSMTPNotifierFromEnv_Table(t *testing.T) {
	type envMap map[string]string

	tests := []struct {
		name    string
		env     envMap
		wantErr bool
	}{
		{
			name: "missing host",
			env: envMap{
				"SMTP_PORT":     "587",
				"SMTP_USERNAME": "user@example.com",
				"SMTP_PASSWORD": "secret",
			},
			wantErr: true,
		},
		{
			name: "missing port",
			env: envMap{
				"SMTP_HOST":     "smtp.example.com",
				"SMTP_USERNAME": "user@example.com",
				"SMTP_PASSWORD": "secret",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			env: envMap{
				"SMTP_HOST":     "smtp.example.com",
				"SMTP_PORT":     "not-a-number",
				"SMTP_USERNAME": "user@example.com",
				"SMTP_PASSWORD": "secret",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			env: envMap{
				"SMTP_HOST":     "smtp.example.com",
				"SMTP_PORT":     "587",
				"SMTP_PASSWORD": "secret",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			env: envMap{
				"SMTP_HOST":     "smtp.example.com",
				"SMTP_PORT":     "587",
				"SMTP_USERNAME": "user@example.com",
			},
			wantErr: true,
		},
		{
			name: "valid configuration",
			env: envMap{
				"SMTP_HOST":           "smtp.example.com",
				"SMTP_PORT":           "587",
				"SMTP_USERNAME":       "user@example.com",
				"SMTP_PASSWORD":       "secret",
				"UNRELATED_OTHER_ENV": "ignored",
			},
			wantErr: false,
		},
	}

	orig := snapshotEnv()
	defer restoreEnv(orig)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearSMTPEnv()
			for k, v := range tt.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Setenv(%s) failed: %v", k, err)
				}
			}

			n, err := NewSMTPNotifierFromEnv()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (notifier=%+v)", n)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSMTPNotifier_SendSummary_Table(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseSummary := domain.UserSummary{
		UserID:            42,
		TotalBalance:      100.50,
		TotalTransactions: 3,
		OverallAvgCredit:  75.0,
		OverallAvgDebit:   -25.0,
		Monthly:           nil,
	}

	tests := []struct {
		name      string
		toEmail   string
		toName    string
		setupMock func(m *MockmailDialer)
		wantErr   bool
	}{
		{
			name:    "empty recipient email yields error and does not dial",
			toEmail: "   ",
			toName:  "Alice",
			setupMock: func(m *MockmailDialer) {
				// No expectations: DialAndSend must NOT be called.
			},
			wantErr: true,
		},
		{
			name:    "dialer error is propagated",
			toEmail: "alice@example.com",
			toName:  "Alice",
			setupMock: func(m *MockmailDialer) {
				m.EXPECT().
					DialAndSend(gomock.Any()).
					DoAndReturn(assertMessage(t, "alice@example.com", "💳 Your account summary for Alice", assertError("forced dialer error"))).
					Times(1)
			},
			wantErr: true,
		},
		{
			name:    "successful send with name",
			toEmail: "bob@example.com",
			toName:  "Bob",
			setupMock: func(m *MockmailDialer) {
				m.EXPECT().
					DialAndSend(gomock.Any()).
					DoAndReturn(assertMessage(t, "bob@example.com", "💳 Your account summary for Bob", nil)).
					Times(1)
			},
			wantErr: false,
		},
		{
			name:    "successful send without name",
			toEmail: "charlie@example.com",
			toName:  "",
			setupMock: func(m *MockmailDialer) {
				m.EXPECT().
					DialAndSend(gomock.Any()).
					DoAndReturn(assertMessage(t, "charlie@example.com", "Account summary", nil)).
					Times(1)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDialer := NewMockmailDialer(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockDialer)
			}

			n := &SMTPNotifier{
				host:          "smtp.example.com",
				port:          587,
				username: "user@example.com",
				password: "secret",
				dialer:   mockDialer,
			}

			err := n.SendSummary(tt.toEmail, tt.toName, baseSummary)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// assertMessage returns a function suitable for use with gomock's DoAndReturn,
// inspecting the gomail.Message headers before returning a provided error.
func assertMessage(t *testing.T, wantTo, wantSubject string, returnErr error) func(m ...*gomail.Message) error {
	t.Helper()
	return func(m ...*gomail.Message) error {
		if len(m) != 1 {
			t.Fatalf("expected 1 message, got %d", len(m))
		}
		msg := m[0]

		if got := msg.GetHeader("To"); len(got) != 1 || got[0] != wantTo {
			t.Fatalf("To header = %v, want %q", got, wantTo)
		}
		if got := msg.GetHeader("Subject"); len(got) != 1 {
			t.Fatalf("Subject header missing or multiple: %v", got)
		} else {
			dec := new(mime.WordDecoder)
			decodedSubj, err := dec.DecodeHeader(got[0])
			if err != nil {
				decodedSubj = got[0]
			}
			if decodedSubj != wantSubject {
				t.Fatalf("Subject header = %q, want %q", decodedSubj, wantSubject)
			}
		}

		return returnErr
	}
}

func assertError(msg string) error {
	return fmt.Errorf("forced dialer error: %s", msg)
}

func snapshotEnv() map[string]string {
	keys := []string{
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_USERNAME",
		"SMTP_PASSWORD",
	}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = os.Getenv(k)
	}
	return out
}

func restoreEnv(m map[string]string) {
	for k, v := range m {
		if v == "" {
			_ = os.Unsetenv(k)
		} else {
			_ = os.Setenv(k, v)
		}
	}
}

func clearSMTPEnv() {
	_ = os.Unsetenv("SMTP_HOST")
	_ = os.Unsetenv("SMTP_PORT")
	_ = os.Unsetenv("SMTP_USERNAME")
	_ = os.Unsetenv("SMTP_PASSWORD")
}

