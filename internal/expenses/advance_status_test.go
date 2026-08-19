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
	scheduled := time.Date(2026, time.September, 20, 0, 0, 0, 0, time.Local)
	tests := []struct {
		name      string
		date      time.Time
		wantError bool
	}{
		{name: "future date in august", date: time.Date(2026, time.August, 20, 0, 0, 0, 0, time.Local)},
		{name: "last day of august", date: time.Date(2026, time.August, 31, 0, 0, 0, 0, time.Local)},
		{name: "day before scheduled date", date: time.Date(2026, time.September, 19, 0, 0, 0, 0, time.Local)},
		{name: "same as scheduled date", date: time.Date(2026, time.September, 20, 0, 0, 0, 0, time.Local), wantError: true},
		{name: "after scheduled date", date: time.Date(2026, time.September, 21, 0, 0, 0, 0, time.Local), wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateAdvanceDate(test.date, scheduled)
			if test.wantError && err == nil {
				t.Fatal("expected advance date to be rejected")
			}
			if !test.wantError && err != nil {
				t.Fatalf("valid advance date rejected: %v", err)
			}
		})
	}
}
