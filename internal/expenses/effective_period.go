package expenses

import (
	"time"

	"gorm.io/gorm"
)

// ApplyEffectivePeriod filters expenses by the period in which they affect
// financial totals. Normal expenses use their scheduled month/year; advanced
// expenses use advanced_at instead.
func ApplyEffectivePeriod(query *gorm.DB, month int, year int) *gorm.DB {
	start, end := periodBounds(month, year)
	return query.Where(`(
		(is_advanced = FALSE AND month = ? AND year = ?)
		OR
		(is_advanced = TRUE AND advanced_at >= ? AND advanced_at < ?)
	)
	`, month, year, start, end)
}

// ApplyEffectiveYear is the yearly counterpart of ApplyEffectivePeriod.
func ApplyEffectiveYear(query *gorm.DB, year int) *gorm.DB {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(1, 0, 0)
	return query.Where(`(
		(is_advanced = FALSE AND year = ?)
		OR
		(is_advanced = TRUE AND advanced_at >= ? AND advanced_at < ?)
	)
	`, year, start, end)
}

// EffectiveMonthYear returns the period used by charts and other in-memory
// aggregations. Invalid legacy advanced rows fall back to their scheduled date.
func EffectiveMonthYear(expense Expense) (int, int) {
	if expense.IsAdvanced && expense.AdvancedAt != nil {
		advancedAt := expense.AdvancedAt.In(time.Local)
		return int(advancedAt.Month()), advancedAt.Year()
	}
	return expense.Month, expense.Year
}

// CanBeAdvanced deliberately excludes fixed expenses. New expense types must
// opt in explicitly instead of inheriting this financial behavior by accident.
func CanBeAdvanced(expenseType string) bool {
	normalized := normalizeExpenseType(expenseType)
	return normalized == "Única" || normalized == "Parcelada"
}

func periodBounds(month int, year int) (time.Time, time.Time) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	return start, start.AddDate(0, 1, 0)
}
