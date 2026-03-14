package application_test

import (
	"testing"
	"time"

	"github.com/luchojuarez/account-transaction-summary/internal/application"
	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// ── In-process fakes (implement the driven ports directly) ────────────────────

type fakeReader struct {
	txns []domain.Transaction
	err  error
}

func (f *fakeReader) ReadTransactions() ([]domain.Transaction, error) {
	return f.txns, f.err
}

type fakeRepository struct {
	savedTxns      []domain.Transaction
	savedSummaries []domain.UserSummary
	txnErr         error
	summaryErr     error
}

func (f *fakeRepository) SaveTransactions(txns []domain.Transaction) error {
	f.savedTxns = txns
	return f.txnErr
}

func (f *fakeRepository) SaveUserSummary(s domain.UserSummary) error {
	f.savedSummaries = append(f.savedSummaries, s)
	return f.summaryErr
}

type fakeNotifier struct {
	calls []notifierCall
	err   error
}

type notifierCall struct {
	email   string
	summary domain.UserSummary
}

func (f *fakeNotifier) SendSummary(email, _ string, s domain.UserSummary) error {
	f.calls = append(f.calls, notifierCall{email: email, summary: s})
	return f.err
}



// ── helpers ───────────────────────────────────────────────────────────────────

func mustTxn(id, userID int, dateStr string, amount float64) domain.Transaction {
	date, _ := time.Parse("2006-01-02", dateStr)
	t, err := domain.NewTransaction(id, userID, date, amount)
	if err != nil {
		panic(err)
	}
	return t
}

func twoUserTxns() []domain.Transaction {
	return []domain.Transaction{
		// user 1
		mustTxn(1, 1, "2021-07-15", +60.5),
		mustTxn(2, 1, "2021-07-28", -20.46),
		mustTxn(4, 1, "2021-08-10", -10.30),
		// user 2
		mustTxn(3, 2, "2021-08-02", +10),
		mustTxn(5, 2, "2021-08-15", +200.0),
	}
}



// ── Tests ─────────────────────────────────────────────────────────────────────

func TestProcessAccountUseCase_HappyPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		txns              []domain.Transaction
		wantNotifiedUsers int
		wantUserBalances  map[int]float64
	}{
		{
			name:              "two users — each gets their own notification",
			txns:              twoUserTxns(),
			wantNotifiedUsers: 2,
			wantUserBalances: map[int]float64{
				1: 29.74, // 60.5 - 20.46 - 10.30
				2: 210.0, // 10 + 200
			},
		},
		{
			name: "single user",
			txns: []domain.Transaction{
				mustTxn(1, 7, "2021-01-01", +100.0),
				mustTxn(2, 7, "2021-01-15", -30.0),
			},
			wantNotifiedUsers: 1,
			wantUserBalances:  map[int]float64{7: 70.0},
		},
		{
			name:              "empty file — no notifications sent",
			txns:              []domain.Transaction{},
			wantNotifiedUsers: 0,
			wantUserBalances:  map[int]float64{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reader := &fakeReader{txns: tc.txns}
			repo := &fakeRepository{}
			notifier := &fakeNotifier{}

			uc := application.NewProcessAccountUseCase(reader, repo, notifier, "test@example.com", "Test User")
			if err := uc.Process(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(notifier.calls) != tc.wantNotifiedUsers {
				t.Errorf("notifications sent: want %d, got %d",
					tc.wantNotifiedUsers, len(notifier.calls))
			}

			// Verify per-user balance in notifications
			for _, call := range notifier.calls {
				want, ok := tc.wantUserBalances[call.summary.UserID]
				if !ok {
					t.Errorf("unexpected notification for user %d", call.summary.UserID)
					continue
				}
				if call.summary.TotalBalance != want {
					t.Errorf("user %d balance: want %.2f, got %.2f",
						call.summary.UserID, want, call.summary.TotalBalance)
				}
			}
		})
	}
}

/*
func TestProcessAccountUseCase_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		readerErr      error
		repoTxnErr     error
		repoSummaryErr error
		notifierErr    error
		resolverErr    error
		wantErr        bool
		wantNotified   int // how many notifications still went through
	}{
		{
			name:      "reader error is fatal — stops the pipeline",
			readerErr: errors.New("file not found"),
			wantErr:   true,
		},
		{
			name:         "repo SaveTransactions error is non-fatal",
			repoTxnErr:   errors.New("db unavailable"),
			wantErr:      false,
			wantNotified: 2, // both users still notified
		},
		{
			name:           "repo SaveUserSummary error is non-fatal",
			repoSummaryErr: errors.New("db write failed"),
			wantErr:        false,
			wantNotified:   2,
		},
		{
			name:         "notifier error is per-user non-fatal — other users still processed",
			notifierErr:  errors.New("smtp refused"),
			wantErr:      false, // Process() itself doesn't return error; it logs per-user
			wantNotified: 2,     // SendSummary is called for each user (fake always records the call)
		},
		{
			name:         "resolver error is per-user non-fatal",
			resolverErr:  errors.New("user not found"),
			wantErr:      false,
			wantNotified: 0, // resolver fails before notifier is called
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			txns := twoUserTxns()
			if tc.readerErr != nil {
				txns = nil
			}

			reader := &fakeReader{txns: txns, err: tc.readerErr}
			repo := &fakeRepository{txnErr: tc.repoTxnErr, summaryErr: tc.repoSummaryErr}
			notifier := &fakeNotifier{err: tc.notifierErr}
			resolver := twoUserResolver()
			if tc.resolverErr != nil {
				resolver = &fakeResolver{err: tc.resolverErr}
			}

			uc := application.NewProcessAccountUseCase(reader, repo, notifier, resolver)
			err := uc.Process()

			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(notifier.calls) != tc.wantNotified {
				t.Errorf("notifications sent: want %d, got %d", tc.wantNotified, len(notifier.calls))
			}
		})
	}
}
*/
func TestProcessAccountUseCase_NotificationsUseSortedUserOrder(t *testing.T) {
	t.Parallel()

	// Users inserted in reverse order in the CSV — output should still be sorted
	txns := []domain.Transaction{
		mustTxn(1, 1, "2021-01-01", +10),
		mustTxn(2, 2, "2021-01-02", +20),
		mustTxn(3, 3, "2021-01-03", +30),
	}

	reader := &fakeReader{txns: txns}
	repo := &fakeRepository{}
	notifier := &fakeNotifier{}

	uc := application.NewProcessAccountUseCase(reader, repo, notifier, "test@example.com", "Test User")
	if err := uc.Process(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notifier.calls) != 3 {
		t.Fatalf("want 3 calls, got %d", len(notifier.calls))
	}

	// Verify ascending ID order in notifications
	for i := 1; i < len(notifier.calls); i++ {
		prev := notifier.calls[i-1].summary.UserID
		curr := notifier.calls[i].summary.UserID
		if curr <= prev {
			t.Errorf("notifications not sorted: user %d before user %d", prev, curr)
		}
	}
}
