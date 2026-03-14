package domain_test

import (
	"testing"
	"time"

	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

func TestNewTransaction(t *testing.T) {
	t.Parallel()

	validDate := time.Date(2021, time.July, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		id       int
		userID   int
		date     time.Time
		amount   float64
		wantType domain.TransactionType
		wantErr  bool
	}{
		{
			name: "positive amount becomes credit",
			id:   1, userID: 1, date: validDate, amount: +60.5,
			wantType: domain.Credit,
		},
		{
			name: "negative amount becomes debit",
			id:   2, userID: 2, date: validDate, amount: -20.46,
			wantType: domain.Debit,
		},
		{
			name: "large credit is valid",
			id:   3, userID: 3, date: validDate, amount: +999999.99,
			wantType: domain.Credit,
		},
		{
			name: "zero amount is invalid",
			id:   4, userID: 4, date: validDate, amount: 0,
			wantErr: true,
		},
		{
			name: "zero date is invalid",
			id:   7, userID: 7, date: time.Time{}, amount: 10,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NewTransaction(tc.id, tc.userID, tc.date, tc.amount)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Type != tc.wantType {
				t.Errorf("Type: want %s, got %s", tc.wantType, got.Type)
			}
			if got.Amount != tc.amount {
				t.Errorf("Amount: want %.2f, got %.2f", tc.amount, got.Amount)
			}
			if !got.Date.Equal(tc.date) {
				t.Errorf("Date: want %v, got %v", tc.date, got.Date)
			}
			if got.UserID != tc.userID {
				t.Errorf("UserID: want %d, got %d", tc.userID, got.UserID)
			}
		})
	}
}
