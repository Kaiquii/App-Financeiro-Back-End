package reports

import (
	"Sobra_Ai_Back-end/internal/expenses"
	"Sobra_Ai_Back-end/internal/incomes"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestParseExportOptionsForFullReport(t *testing.T) {
	options, err := parseExportOptions(url.Values{
		"type":                          {exportTypeFullReport},
		"month":                         {"1"},
		"year":                          {"2026"},
		"format":                        {"csv"},
		"months":                        {"18"},
		"include_current_month_as_paid": {"true"},
	})
	if err != nil {
		t.Fatalf("expected valid options, got %v", err)
	}
	if options.CompareMonth != 12 || options.CompareYear != 2025 {
		t.Fatalf("expected previous month 12/2025, got %d/%d", options.CompareMonth, options.CompareYear)
	}
	if options.Months != 18 || !options.IncludeCurrentMonthPaid {
		t.Fatalf("unexpected installment options: %+v", options)
	}
}

func TestParseExportOptionsRejectsInvalidParameters(t *testing.T) {
	tests := []struct {
		name   string
		values url.Values
	}{
		{name: "missing type", values: url.Values{"month": {"6"}, "year": {"2026"}}},
		{name: "invalid type", values: url.Values{"type": {"pdf"}, "month": {"6"}, "year": {"2026"}}},
		{name: "missing format", values: url.Values{"type": {exportTypeSummary}, "month": {"6"}, "year": {"2026"}}},
		{name: "invalid format", values: url.Values{"type": {exportTypeSummary}, "month": {"6"}, "year": {"2026"}, "format": {"pdf"}}},
		{name: "invalid month", values: url.Values{"type": {exportTypeSummary}, "month": {"13"}, "year": {"2026"}}},
		{name: "invalid year", values: url.Values{"type": {exportTypeSummary}, "month": {"6"}, "year": {"1999"}}},
		{name: "incomplete comparison", values: url.Values{"type": {exportTypeMonthComparison}, "month": {"6"}, "year": {"2026"}, "compare_month": {"5"}}},
		{name: "invalid months", values: url.Values{"type": {exportTypeInstallmentCommitments}, "month": {"6"}, "year": {"2026"}, "months": {"61"}}},
		{name: "invalid paid flag", values: url.Values{"type": {exportTypeFullReport}, "month": {"6"}, "year": {"2026"}, "include_current_month_as_paid": {"yes"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseExportOptions(test.values); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestGenerateExpensesCSVPreservesSpecialCharacters(t *testing.T) {
	dataset := exportDataset{
		Options: exportOptions{ReportType: exportTypeExpenses},
		Expenses: []expenses.Expense{{
			Description:   "Mercado, centro",
			Notes:         "Linha 1\n\"Linha 2\"",
			CategoryID:    1,
			PaymentSource: "Salario",
			Type:          "Unica",
			Amount:        120.5,
			Date:          time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		}},
		CategoryNames: map[uint]string{1: "Alimentacao"},
	}

	content, err := generateExportCSV(dataset)
	if err != nil {
		t.Fatalf("failed to generate CSV: %v", err)
	}
	if len(content) < 3 || string(content[:3]) != "\xEF\xBB\xBF" {
		t.Fatal("expected UTF-8 BOM")
	}

	records := readCSVRecords(t, content)
	if len(records) != 2 {
		t.Fatalf("expected header and one row, got %d records", len(records))
	}
	if records[1][1] != "Mercado, centro" || records[1][7] != "Linha 1\n\"Linha 2\"" {
		t.Fatalf("special characters were not preserved: %#v", records[1])
	}
	if records[1][6] != "120,50" {
		t.Fatalf("expected monetary value with two decimals, got %q", records[1][6])
	}
}

func TestGenerateExpensesCSVPreventsSpreadsheetFormulaInjection(t *testing.T) {
	dataset := exportDataset{
		Options: exportOptions{ReportType: exportTypeExpenses},
		Expenses: []expenses.Expense{{
			Description: "=HYPERLINK(\"https://example.com\")",
			Notes:       "+cmd|' /C calc'!A0",
			Amount:      10,
			Date:        time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		}},
		CategoryNames: map[uint]string{},
	}

	content, err := generateExportCSV(dataset)
	if err != nil {
		t.Fatalf("failed to generate CSV: %v", err)
	}
	records := readCSVRecords(t, content)
	if records[1][1][0] != '\'' || records[1][7][0] != '\'' {
		t.Fatalf("expected dangerous text cells to be escaped: %#v", records[1])
	}
}

func TestGenerateExpensesCSVDoesNotFormatInstallmentAsDate(t *testing.T) {
	dataset := exportDataset{
		Options: exportOptions{ReportType: exportTypeExpenses},
		Expenses: []expenses.Expense{{
			Type:           "Parcelada",
			CurrentInstall: 3,
			Installments:   5,
			Amount:         100,
			Date:           time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		}},
		CategoryNames: map[uint]string{},
	}

	content, err := generateExportCSV(dataset)
	if err != nil {
		t.Fatalf("failed to generate CSV: %v", err)
	}
	records := readCSVRecords(t, content)
	if records[1][5] != "3 de 5" {
		t.Fatalf("expected Excel-safe installment label, got %q", records[1][5])
	}
}

func TestGenerateAllExportTypes(t *testing.T) {
	dataset := completeExportTestDataset()
	types := []string{
		exportTypeExpenses,
		exportTypeIncomes,
		exportTypeCategories,
		exportTypeSummary,
		exportTypeMonthComparison,
		exportTypeInstallmentCommitments,
		exportTypeFullReport,
	}

	for _, reportType := range types {
		t.Run(reportType, func(t *testing.T) {
			dataset.Options.ReportType = reportType
			content, err := generateExportCSV(dataset)
			if err != nil {
				t.Fatalf("failed to generate %s: %v", reportType, err)
			}
			records := readCSVRecords(t, content)
			if len(records) < 2 {
				t.Fatalf("expected %s to contain a header and data", reportType)
			}
		})
	}
}

func TestFullReportUsesNamedSectionsAndHeaders(t *testing.T) {
	dataset := completeExportTestDataset()
	dataset.Options.ReportType = exportTypeFullReport

	content, err := generateExportCSV(dataset)
	if err != nil {
		t.Fatalf("failed to generate full report: %v", err)
	}
	text := string(content)
	if strings.Contains(text, "Campo1") || strings.Contains(text, "Campo2") {
		t.Fatal("full report must not contain generic field names")
	}
	for _, expected := range []string{
		"RESUMO MENSAL",
		"RECEITAS",
		"DESPESAS",
		"RESUMO POR CATEGORIA",
		"COMPARATIVO - RESUMO",
		"COMPARATIVO - CATEGORIAS",
		"COMPARATIVO - FONTES DE PAGAMENTO",
		"COMPARATIVO - TIPOS DE DESPESA",
		"INSIGHTS DO COMPARATIVO",
		"COMPROMISSOS PARCELADOS - RESUMO",
		"COMPROMISSOS PARCELADOS - COMPRAS",
		"COMPROMISSOS PARCELADOS - LINHA DO TEMPO",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("full report is missing section %q", expected)
		}
	}
}

func TestEmptyExportsKeepTheirHeader(t *testing.T) {
	for _, reportType := range []string{exportTypeExpenses, exportTypeIncomes, exportTypeCategories} {
		dataset := exportDataset{Options: exportOptions{ReportType: reportType}, CategoryNames: map[uint]string{}}
		content, err := generateExportCSV(dataset)
		if err != nil {
			t.Fatalf("failed to generate empty %s: %v", reportType, err)
		}
		if records := readCSVRecords(t, content); len(records) != 1 {
			t.Fatalf("expected only a header for %s, got %d records", reportType, len(records))
		}
	}
}

func TestExportReportRequiresAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodGet, "/api/reports/export?type=summary&month=6&year=2026", nil)
	exportReport(context)
	if context.Writer.Status() != http.StatusUnauthorized {
		t.Fatalf("expected 401 without user, got %d", context.Writer.Status())
	}
}

func completeExportTestDataset() exportDataset {
	incomeItems := []incomes.Income{{Source: "Salario", Amount: 3000, Month: 6, Year: 2026}}
	expenseItems := []expenses.Expense{{
		ID: 1, SeriesID: "series-1", Description: "Notebook", Notes: "Trabalho", CategoryID: 1,
		PaymentSource: "Salario", Type: "Parcelada", Amount: 300, Date: time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Month: 6, Year: 2026, CurrentInstall: 1, Installments: 10,
	}}
	categoryNames := map[uint]string{1: "Eletronicos"}
	comparison := buildMonthComparisonResponse(incomeItems, nil, expenseItems, nil, categoryNames, 6, 2026, 5, 2026)
	commitments := buildInstallmentCommitmentsResponse(expenseItems, categoryNames, 6, 2026, 12, false)
	return exportDataset{
		Options:         exportOptions{Month: 6, Year: 2026},
		Incomes:         incomeItems,
		Expenses:        expenseItems,
		CategoryNames:   categoryNames,
		CategorySummary: buildExportCategorySummary(expenseItems, categoryNames),
		Summary:         buildMonthlyExportSummary(incomeItems, expenseItems),
		Comparison:      &comparison,
		Commitments:     &commitments,
	}
}

func readCSVRecords(t *testing.T, content []byte) [][]string {
	t.Helper()
	text := strings.TrimPrefix(string(content), "\xEF\xBB\xBF")
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}
	return records
}
