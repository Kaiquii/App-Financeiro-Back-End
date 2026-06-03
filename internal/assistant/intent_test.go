package assistant

import "testing"

func TestNormalizeInterpretedIntentKeepsSpecificCategoryQuery(t *testing.T) {
	intent := normalizeInterpretedIntent(financialIntent{
		Intent:       "all_category_expenses",
		Month:        6,
		Year:         2026,
		CategoryName: "Comida",
	})

	if intent.Intent != "category_expenses" {
		t.Fatalf("expected category_expenses, got %q", intent.Intent)
	}
}

func TestNormalizeInterpretedIntentDoesNotChangeAllCategoriesWithoutCategoryName(t *testing.T) {
	intent := normalizeInterpretedIntent(financialIntent{
		Intent: "all_category_expenses",
		Month:  6,
		Year:   2026,
	})

	if intent.Intent != "all_category_expenses" {
		t.Fatalf("expected all_category_expenses, got %q", intent.Intent)
	}
}

func TestNormalizeInterpretedIntentRedirectsMonthlySummaryWithCategoryName(t *testing.T) {
	intent := normalizeInterpretedIntent(financialIntent{
		Intent:       "monthly_summary",
		Month:        6,
		Year:         2026,
		CategoryName: "Comida",
	})

	if intent.Intent != "category_expenses" {
		t.Fatalf("expected category_expenses, got %q", intent.Intent)
	}
}
