package domain

import (
	"math"
	"time"
	"sort"
)

// MonthSummary holds aggregated stats for one calendar month.
type MonthSummary struct {
	MonthName string
	MonthNum  int
	Year      int
	Count     int
	AvgCredit *float64 // nil when there are no credits that month
	AvgDebit  *float64 // nil when there are no debits that month
}

// UserSummary holds the full computed summary for all transactions.
type UserSummary struct {
	UserID            int
	TotalBalance      float64
	TotalTransactions int
	OverallAvgCredit  float64
	OverallAvgDebit   float64
	Monthly           []MonthSummary
}

// monthKey uniquely identifies a year-month bucket.
type monthKey struct {
	year  int
	month time.Month
}

type monthAccum struct {
	credits []float64
	debits  []float64
}

// NewUserSummaries computes per-user aggregate summaries across all transactions.
func NewUserSummaries(txns []Transaction) []UserSummary {
	if len(txns) == 0 {
		return nil
	}

	// Group transactions by user.
	byUser := make(map[int][]Transaction)
	for _, t := range txns {
		byUser[t.UserID] = append(byUser[t.UserID], t)
	}

	summaries := make([]UserSummary, 0, len(byUser))
	for userID, userTxns := range byUser {
		us := computeUserSummary(userTxns)
		us.UserID = userID
		summaries = append(summaries, us)
	}

	// Stable deterministic order: ascending userID.
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UserID < summaries[j].UserID
	})

	return summaries
}

func computeUserSummary(txns []Transaction) UserSummary {
	monthly := make(map[monthKey]*monthAccum)
	var totalBalance float64
	var allCredits, allDebits []float64

	for _, t := range txns {
		totalBalance += t.Amount
		key := monthKey{year: t.Date.Year(), month: t.Date.Month()}
		if _, ok := monthly[key]; !ok {
			monthly[key] = &monthAccum{}
		}
		if t.Type == Credit {
			monthly[key].credits = append(monthly[key].credits, t.Amount)
			allCredits = append(allCredits, t.Amount)
		} else {
			monthly[key].debits = append(monthly[key].debits, t.Amount)
			allDebits = append(allDebits, t.Amount)
		}
	}

	// Sort month keys chronologically
	keys := make([]monthKey, 0, len(monthly))
	for k := range monthly {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].year != keys[j].year {
			return keys[i].year < keys[j].year
		}
		return keys[i].month < keys[j].month
	})

	months := make([]MonthSummary, 0, len(keys))
	for _, k := range keys {
		acc := monthly[k]
		ms := MonthSummary{
			MonthName: k.month.String(),
			MonthNum:  int(k.month),
			Year:      k.year,
			Count:     len(acc.credits) + len(acc.debits),
		}
		if len(acc.credits) > 0 {
			v := round2(mean(acc.credits))
			ms.AvgCredit = &v
		}
		if len(acc.debits) > 0 {
			v := round2(mean(acc.debits))
			ms.AvgDebit = &v
		}
		months = append(months, ms)
	}

	return UserSummary{
		TotalBalance:      round2(totalBalance),
		TotalTransactions: len(txns),
		OverallAvgCredit:  round2(mean(allCredits)),
		OverallAvgDebit:   round2(mean(allDebits)),
		Monthly:           months,
	}
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
