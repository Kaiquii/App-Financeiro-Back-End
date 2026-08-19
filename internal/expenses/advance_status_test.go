package expenses

import (
	"testing"
	"time"
)

func TestCanBeAdvanced(t *testing.T) {
	if !CanBeAdvanced("Única") || !CanBeAdvanced("Unica") || !CanBeAdvanced("Parcelada") {
		t.Fatal("unique and installment expenses should support advance status")
	}
	if CanBeAdvanced("Fixa") {
		t.Fatal("fixed expenses must not support advance status")
	}
}

func TestEffectiveMonthYearUsesAdvanceDate(t *testing.T) {
	advancedAt := time.Date(2026, time.August, 19, 0, 0, 0, 0, time.Local)
	expense := Expense{Month: 9, Year: 2026, IsAdvanced: true, AdvancedAt: &advancedAt}
	month, year := EffectiveMonthYear(expense)
	if month != 8 || year != 2026 {
		t.Fatalf("effective period = %d/%d, want 8/2026", month, year)
	}
}

func TestEffectiveMonthYearKeepsScheduledPeriodForNormalExpense(t *testing.T) {
	month, year := EffectiveMonthYear(Expense{Month: 9, Year: 2026})
	if month != 9 || year != 2026 {
		t.Fatalf("effective period = %d/%d, want 9/2026", month, year)
	}
}

func TestValidateAdvanceDate(t *testing.T) {
	scheduled := time.Date(2026, time.September, 10, 0, 0, 0, 0, time.Local)
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.Local)

	if err := validateAdvanceDate(time.Date(2026, time.August, 19, 0, 0, 0, 0, time.Local), scheduled, now); err != nil {
		t.Fatalf("valid advance date rejected: %v", err)
	}
	if err := validateAdvanceDate(time.Date(2026, time.September, 10, 0, 0, 0, 0, time.Local), scheduled, now); err == nil {
		t.Fatal("scheduled date should not be accepted as an advance date")
	}
	if err := validateAdvanceDate(time.Date(2026, time.August, 21, 0, 0, 0, 0, time.Local), scheduled, now); err == nil {
		t.Fatal("future advance date should be rejected")
	}
}
