package reports

import (
	"Sobra_Ai_Back-end/internal/expenses"
	"Sobra_Ai_Back-end/internal/incomes"
	"testing"
)

func TestBuildMonthComparisonResponse(t *testing.T) {
	currentIncomes := []incomes.Income{
		{Amount: 3500, Month: 6, Year: 2026},
	}
	previousIncomes := []incomes.Income{
		{Amount: 3000, Month: 5, Year: 2026},
	}
	currentExpenses := []expenses.Expense{
		{CategoryID: 1, Amount: 620, PaymentSource: "Salario", Type: "Unica", Month: 6, Year: 2026},
		{CategoryID: 2, Amount: 260, PaymentSource: "Adiantamento", Type: "Fixa", Month: 6, Year: 2026},
	}
	previousExpenses := []expenses.Expense{
		{CategoryID: 1, Amount: 500, PaymentSource: "Salario", Type: "Unica", Month: 5, Year: 2026},
		{CategoryID: 2, Amount: 300, PaymentSource: "Adiantamento", Type: "Fixa", Month: 5, Year: 2026},
	}
	categoryNames := map[uint]string{
		1: "Alimentacao",
		2: "Transporte",
	}

	response := buildMonthComparisonResponse(
		currentIncomes,
		previousIncomes,
		currentExpenses,
		previousExpenses,
		categoryNames,
		6,
		2026,
		5,
		2026,
	)

	if response.Summary.CurrentIncome != 3500 {
		t.Fatalf("expected current income 3500, got %.2f", response.Summary.CurrentIncome)
	}
	if response.ComparedMonth != 5 || response.ComparedYear != 2026 {
		t.Fatalf("expected compared period 5/2026, got %d/%d", response.ComparedMonth, response.ComparedYear)
	}
	if response.Summary.IncomeDifference != 500 {
		t.Fatalf("expected income difference 500, got %.2f", response.Summary.IncomeDifference)
	}
	if response.Summary.CurrentExpense != 880 {
		t.Fatalf("expected current expense 880, got %.2f", response.Summary.CurrentExpense)
	}
	if response.Summary.ExpenseDifference != 80 {
		t.Fatalf("expected expense difference 80, got %.2f", response.Summary.ExpenseDifference)
	}
	if response.Summary.BalanceDifference != 420 {
		t.Fatalf("expected balance difference 420, got %.2f", response.Summary.BalanceDifference)
	}
	if len(response.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(response.Categories))
	}
	if response.Categories[0].CategoryName != "Alimentacao" || response.Categories[0].Difference != 120 || response.Categories[0].Status != "subiu" {
		t.Fatalf("unexpected first category comparison: %+v", response.Categories[0])
	}
	if len(response.Insights) == 0 {
		t.Fatal("expected insights")
	}
}

func TestPreviousMonthYearHandlesJanuary(t *testing.T) {
	month, year := previousMonthYear(1, 2026)
	if month != 12 || year != 2025 {
		t.Fatalf("expected 12/2025, got %d/%d", month, year)
	}
}
