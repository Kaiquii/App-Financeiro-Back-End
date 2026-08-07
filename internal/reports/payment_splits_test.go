package reports

import (
	"Sobra_Ai_Back-end/internal/expenses"
	"testing"
)

func TestBuildMonthlyExportSummaryUsesPaymentSplits(t *testing.T) {
	summary := buildMonthlyExportSummary(nil, []expenses.Expense{{
		Amount: 1200,
		PaymentSplits: []expenses.PaymentSplit{
			{PaymentSource: "Salário", Amount: 1000},
			{PaymentSource: "Renda Extra", Amount: 200},
		},
	}})

	if summary.TotalExpense != 1200 {
		t.Fatalf("total expense = %v, want 1200", summary.TotalExpense)
	}
	if summary.SpentSalary != 1000 || summary.SpentExtraIncome != 200 {
		t.Fatalf("spent salary/extra = %v/%v, want 1000/200", summary.SpentSalary, summary.SpentExtraIncome)
	}
}
