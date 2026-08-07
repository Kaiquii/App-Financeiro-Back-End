package assistant

import (
	"Sobra_Ai_Back-end/internal/expenses"
	"strings"
)

func expenseSourceAmount(expense expenses.Expense, source string) float64 {
	normalizedSource := normalizeMoneySource(source)
	if normalizedSource == "" {
		return expense.Amount
	}
	var amount float64
	for _, split := range expenses.PaymentSplitsOrLegacy(expense) {
		if normalizeMoneySource(split.PaymentSource) == normalizedSource {
			amount += split.Amount
		}
	}
	return amount
}

func filterExpensesByPaymentSource(items []expenses.Expense, source string) []expenses.Expense {
	if strings.TrimSpace(source) == "" {
		return items
	}
	filtered := make([]expenses.Expense, 0, len(items))
	for _, expense := range items {
		amount := expenseSourceAmount(expense, source)
		if amount == 0 {
			continue
		}
		// The assistant is answering a question scoped to this payment source.
		expense.Amount = amount
		filtered = append(filtered, expense)
	}
	return filtered
}
