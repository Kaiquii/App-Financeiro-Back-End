package reports

import (
	"bytes"
	"testing"
)

func TestParseExportOptionsAcceptsPDF(t *testing.T) {
	options, err := parseExportOptions(mapValues(
		"type", exportTypeFullReport,
		"month", "7",
		"year", "2026",
		"format", "pdf",
	))
	if err != nil {
		t.Fatalf("expected PDF format to be valid, got %v", err)
	}
	if options.Format != "pdf" {
		t.Fatalf("expected pdf format, got %q", options.Format)
	}
}

func TestGenerateAllExportTypesAsPDF(t *testing.T) {
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
			dataset.Options.Format = "pdf"
			content, err := generateExportPDF(dataset)
			if err != nil {
				t.Fatalf("failed to generate %s PDF: %v", reportType, err)
			}
			assertValidPDFBytes(t, content)
		})
	}
}

func TestFullReportPDFUsesMultiplePages(t *testing.T) {
	dataset := completeExportTestDataset()
	dataset.Options.ReportType = exportTypeFullReport
	dataset.Options.Format = "pdf"

	content, err := generateExportPDF(dataset)
	if err != nil {
		t.Fatalf("failed to generate full PDF: %v", err)
	}
	assertValidPDFBytes(t, content)

	pageObjects := bytes.Count(content, []byte("/Type /Page")) - bytes.Count(content, []byte("/Type /Pages"))
	if pageObjects < 7 {
		t.Fatalf("expected full report to use at least 7 pages, got %d", pageObjects)
	}
}

func TestEmptyExpensesPDFRemainsValid(t *testing.T) {
	dataset := exportDataset{
		Options:       exportOptions{ReportType: exportTypeExpenses, Format: "pdf", Month: 7, Year: 2026},
		CategoryNames: map[uint]string{},
	}

	content, err := generateExportPDF(dataset)
	if err != nil {
		t.Fatalf("failed to generate empty expenses PDF: %v", err)
	}
	assertValidPDFBytes(t, content)
}

func TestGenerateExportFileReturnsPDFMetadata(t *testing.T) {
	dataset := completeExportTestDataset()
	dataset.Options.ReportType = exportTypeSummary
	dataset.Options.Format = "pdf"

	content, contentType, extension, err := generateExportFile(dataset)
	if err != nil {
		t.Fatalf("failed to generate PDF export file: %v", err)
	}
	if contentType != "application/pdf" || extension != "pdf" {
		t.Fatalf("unexpected PDF metadata: contentType=%q extension=%q", contentType, extension)
	}
	assertValidPDFBytes(t, content)
}

func TestFormatPDFCurrencyUsesBrazilianSeparators(t *testing.T) {
	tests := map[float64]string{
		3000:     "R$ 3.000,00",
		1200.5:   "R$ 1.200,50",
		-9876.54: "-R$ 9.876,54",
	}
	for value, expected := range tests {
		if actual := formatPDFCurrency(value); actual != expected {
			t.Fatalf("formatPDFCurrency(%v) = %q, want %q", value, actual, expected)
		}
	}
}

func assertValidPDFBytes(t *testing.T, content []byte) {
	t.Helper()
	if len(content) < 8 || !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatalf("generated content is not a PDF: bytes=%d", len(content))
	}
	if !bytes.Contains(content, []byte("%%EOF")) {
		t.Fatal("generated PDF does not contain an EOF marker")
	}
}
