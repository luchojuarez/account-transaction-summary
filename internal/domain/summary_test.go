package domain_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func mustTxn(t *testing.T, id, userID int, dateStr string, amount float64) domain.Transaction {
	t.Helper()
	date, err := time.Parse(domain.DateLayout, dateStr)
	if err != nil {
		t.Fatalf("mustTxn: invalid date %q: %v", dateStr, err)
	}
	txn, err := domain.NewTransaction(id, userID, date, amount)
	if err != nil {
		t.Fatalf("mustTxn: %v", err)
	}
	return txn
}

func floatPtr(v float64) *float64 { return &v }

func assertOptFloat(t *testing.T, field string, got, want *float64) {
	t.Helper()
	if want == nil && got == nil {
		return
	}
	if want == nil {
		t.Errorf("%s: want nil, got %.2f", field, *got)
		return
	}
	if got == nil {
		t.Errorf("%s: want %.2f, got nil", field, *want)
		return
	}
	if *got != *want {
		t.Errorf("%s: want %.2f, got %.2f", field, *want, *got)
	}
}

// findUser returns the UserSummary for the given userID or fails the test.
func findUser(t *testing.T, summaries []domain.UserSummary, userID int) domain.UserSummary {
	t.Helper()
	for _, s := range summaries {
		if s.UserID == userID {
			return s
		}
	}
	t.Fatalf("no summary found for userID %d", userID)
	return domain.UserSummary{}
}

// ─── Grouping ─────────────────────────────────────────────────────────────────

func TestNewUserSummaries_Grouping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		txns          []domain.Transaction
		wantUserCount int
		wantUserIDs   []int // expected order (sorted ascending)
	}{
		{
			name: "two users produce two summaries sorted by userID",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-07-15", +60.5),
				mustTxn(t, 2, 1, "2021-07-28", -20.46),
				mustTxn(t, 3, 2, "2021-08-02", +10),
				mustTxn(t, 4, 2, "2021-08-10", -10.30),
			},
			wantUserCount: 2,
			wantUserIDs:   []int{1, 2},
		},
		{
			name: "three users interleaved in CSV produce sorted summaries",
			txns: []domain.Transaction{
				mustTxn(t, 1, 3, "2021-01-01", +10),
				mustTxn(t, 2, 1, "2021-01-01", +20),
				mustTxn(t, 3, 2, "2021-01-01", +30),
				mustTxn(t, 4, 3, "2021-01-02", +10),
			},
			wantUserCount: 3,
			wantUserIDs:   []int{1, 2, 3},
		},
		{
			name: "single user produces one summary",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-01-01", +100),
			},
			wantUserCount: 1,
			wantUserIDs:   []int{1},
		},
		{
			name:          "empty input produces no summaries",
			txns:          []domain.Transaction{},
			wantUserCount: 0,
			wantUserIDs:   []int{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := domain.NewUserSummaries(tc.txns, 0)

			if len(got) != tc.wantUserCount {
				t.Fatalf("user count: want %d, got %d", tc.wantUserCount, len(got))
			}
			for i, wantUID := range tc.wantUserIDs {
				if got[i].UserID != wantUID {
					t.Errorf("summaries[%d].UserID: want %d, got %d", i, wantUID, got[i].UserID)
				}
			}
		})
	}
}

// ─── Balance and overall averages ────────────────────────────────────────────

func TestNewUserSummaries_BalanceAndAverages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		txns       []domain.Transaction
		wantByUser map[int]struct {
			balance    float64
			totalCount int
			avgCredit  float64
			avgDebit   float64
		}
	}{
		{
			name: "example CSV: two users computed independently",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-07-15", +60.5),
				mustTxn(t, 2, 1, "2021-07-28", -20.46),
				mustTxn(t, 3, 2, "2021-08-02", +10),
				mustTxn(t, 4, 1, "2021-08-10", -10.30),
			},
			wantByUser: map[int]struct {
				balance    float64
				totalCount int
				avgCredit  float64
				avgDebit   float64
			}{
				1: {balance: 29.74, totalCount: 3, avgCredit: 60.5, avgDebit: -15.38},
				2: {balance: 10.00, totalCount: 1, avgCredit: 10.0, avgDebit: 0},
			},
		},
		{
			name: "user with only debits has negative balance and zero avg credit",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-03-01", -100),
				mustTxn(t, 2, 1, "2021-03-15", -200),
			},
			wantByUser: map[int]struct {
				balance    float64
				totalCount int
				avgCredit  float64
				avgDebit   float64
			}{
				1: {balance: -300, totalCount: 2, avgCredit: 0, avgDebit: -150},
			},
		},
		{
			name: "avg is rounded to 2 decimal places",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-01-01", +10),
				mustTxn(t, 2, 1, "2021-01-02", -20),
				mustTxn(t, 3, 1, "2021-01-03", +30),
				mustTxn(t, 4, 2, "2021-01-01", -15),
				mustTxn(t, 5, 2, "2021-01-02", -25),
			},
			wantByUser: map[int]struct {
				balance    float64
				totalCount int
				avgCredit  float64
				avgDebit   float64
			}{
				1: {balance: 20, totalCount: 3, avgCredit: 20, avgDebit: -20},
				2: {balance: -40, totalCount: 2, avgCredit: 0, avgDebit: -20},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summaries := domain.NewUserSummaries(tc.txns, 0)

			if len(summaries) != len(tc.wantByUser) {
				t.Fatalf("user count: want %d, got %d", len(tc.wantByUser), len(summaries))
			}

			for userID, want := range tc.wantByUser {
				s := findUser(t, summaries, userID)

				if s.TotalBalance != want.balance {
					t.Errorf("user %d TotalBalance: want %.2f, got %.2f", userID, want.balance, s.TotalBalance)
				}
				if s.TotalTransactions != want.totalCount {
					t.Errorf("user %d TotalTransactions: want %d, got %d", userID, want.totalCount, s.TotalTransactions)
				}
				if s.OverallAvgCredit != want.avgCredit {
					t.Errorf("user %d OverallAvgCredit: want %.2f, got %.2f", userID, want.avgCredit, s.OverallAvgCredit)
				}
				if s.OverallAvgDebit != want.avgDebit {
					t.Errorf("user %d OverallAvgDebit: want %.2f, got %.2f", userID, want.avgDebit, s.OverallAvgDebit)
				}
			}
		})
	}
}

// ─── Monthly breakdown ────────────────────────────────────────────────────────

func TestNewUserSummaries_MonthlyCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		txns           []domain.Transaction
		userID         int
		wantMonthCount int
		wantMonths     map[string]int
	}{
		{
			name: "user 1 spans july and august; user 2 only august",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-07-15", +60.5),
				mustTxn(t, 2, 1, "2021-07-28", -20.46),
				mustTxn(t, 3, 2, "2021-08-02", +10),
				mustTxn(t, 4, 1, "2021-08-10", -10.30),
			},
			userID:         1,
			wantMonthCount: 2,
			wantMonths:     map[string]int{"July 2021": 2, "August 2021": 1},
		},
		{
			name: "user 2 has only one month",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-07-15", +60.5),
				mustTxn(t, 2, 2, "2021-08-02", +10),
				mustTxn(t, 3, 2, "2021-08-25", -5),
			},
			userID:         2,
			wantMonthCount: 1,
			wantMonths:     map[string]int{"August 2021": 2},
		},
		{
			name: "same month in different years are separate buckets",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2020-01-10", +50),
				mustTxn(t, 2, 1, "2021-01-10", +50),
			},
			userID:         1,
			wantMonthCount: 2,
			wantMonths:     map[string]int{"January 2020": 1, "January 2021": 1},
		},
		{
			name: "months are returned in chronological order",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-12-01", +10),
				mustTxn(t, 2, 1, "2021-01-01", +10),
				mustTxn(t, 3, 1, "2021-06-01", +10),
			},
			userID:         1,
			wantMonthCount: 3,
			wantMonths:     map[string]int{"January 2021": 1, "June 2021": 1, "December 2021": 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summaries := domain.NewUserSummaries(tc.txns, 0)
			s := findUser(t, summaries, tc.userID)

			if len(s.Monthly) != tc.wantMonthCount {
				t.Errorf("user %d month count: want %d, got %d", tc.userID, tc.wantMonthCount, len(s.Monthly))
			}

			// Verify ascending chronological order
			for i := 1; i < len(s.Monthly); i++ {
				prev, cur := s.Monthly[i-1], s.Monthly[i]
				if cur.Year < prev.Year || (cur.Year == prev.Year && cur.MonthNum <= prev.MonthNum) {
					t.Errorf("user %d months not sorted at index %d: %s %d >= %s %d",
						tc.userID, i, prev.MonthName, prev.Year, cur.MonthName, cur.Year)
				}
			}

			for _, ms := range s.Monthly {
				key := fmt.Sprintf("%s %d", ms.MonthName, ms.Year)
				want, ok := tc.wantMonths[key]
				if !ok {
					t.Errorf("user %d: unexpected month %q", tc.userID, key)
					continue
				}
				if ms.Count != want {
					t.Errorf("user %d %s count: want %d, got %d", tc.userID, key, want, ms.Count)
				}
			}
		})
	}
}

// ─── Per-month avg credit / debit ─────────────────────────────────────────────

func TestNewUserSummaries_MonthAvgCreditDebit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		txns          []domain.Transaction
		userID        int
		month         string
		wantAvgCredit *float64
		wantAvgDebit  *float64
	}{
		{
			name: "one credit one debit in same month",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-07-15", +60.5),
				mustTxn(t, 2, 1, "2021-07-28", -10.3),
			},
			userID:        1,
			month:         "July 2021",
			wantAvgCredit: floatPtr(60.5),
			wantAvgDebit:  floatPtr(-10.3),
		},
		{
			name: "only credits: avg debit is nil",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-04-01", +100),
				mustTxn(t, 2, 1, "2021-04-15", +200),
			},
			userID:        1,
			month:         "April 2021",
			wantAvgCredit: floatPtr(150),
			wantAvgDebit:  nil,
		},
		{
			name: "only debits: avg credit is nil",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-06-05", -40),
				mustTxn(t, 2, 1, "2021-06-20", -60),
			},
			userID:        1,
			month:         "June 2021",
			wantAvgCredit: nil,
			wantAvgDebit:  floatPtr(-50),
		},
		{
			name: "user 2 month is independent from user 1 same month",
			txns: []domain.Transaction{
				mustTxn(t, 1, 1, "2021-08-01", +500),
				mustTxn(t, 2, 2, "2021-08-02", +10),
				mustTxn(t, 3, 2, "2021-08-15", -30),
			},
			userID:        2,
			month:         "August 2021",
			wantAvgCredit: floatPtr(10),
			wantAvgDebit:  floatPtr(-30),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summaries := domain.NewUserSummaries(tc.txns, 0)
			s := findUser(t, summaries, tc.userID)

			var ms *domain.MonthSummary
			for i := range s.Monthly {
				key := fmt.Sprintf("%s %d", s.Monthly[i].MonthName, s.Monthly[i].Year)
				if key == tc.month {
					ms = &s.Monthly[i]
					break
				}
			}
			if ms == nil {
				t.Fatalf("user %d: month %q not found in summary", tc.userID, tc.month)
			}

			assertOptFloat(t, "AvgCredit", ms.AvgCredit, tc.wantAvgCredit)
			assertOptFloat(t, "AvgDebit", ms.AvgDebit, tc.wantAvgDebit)
		})
	}
}
