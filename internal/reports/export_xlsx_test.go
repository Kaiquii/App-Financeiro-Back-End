package reports

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"Sobra_Ai_Back-end/internal/expenses"

	"github.com/xuri/excelize/v2"
)

func TestParseExportOptionsAcceptsXLSX(t *testing.T) {
	options, err := parseExportOptions(mapValues(
		"type", exportTypeFullReport,
		"month", "7",
		"year", "2026",
		"format", "xlsx",
	))
	if err != nil {
		t.Fatalf("expected XLSX format to be valid, got %v", err)
	}
	if options.Format != "xlsx" {
		t.Fatalf("expected xlsx format, got %q", options.Format)
	}
}

func TestGenerateAllExportTypesAsXLSX(t *testing.T) {
	dataset := completeExportTestDataset()
	reportTypes := []string{
		exportTypeExpenses,
		exportTypeIncomes,
		exportTypeCategories,
		exportTypeSummary,
		exportTypeMonthComparison,
		exportTypeInstallmentCommitments,
		exportTypeFullReport,
	}

	for _, reportType := range reportTypes {
		t.Run(reportType, func(t *testing.T) {
			dataset.Options.ReportType = reportType
			dataset.Options.Format = "xlsx"
			content, err := generateExportXLSX(dataset)
			if err != nil {
				t.Fatalf("failed to generate %s XLSX: %v", reportType, err)
			}
			workbook := openXLSXForTest(t, content)
			defer workbook.Close()
			if len(workbook.GetSheetList()) == 0 {
				t.Fatalf("expected %s workbook to contain a sheet", reportType)
			}
		})
	}
}

func TestFullReportXLSXUsesExpectedSheets(t *testing.T) {
	dataset := completeExportTestDataset()
	dataset.Options.ReportType = exportTypeFullReport
	dataset.Options.Format = "xlsx"

	content, err := generateExportXLSX(dataset)
	if err != nil {
		t.Fatalf("failed to generate full XLSX: %v", err)
	}
	workbook := openXLSXForTest(t, content)
	defer workbook.Close()

	expected := []string{"Resumo", "Receitas", "Despesas", "Categorias", "Comparativo", "Parcelamentos", "Insights"}
	if actual := workbook.GetSheetList(); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected sheets: got %#v, want %#v", actual, expected)
	}

	for _, sheet := range expected {
		value, err := workbook.GetCellValue(sheet, "A1")
		if err != nil || value == "" {
			t.Fatalf("sheet %s should have a title, value=%q err=%v", sheet, value, err)
		}
	}
}

func TestExpensesXLSXStoresTypedValuesAndSafeText(t *testing.T) {
	dataset := exportDataset{
		Options: exportOptions{ReportType: exportTypeExpenses, Format: "xlsx", Month: 7, Year: 2026},
		Expenses: []expenses.Expense{{
			Description:   `=HYPERLINK("https://example.com")`,
			Notes:         "Linha 1\nLinha 2",
			CategoryID:    1,
			PaymentSource: "Salario",
			Type:          "Unica",
			Amount:        120.5,
			Date:          time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		}},
		CategoryNames: map[uint]string{1: "Alimentação"},
	}

	content, err := generateExportXLSX(dataset)
	if err != nil {
		t.Fatalf("failed to generate expenses XLSX: %v", err)
	}
	workbook := openXLSXForTest(t, content)
	defer workbook.Close()

	amount, err := workbook.GetCellValue("Despesas", "G6", excelize.Options{RawCellValue: true})
	if err != nil || amount != "120.5" {
		t.Fatalf("expected numeric amount 120.5, got %q err=%v", amount, err)
	}
	formula, err := workbook.GetCellFormula("Despesas", "B6")
	if err != nil || formula != "" {
		t.Fatalf("description must be stored as text, formula=%q err=%v", formula, err)
	}
	description, err := workbook.GetCellValue("Despesas", "B6")
	if err != nil || description != dataset.Expenses[0].Description {
		t.Fatalf("description was not preserved: %q err=%v", description, err)
	}
	styleID, err := workbook.GetCellStyle("Despesas", "G6")
	if err != nil || styleID == 0 {
		t.Fatalf("amount should have an explicit currency style, style=%d err=%v", styleID, err)
	}
}

func TestEmptyExpensesXLSXKeepsHeaders(t *testing.T) {
	dataset := exportDataset{
		Options:       exportOptions{ReportType: exportTypeExpenses, Format: "xlsx", Month: 7, Year: 2026},
		CategoryNames: map[uint]string{},
	}

	content, err := generateExportXLSX(dataset)
	if err != nil {
		t.Fatalf("failed to generate empty expenses XLSX: %v", err)
	}
	workbook := openXLSXForTest(t, content)
	defer workbook.Close()

	header, err := workbook.GetCellValue("Despesas", "A5")
	if err != nil || header != "Data" {
		t.Fatalf("expected header in empty workbook, got %q err=%v", header, err)
	}
	value, err := workbook.GetCellValue("Despesas", "A6")
	if err != nil || value != "" {
		t.Fatalf("expected first data row to be empty, got %q err=%v", value, err)
	}
}

func TestGenerateExportFileReturnsXLSXMetadata(t *testing.T) {
	dataset := completeExportTestDataset()
	dataset.Options.ReportType = exportTypeSummary
	dataset.Options.Format = "xlsx"

	content, contentType, extension, err := generateExportFile(dataset)
	if err != nil {
		t.Fatalf("failed to generate XLSX export file: %v", err)
	}
	if len(content) == 0 || contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" || extension != "xlsx" {
		t.Fatalf("unexpected XLSX metadata: bytes=%d contentType=%q extension=%q", len(content), contentType, extension)
	}
}

func openXLSXForTest(t *testing.T, content []byte) *excelize.File {
	t.Helper()
	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("generated content is not a valid XLSX workbook: %v", err)
	}
	return workbook
}

func mapValues(values ...string) map[string][]string {
	result := make(map[string][]string, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		result[values[index]] = []string{values[index+1]}
	}
	return result
}
