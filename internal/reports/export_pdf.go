package reports

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	pdfFontFamily    = "SobraAiSans"
	pdfMargin        = 12.0
	pdfContentTop    = 42.0
	pdfContentBottom = 15.0
)

type pdfReportRenderer struct {
	document    *fpdf.Fpdf
	reportTitle string
}

type pdfTableContext struct {
	sheetTitle   string
	subtitle     string
	sectionTitle string
	orientation  string
	widths       []float64
	fontSize     float64
}

func generateExportPDF(dataset exportDataset) ([]byte, error) {
	sheets, err := buildXLSXSheets(dataset)
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, fmt.Errorf("missing PDF report sections")
	}

	renderer := newPDFReportRenderer(sheets[0].Title)
	for _, sheet := range sheets {
		if err := renderer.renderSheet(sheet); err != nil {
			return nil, err
		}
	}

	var output bytes.Buffer
	if err := renderer.document.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func newPDFReportRenderer(reportTitle string) *pdfReportRenderer {
	document := fpdf.New("P", "mm", "A4", "")
	document.AddUTF8FontFromBytes(pdfFontFamily, "", goregular.TTF)
	document.AddUTF8FontFromBytes(pdfFontFamily, "B", gobold.TTF)
	document.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	document.SetAutoPageBreak(false, pdfContentBottom)
	document.SetTitle(reportTitle, true)
	document.SetSubject("Relatório financeiro do SobraAi", true)
	document.SetAuthor("SobraAi", true)
	document.SetCreator("SobraAi", true)
	document.AliasNbPages("{nb}")

	renderer := &pdfReportRenderer{document: document, reportTitle: reportTitle}
	document.SetFooterFunc(func() {
		pageWidth, pageHeight := document.GetPageSize()
		document.SetDrawColor(215, 222, 232)
		document.Line(pdfMargin, pageHeight-10, pageWidth-pdfMargin, pageHeight-10)
		document.SetY(pageHeight - 8)
		document.SetFont(pdfFontFamily, "", 7)
		document.SetTextColor(91, 107, 122)
		document.CellFormat((pageWidth-2*pdfMargin)/2, 4, "SobraAi - Relatório financeiro", "", 0, "L", false, 0, "")
		document.CellFormat((pageWidth-2*pdfMargin)/2, 4, fmt.Sprintf("Página %d de {nb}", document.PageNo()), "", 0, "R", false, 0, "")
	})
	return renderer
}

func (renderer *pdfReportRenderer) renderSheet(sheet xlsxSheet) error {
	orientation := pdfOrientationForSheet(sheet)
	renderer.addPage(orientation, sheet.Title, sheet.Subtitle)

	if sheet.Name == "Resumo" && len(sheet.Sections) > 0 {
		renderer.renderSummary(sheet, sheet.Sections[0], orientation)
		return renderer.document.Error()
	}

	for index, section := range sheet.Sections {
		if index > 0 {
			renderer.ensureSpace(orientation, sheet.Title, sheet.Subtitle, 16)
		}
		switch {
		case section.Title == "Insights":
			renderer.renderInsights(sheet, section, orientation)
		case sheet.Name == "Parcelamentos" && section.Title == "Compras":
			renderer.renderPurchaseCards(sheet, section, orientation)
		default:
			renderer.renderTableSection(sheet, section, orientation)
		}
	}
	return renderer.document.Error()
}

func (renderer *pdfReportRenderer) addPage(orientation, title, subtitle string) {
	renderer.document.AddPageFormat(orientation, fpdf.SizeType{Wd: 210, Ht: 297})
	pageWidth, _ := renderer.document.GetPageSize()

	renderer.document.SetFillColor(23, 50, 77)
	renderer.document.Rect(0, 0, pageWidth, 18, "F")
	renderer.document.SetTextColor(255, 255, 255)
	renderer.document.SetFont(pdfFontFamily, "B", 13)
	renderer.document.SetXY(pdfMargin, 5)
	renderer.document.CellFormat(45, 8, "SobraAi", "", 0, "L", false, 0, "")
	renderer.document.SetFont(pdfFontFamily, "", 8)
	renderer.document.SetXY(pageWidth-92, 6)
	renderer.document.CellFormat(80, 7, "RELATÓRIO FINANCEIRO", "", 0, "R", false, 0, "")

	renderer.document.SetTextColor(23, 50, 77)
	renderer.document.SetFont(pdfFontFamily, "B", 17)
	renderer.document.SetXY(pdfMargin, 24)
	renderer.document.CellFormat(pageWidth-2*pdfMargin, 8, title, "", 1, "L", false, 0, "")
	renderer.document.SetTextColor(82, 96, 109)
	renderer.document.SetFont(pdfFontFamily, "", 8)
	renderer.document.SetXY(pdfMargin, 33)
	renderer.document.CellFormat(pageWidth-2*pdfMargin, 5, subtitle, "", 1, "L", false, 0, "")
	renderer.document.SetDrawColor(215, 222, 232)
	renderer.document.Line(pdfMargin, 40, pageWidth-pdfMargin, 40)
	renderer.document.SetY(pdfContentTop)
}

func (renderer *pdfReportRenderer) renderSummary(sheet xlsxSheet, section xlsxSection, orientation string) {
	renderer.sectionHeading(section.Title)
	rows := section.Rows
	if len(rows) >= 3 {
		renderer.renderSummaryCards(rows[:3])
		rows = rows[3:]
	}
	if len(rows) > 0 {
		detail := section
		detail.Title = "Detalhamento"
		detail.Rows = rows
		renderer.renderTableSection(sheet, detail, orientation)
	}
}

func (renderer *pdfReportRenderer) renderSummaryCards(rows [][]xlsxCell) {
	pageWidth, _ := renderer.document.GetPageSize()
	gap := 4.0
	cardWidth := (pageWidth - 2*pdfMargin - 2*gap) / 3
	y := renderer.document.GetY() + 2
	cardColors := [][3]int{{232, 246, 239}, {253, 238, 238}, {232, 243, 241}}
	accentColors := [][3]int{{36, 139, 84}, {190, 64, 64}, {22, 125, 127}}

	for index, row := range rows {
		x := pdfMargin + float64(index)*(cardWidth+gap)
		fill := cardColors[index%len(cardColors)]
		accent := accentColors[index%len(accentColors)]
		if index == 2 && numericPDFCellValue(row[1]) < 0 {
			fill = cardColors[1]
			accent = accentColors[1]
		}
		renderer.document.SetFillColor(fill[0], fill[1], fill[2])
		renderer.document.SetDrawColor(215, 222, 232)
		renderer.document.Rect(x, y, cardWidth, 25, "DF")
		renderer.document.SetFillColor(accent[0], accent[1], accent[2])
		renderer.document.Rect(x, y, 2, 25, "F")
		renderer.document.SetTextColor(82, 96, 109)
		renderer.document.SetFont(pdfFontFamily, "", 8)
		renderer.document.SetXY(x+5, y+4)
		renderer.document.CellFormat(cardWidth-8, 5, pdfCellText(row[0]), "", 1, "L", false, 0, "")
		renderer.document.SetTextColor(accent[0], accent[1], accent[2])
		renderer.document.SetFont(pdfFontFamily, "B", 13)
		renderer.document.SetXY(x+5, y+11)
		renderer.document.CellFormat(cardWidth-8, 8, pdfCellText(row[1]), "", 0, "L", false, 0, "")
	}
	renderer.document.SetY(y + 31)
}

func (renderer *pdfReportRenderer) renderTableSection(sheet xlsxSheet, section xlsxSection, orientation string) {
	widths := pdfColumnWidths(section, renderer.pageContentWidth())
	context := pdfTableContext{
		sheetTitle: sheet.Title, subtitle: sheet.Subtitle, sectionTitle: section.Title,
		orientation: orientation, widths: widths, fontSize: pdfTableFontSize(len(section.Headers)),
	}
	renderer.startTable(context, section.Headers, false)

	if len(section.Rows) == 0 {
		renderer.emptyState("Nenhum dado encontrado para o período selecionado.")
		return
	}

	for index, row := range section.Rows {
		rowHeight := renderer.pdfTableRowHeight(row, widths, context.fontSize)
		if renderer.needsPage(rowHeight) {
			renderer.addPage(orientation, sheet.Title, sheet.Subtitle)
			renderer.startTable(context, section.Headers, true)
		}
		renderer.drawTableRow(row, widths, rowHeight, context.fontSize, index%2 == 1)
	}
	renderer.document.SetY(renderer.document.GetY() + 6)
}

func (renderer *pdfReportRenderer) startTable(context pdfTableContext, headers []string, continued bool) {
	title := context.sectionTitle
	if continued {
		title += " - continuação"
	}
	renderer.sectionHeading(title)
	if len(headers) == 0 {
		return
	}

	headerCells := make([]xlsxCell, len(headers))
	for index, header := range headers {
		headerCells[index] = xlsxCell{Value: header, Kind: xlsxText}
	}
	height := renderer.pdfTableRowHeight(headerCells, context.widths, context.fontSize)
	if height < 8 {
		height = 8
	}
	y := renderer.document.GetY()
	x := pdfMargin
	renderer.document.SetFont(pdfFontFamily, "B", context.fontSize)
	renderer.document.SetTextColor(255, 255, 255)
	for index, header := range headers {
		width := context.widths[index]
		renderer.document.SetFillColor(22, 125, 127)
		renderer.document.SetDrawColor(255, 255, 255)
		renderer.document.Rect(x, y, width, height, "DF")
		renderer.drawWrappedText(x+1.5, y+1.2, width-3, height-2, header, "L", context.fontSize)
		x += width
	}
	renderer.document.SetY(y + height)
}

func (renderer *pdfReportRenderer) drawTableRow(row []xlsxCell, widths []float64, height, fontSize float64, alternate bool) {
	y := renderer.document.GetY()
	x := pdfMargin
	renderer.document.SetFont(pdfFontFamily, "", fontSize)
	renderer.document.SetTextColor(35, 49, 61)
	for index, width := range widths {
		cell := xlsxCell{Value: "", Kind: xlsxText}
		if index < len(row) {
			cell = row[index]
		}
		if alternate {
			renderer.document.SetFillColor(247, 249, 251)
		} else {
			renderer.document.SetFillColor(255, 255, 255)
		}
		renderer.document.SetDrawColor(222, 227, 233)
		renderer.document.Rect(x, y, width, height, "DF")
		renderer.drawWrappedText(x+1.5, y+1.2, width-3, height-2, pdfCellText(cell), pdfCellAlignment(cell), fontSize)
		x += width
	}
	renderer.document.SetY(y + height)
}

func (renderer *pdfReportRenderer) renderPurchaseCards(sheet xlsxSheet, section xlsxSection, orientation string) {
	renderer.sectionHeading(section.Title)
	if len(section.Rows) == 0 {
		renderer.emptyState("Nenhuma compra parcelada encontrada para o período.")
		return
	}

	for _, row := range section.Rows {
		const cardHeight = 35.0
		if renderer.needsPage(cardHeight + 4) {
			renderer.addPage(orientation, sheet.Title, sheet.Subtitle)
			renderer.sectionHeading(section.Title + " - continuação")
		}
		renderer.drawPurchaseCard(row, cardHeight)
	}
	renderer.document.SetY(renderer.document.GetY() + 4)
}

func (renderer *pdfReportRenderer) drawPurchaseCard(row []xlsxCell, height float64) {
	pageWidth, _ := renderer.document.GetPageSize()
	width := pageWidth - 2*pdfMargin
	x := pdfMargin
	y := renderer.document.GetY()
	renderer.document.SetFillColor(250, 251, 252)
	renderer.document.SetDrawColor(215, 222, 232)
	renderer.document.Rect(x, y, width, height, "DF")
	renderer.document.SetFillColor(22, 125, 127)
	renderer.document.Rect(x, y, 2, height, "F")

	renderer.document.SetTextColor(23, 50, 77)
	renderer.document.SetFont(pdfFontFamily, "B", 10)
	renderer.document.SetXY(x+5, y+3)
	renderer.document.CellFormat(width*0.58, 5, pdfRowText(row, 0), "", 0, "L", false, 0, "")
	renderer.document.SetTextColor(82, 96, 109)
	renderer.document.SetFont(pdfFontFamily, "", 7.5)
	renderer.document.SetXY(x+5, y+9)
	renderer.document.CellFormat(width-10, 4, fmt.Sprintf("%s | %s", pdfRowText(row, 1), pdfRowText(row, 2)), "", 0, "L", false, 0, "")

	labels := []string{"Parcela", "Total original", "Total pago", "Total restante"}
	indexes := []int{3, 4, 5, 6}
	metricY := y + 16
	metricWidth := (width - 10) / 4
	for index, label := range labels {
		metricX := x + 5 + float64(index)*metricWidth
		renderer.document.SetTextColor(91, 107, 122)
		renderer.document.SetFont(pdfFontFamily, "", 6.8)
		renderer.document.SetXY(metricX, metricY)
		renderer.document.CellFormat(metricWidth-2, 4, label, "", 1, "L", false, 0, "")
		renderer.document.SetTextColor(35, 49, 61)
		renderer.document.SetFont(pdfFontFamily, "B", 8.5)
		renderer.document.SetXY(metricX, metricY+4)
		renderer.document.CellFormat(metricWidth-2, 5, pdfRowText(row, indexes[index]), "", 0, "L", false, 0, "")
	}

	renderer.document.SetTextColor(82, 96, 109)
	renderer.document.SetFont(pdfFontFamily, "", 7)
	renderer.document.SetXY(x+5, y+28)
	parcelText := fmt.Sprintf("Parcelas: %s pagas | %s restantes | %s no total", pdfRowText(row, 7), pdfRowText(row, 8), pdfRowText(row, 9))
	renderer.document.CellFormat(width*0.55, 4, parcelText, "", 0, "L", false, 0, "")
	renderer.document.SetXY(x+width*0.56, y+28)
	renderer.document.CellFormat(width*0.41, 4, "Próxima: "+emptyDash(pdfRowText(row, 10)), "", 0, "R", false, 0, "")
	renderer.document.SetY(y + height + 3)
}

func (renderer *pdfReportRenderer) renderInsights(sheet xlsxSheet, section xlsxSection, orientation string) {
	renderer.sectionHeading(section.Title)
	if len(section.Rows) == 0 {
		renderer.emptyState("Nenhum insight disponível para o período selecionado.")
		return
	}

	pageWidth, _ := renderer.document.GetPageSize()
	width := pageWidth - 2*pdfMargin
	for index, row := range section.Rows {
		text := pdfRowText(row, 0)
		renderer.document.SetFont(pdfFontFamily, "", 9)
		lines := renderer.document.SplitText(text, width-18)
		height := maxFloat(13, float64(len(lines))*4.5+5)
		if renderer.needsPage(height + 3) {
			renderer.addPage(orientation, sheet.Title, sheet.Subtitle)
			renderer.sectionHeading(section.Title + " - continuação")
		}
		x := pdfMargin
		y := renderer.document.GetY()
		renderer.document.SetFillColor(242, 247, 247)
		renderer.document.SetDrawColor(215, 226, 225)
		renderer.document.Rect(x, y, width, height, "DF")
		renderer.document.SetFillColor(22, 125, 127)
		renderer.document.Rect(x+3, y+3, 8, 8, "F")
		renderer.document.SetTextColor(255, 255, 255)
		renderer.document.SetFont(pdfFontFamily, "B", 8)
		renderer.document.SetXY(x+3, y+4.5)
		renderer.document.CellFormat(8, 5, strconv.Itoa(index+1), "", 0, "C", false, 0, "")
		renderer.document.SetTextColor(35, 49, 61)
		renderer.document.SetFont(pdfFontFamily, "", 9)
		renderer.drawWrappedText(x+14, y+2.5, width-17, height-4, text, "L", 9)
		renderer.document.SetY(y + height + 3)
	}
}

func (renderer *pdfReportRenderer) sectionHeading(title string) {
	renderer.ensureSpace("", "", "", 10)
	pageWidth, _ := renderer.document.GetPageSize()
	y := renderer.document.GetY()
	renderer.document.SetFillColor(232, 243, 241)
	renderer.document.SetTextColor(23, 50, 77)
	renderer.document.SetFont(pdfFontFamily, "B", 10)
	renderer.document.Rect(pdfMargin, y, pageWidth-2*pdfMargin, 8, "F")
	renderer.document.SetXY(pdfMargin+3, y+1.4)
	renderer.document.CellFormat(pageWidth-2*pdfMargin-6, 5, title, "", 0, "L", false, 0, "")
	renderer.document.SetY(y + 10)
}

func (renderer *pdfReportRenderer) emptyState(message string) {
	pageWidth, _ := renderer.document.GetPageSize()
	y := renderer.document.GetY() + 2
	renderer.document.SetFillColor(247, 249, 251)
	renderer.document.SetDrawColor(222, 227, 233)
	renderer.document.Rect(pdfMargin, y, pageWidth-2*pdfMargin, 15, "DF")
	renderer.document.SetTextColor(91, 107, 122)
	renderer.document.SetFont(pdfFontFamily, "", 9)
	renderer.document.SetXY(pdfMargin+4, y+5)
	renderer.document.CellFormat(pageWidth-2*pdfMargin-8, 5, message, "", 0, "C", false, 0, "")
	renderer.document.SetY(y + 20)
}

func (renderer *pdfReportRenderer) drawWrappedText(x, y, width, height float64, text, alignment string, fontSize float64) {
	renderer.document.SetXY(x, y)
	lineHeight := maxFloat(3.2, fontSize*0.46)
	renderer.document.MultiCell(width, lineHeight, text, "", alignment, false)
}

func (renderer *pdfReportRenderer) pdfTableRowHeight(row []xlsxCell, widths []float64, fontSize float64) float64 {
	renderer.document.SetFont(pdfFontFamily, "", fontSize)
	lineHeight := maxFloat(3.2, fontSize*0.46)
	maxLines := 1
	for index, cell := range row {
		if index >= len(widths) {
			break
		}
		lines := renderer.document.SplitText(pdfCellText(cell), maxFloat(2, widths[index]-3))
		maxLines = maxInt(maxLines, len(lines))
	}
	return maxFloat(6.5, float64(maxLines)*lineHeight+2.4)
}

func (renderer *pdfReportRenderer) ensureSpace(orientation, title, subtitle string, required float64) {
	if !renderer.needsPage(required) {
		return
	}
	if orientation == "" {
		orientation = "P"
	}
	if title == "" {
		title = renderer.reportTitle
	}
	renderer.addPage(orientation, title, subtitle)
}

func (renderer *pdfReportRenderer) needsPage(required float64) bool {
	_, pageHeight := renderer.document.GetPageSize()
	return renderer.document.GetY()+required > pageHeight-pdfContentBottom
}

func (renderer *pdfReportRenderer) pageContentWidth() float64 {
	pageWidth, _ := renderer.document.GetPageSize()
	return pageWidth - 2*pdfMargin
}

func pdfOrientationForSheet(sheet xlsxSheet) string {
	switch sheet.Name {
	case "Despesas", "Comparativo", "Parcelamentos":
		return "L"
	default:
		return "P"
	}
}

func pdfColumnWidths(section xlsxSection, available float64) []float64 {
	weights := make([]float64, len(section.Headers))
	for index, header := range section.Headers {
		weights[index] = maxFloat(1, float64(len([]rune(header)))/8)
	}

	switch section.Title {
	case "Despesas":
		weights = []float64{1.2, 2.4, 1.7, 1.8, 1.1, 1.1, 1.3, 3.0}
	case "Receitas":
		weights = []float64{1, 1, 3.2, 1.8}
	case "Categorias":
		if len(section.Headers) == 3 {
			weights = []float64{3.2, 1.7, 1.4}
		} else {
			weights = []float64{2.2, 1.5, 1.5, 1.5, 1.2, 1.2}
		}
	case "Resumo Mensal", "Detalhamento", "Resumo":
		if len(section.Headers) == 2 {
			weights = []float64{3, 2}
		} else if len(section.Headers) == 6 {
			weights = []float64{2.2, 1.5, 1.5, 1.5, 1.2, 1.2}
		}
	case "Fontes de Pagamento", "Tipos de Despesa":
		weights = []float64{2.2, 1.5, 1.5, 1.5, 1.2, 1.2}
	case "Linha do Tempo":
		weights = []float64{0.8, 1, 2.5, 1.8, 1.6, 1.2, 1.3}
	}

	if len(weights) != len(section.Headers) {
		weights = make([]float64, len(section.Headers))
		for index := range weights {
			weights[index] = 1
		}
	}
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	widths := make([]float64, len(weights))
	for index, weight := range weights {
		widths[index] = available * weight / total
	}
	return widths
}

func pdfTableFontSize(columns int) float64 {
	switch {
	case columns <= 3:
		return 9
	case columns <= 5:
		return 8.2
	case columns <= 7:
		return 7.3
	default:
		return 6.8
	}
}

func pdfCellText(cell xlsxCell) string {
	switch cell.Kind {
	case xlsxDate:
		if value, ok := cell.Value.(time.Time); ok {
			return value.Format("02/01/2006")
		}
	case xlsxCurrency:
		return formatPDFCurrency(numericPDFCellValue(cell))
	case xlsxPercentage:
		return formatDecimal(numericPDFCellValue(cell)*100) + "%"
	case xlsxInteger:
		switch value := cell.Value.(type) {
		case int:
			return strconv.Itoa(value)
		case int64:
			return strconv.FormatInt(value, 10)
		case float64:
			return strconv.FormatInt(int64(value), 10)
		}
	}
	return fmt.Sprint(cell.Value)
}

func pdfCellAlignment(cell xlsxCell) string {
	switch cell.Kind {
	case xlsxCurrency, xlsxPercentage, xlsxInteger:
		return "R"
	case xlsxDate:
		return "C"
	default:
		return "L"
	}
}

func numericPDFCellValue(cell xlsxCell) float64 {
	switch value := cell.Value.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}

func formatPDFCurrency(value float64) string {
	if value < 0 {
		return "-R$ " + formatPDFNumber(-value)
	}
	return "R$ " + formatPDFNumber(value)
}

func formatPDFNumber(value float64) string {
	parts := strings.Split(strconv.FormatFloat(roundMoney(value), 'f', 2, 64), ".")
	integer := parts[0]
	for index := len(integer) - 3; index > 0; index -= 3 {
		integer = integer[:index] + "." + integer[index:]
	}
	return integer + "," + parts[1]
}

func pdfRowText(row []xlsxCell, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return pdfCellText(row[index])
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
