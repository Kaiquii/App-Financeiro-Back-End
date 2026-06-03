package assistant

import (
	"strings"
	"testing"
)

func TestLocalToolReplyMonthlySummaryExplainsSourcesAndBalance(t *testing.T) {
	reply := localToolReply("get_monthly_summary", map[string]any{
		"month":                    6,
		"year":                     2026,
		"total_income":             2802.00,
		"total_expense":            371.58,
		"total_gasto_salario":      315.99,
		"total_gasto_adiantamento": 20.59,
		"total_gasto_renda_extra":  35.00,
		"total_geral_disponivel":   2430.42,
	})

	expectedParts := []string{
		"Em junho/2026, voce gastou R$ 371,58.",
		"De onde saiu:",
		"Salario: R$ 315,99",
		"Adiantamento: R$ 20,59",
		"Renda Extra: R$ 35,00",
		"Entradas do mes: R$ 2.802,00",
		"Saldo restante: R$ 2.430,42.",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(reply, expected) {
			t.Fatalf("expected reply to contain %q, got %q", expected, reply)
		}
	}
}

func TestFormatMoneyUsesBrazilianThousandsSeparator(t *testing.T) {
	tests := map[float64]string{
		371.58:     "371,58",
		2802.00:    "2.802,00",
		1234567.89: "1.234.567,89",
		-2430.42:   "-2.430,42",
	}

	for value, expected := range tests {
		if got := formatMoney(value); got != expected {
			t.Fatalf("formatMoney(%v) = %q, want %q", value, got, expected)
		}
	}
}

func TestLocalToolReplyCategorySummaryShowsMonthlyPercentage(t *testing.T) {
	reply := localToolReply("get_category_expenses", map[string]any{
		"month":          6,
		"year":           2026,
		"category_name":  "Comida",
		"payment_source": "",
		"total_expense":  300.00,
		"monthly_total":  371.58,
		"percentage":     80.74,
	})

	expectedParts := []string{
		"Em junho/2026, voce gastou R$ 300,00 com Comida.",
		"Isso representa 80,74% dos seus gastos do mes.",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(reply, expected) {
			t.Fatalf("expected reply to contain %q, got %q", expected, reply)
		}
	}
}
