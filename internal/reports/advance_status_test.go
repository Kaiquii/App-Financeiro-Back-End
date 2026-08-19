package reports

import (
	"Sobra_Ai_Back-end/internal/expenses"
	"testing"
	"time"
)

func TestBuildChartResultsUsesAdvanceMonth(t *testing.T) {
	advancedAt := time.Date(2026, time.August, 19, 0, 0, 0, 0, time.Local)
	results := buildChartResults(nil, []expenses.Expense{{
		Amount: 70, Month: 9, Year: 2026, IsAdvanced: true, AdvancedAt: &advancedAt,
	}})

	if results[7].Expense != 70 {
		t.Fatalf("august expense = %.2f, want 70", results[7].Expense)
	}
	if results[8].Expense != 0 {
		t.Fatalf("september expense = %.2f, want 0", results[8].Expense)
	}
}

func TestInstallmentCommitmentsKeepAdvancedInstallmentVisibleButNotOpen(t *testing.T) {
	advancedAt := time.Date(2026, time.August, 19, 0, 0, 0, 0, time.Local)
	installments := []expenses.Expense{
		{ID: 1, SeriesID: "album", Description: "Album", Type: "Parcelada", Amount: 70, Month: 9, Year: 2026, CurrentInstall: 2, Installments: 3, IsAdvanced: true, AdvancedAt: &advancedAt},
		{ID: 2, SeriesID: "album", Description: "Album", Type: "Parcelada", Amount: 70, Month: 10, Year: 2026, CurrentInstall: 3, Installments: 3},
	}

	result := buildInstallmentCommitmentsResponse(installments, map[uint]string{}, 8, 2026, 3, false)
	if result.Summary.PaidTotal != 70 || result.Summary.RemainingTotal != 70 {
		t.Fatalf("unexpected totals: paid=%.2f remaining=%.2f", result.Summary.PaidTotal, result.Summary.RemainingTotal)
	}
	if result.Timeline[1].Total != 0 || len(result.Timeline[1].Installments) != 1 {
		t.Fatalf("advanced september installment must remain visible with zero commitment: %+v", result.Timeline[1])
	}
	if !result.Timeline[1].Installments[0].IsAdvanced {
		t.Fatal("advanced status missing from installment timeline")
	}
	if result.Timeline[2].Total != 70 {
		t.Fatalf("october commitment = %.2f, want 70", result.Timeline[2].Total)
	}
}
