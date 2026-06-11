package reports

import (
	"Sobra_Ai_Back-end/internal/categories"
	"Sobra_Ai_Back-end/internal/database"
	"Sobra_Ai_Back-end/internal/expenses"
	"Sobra_Ai_Back-end/internal/incomes"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	reportsGroup := rg.Group("/reports")
	{
		reportsGroup.GET("/summary", getMonthlySummary)
		reportsGroup.GET("/categories", getCategorySummary)
		reportsGroup.GET("/chart", getChartData)
		reportsGroup.GET("/yearly-summary", getYearlySummary)
		reportsGroup.GET("/installment-commitments", getInstallmentCommitments)
	}
}

func normalizeMoneySource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "salário", "salario":
		return "salario"
	case "adiantamento":
		return "adiantamento"
	case "renda extra", "renda_extra":
		return "renda_extra"
	default:
		return ""
	}
}

func getMonthlySummary(c *gin.Context) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}
	userID := userIDObj.(uint)

	monthStr := c.Query("month")
	yearStr := c.Query("year")
	if monthStr == "" || yearStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mês e ano são obrigatórios. Ex: ?month=3&year=2026"})
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mês inválido"})
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ano inválido"})
		return
	}

	var incomesList []incomes.Income
	if err := database.DB.Where("user_id = ? AND month = ? AND year = ?", userID, month, year).Find(&incomesList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar rendas"})
		return
	}

	var expensesList []expenses.Expense
	if err := database.DB.Where("user_id = ? AND month = ? AND year = ?", userID, month, year).Find(&expensesList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar despesas"})
		return
	}

	var totalIncome float64
	var totalExpense float64

	var totalSalary float64
	var totalAdiantamento float64
	var totalRendaExtra float64

	var totalSpentSalary float64
	var totalSpentAdiantamento float64
	var totalSpentRendaExtra float64

	for _, income := range incomesList {
		totalIncome += income.Amount

		switch normalizeMoneySource(income.Source) {
		case "salario":
			totalSalary += income.Amount
		case "adiantamento":
			totalAdiantamento += income.Amount
		case "renda_extra":
			totalRendaExtra += income.Amount
		}
	}

	for _, expense := range expensesList {
		totalExpense += expense.Amount

		switch normalizeMoneySource(expense.PaymentSource) {
		case "salario":
			totalSpentSalary += expense.Amount
		case "adiantamento":
			totalSpentAdiantamento += expense.Amount
		case "renda_extra":
			totalSpentRendaExtra += expense.Amount
		}
	}

	restanteSalario := totalSalary - totalSpentSalary
	restanteAdiantamento := totalAdiantamento - totalSpentAdiantamento
	restanteRendaExtra := totalRendaExtra - totalSpentRendaExtra

	totalGeralDisponivel := totalIncome - totalExpense

	c.JSON(http.StatusOK, gin.H{
		"month":                    month,
		"year":                     year,
		"total_income":             totalIncome,
		"total_expense":            totalExpense,
		"total_geral_disponivel":   totalGeralDisponivel,
		"salario":                  totalSalary,
		"adiantamento":             totalAdiantamento,
		"renda_extra_amt":          totalRendaExtra,
		"total_gasto_salario":      totalSpentSalary,
		"total_gasto_adiantamento": totalSpentAdiantamento,
		"total_gasto_renda_extra":  totalSpentRendaExtra,
		"restante_salario":         restanteSalario,
		"restante_adiantamento":    restanteAdiantamento,
		"restante_renda_extra":     restanteRendaExtra,
	})
}

func getCategorySummary(c *gin.Context) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}
	userID := userIDObj.(uint)

	yearStr := c.Query("year")
	monthStr := c.Query("month")

	if yearStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "O ano é obrigatório"})
		return
	}
	year, _ := strconv.Atoi(yearStr)

	var results []CategoryResult

	query := database.DB.Table("expenses").
		Select("expenses.category_id, categories.name as category_name, sum(expenses.amount) as total_amount").
		Joins("left join categories on categories.id = expenses.category_id").
		Where("expenses.user_id = ? AND expenses.year = ?", userID, year)

	if monthStr != "" {
		month, _ := strconv.Atoi(monthStr)
		query = query.Where("expenses.month = ?", month)
	}

	query.Group("expenses.category_id, categories.name").Scan(&results)

	var totalGeral float64
	for _, r := range results {
		totalGeral += r.TotalAmount
	}

	if totalGeral > 0 {
		for i := range results {
			results[i].Percentage = (results[i].TotalAmount / totalGeral) * 100
		}
	}

	c.JSON(http.StatusOK, results)
}

func getChartData(c *gin.Context) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}
	userID := userIDObj.(uint)

	yearStr := c.Query("year")
	if yearStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "O ano é obrigatório. Ex: ?year=2026"})
		return
	}
	year, _ := strconv.Atoi(yearStr)

	var incomesList []incomes.Income
	database.DB.Where("user_id = ? AND year = ?", userID, year).Find(&incomesList)

	var expensesList []expenses.Expense
	database.DB.Where("user_id = ? AND year = ?", userID, year).Find(&expensesList)

	monthlyData := make(map[int]ChartResult)

	for _, inc := range incomesList {
		data := monthlyData[inc.Month]
		data.Month = inc.Month
		data.Income += inc.Amount
		monthlyData[inc.Month] = data
	}

	for _, exp := range expensesList {
		data := monthlyData[exp.Month]
		data.Month = exp.Month
		data.Expense += exp.Amount
		monthlyData[exp.Month] = data
	}

	var results []ChartResult
	for i := 1; i <= 12; i++ {
		if data, ok := monthlyData[i]; ok {
			results = append(results, data)
		} else {
			results = append(results, ChartResult{Month: i, Income: 0, Expense: 0})
		}
	}

	c.JSON(http.StatusOK, results)
}

func getYearlySummary(c *gin.Context) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}
	userID := userIDObj.(uint)

	yearStr := c.Query("year")
	if yearStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "O ano é obrigatório. Ex: ?year=2026"})
		return
	}
	year, _ := strconv.Atoi(yearStr)

	var totalIncome float64
	database.DB.Table("incomes").
		Where("user_id = ? AND year = ?", userID, year).
		Select("COALESCE(sum(amount), 0)").Scan(&totalIncome)

	var totalExpense float64
	database.DB.Table("expenses").
		Where("user_id = ? AND year = ?", userID, year).
		Select("COALESCE(sum(amount), 0)").Scan(&totalExpense)

	economiaTotal := totalIncome - totalExpense
	mediaMensal := totalExpense / 12

	c.JSON(http.StatusOK, gin.H{
		"year":           year,
		"economia_total": economiaTotal,
		"media_mensal":   mediaMensal,
	})
}

func getInstallmentCommitments(c *gin.Context) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario nao identificado"})
		return
	}
	userID := userIDObj.(uint)

	now := time.Now()
	baseMonth := int(now.Month())
	baseYear := now.Year()

	if monthStr := c.Query("month"); monthStr != "" {
		month, err := strconv.Atoi(monthStr)
		if err != nil || month < 1 || month > 12 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mes invalido"})
			return
		}
		baseMonth = month
	}

	if yearStr := c.Query("year"); yearStr != "" {
		year, err := strconv.Atoi(yearStr)
		if err != nil || year < 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ano invalido"})
			return
		}
		baseYear = year
	}

	months := 12
	if monthsStr := c.Query("months"); monthsStr != "" {
		parsedMonths, err := strconv.Atoi(monthsStr)
		if err != nil || parsedMonths < 1 || parsedMonths > 60 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Months deve ser um numero entre 1 e 60"})
			return
		}
		months = parsedMonths
	}

	includeCurrentMonthAsPaid := c.Query("include_current_month_as_paid") == "true"

	var installments []expenses.Expense
	if err := database.DB.
		Where("user_id = ? AND type = ?", userID, "Parcelada").
		Order("year asc, month asc, current_install asc, id asc").
		Find(&installments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar parcelas"})
		return
	}

	categoryNames, err := loadCategoryNames(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar categorias"})
		return
	}

	response := buildInstallmentCommitmentsResponse(installments, categoryNames, baseMonth, baseYear, months, includeCurrentMonthAsPaid)
	c.JSON(http.StatusOK, response)
}

func loadCategoryNames(userID uint) (map[uint]string, error) {
	var categoriesList []categories.Category
	if err := database.DB.Where("user_id = ?", userID).Find(&categoriesList).Error; err != nil {
		return nil, err
	}

	categoryNames := make(map[uint]string, len(categoriesList))
	for _, category := range categoriesList {
		categoryNames[category.ID] = category.Name
	}

	return categoryNames, nil
}

func buildInstallmentCommitmentsResponse(installments []expenses.Expense, categoryNames map[uint]string, baseMonth int, baseYear int, months int, includeCurrentMonthAsPaid bool) InstallmentCommitmentsResponse {
	baseDate := monthDate(baseYear, baseMonth)
	openStartDate := baseDate
	if includeCurrentMonthAsPaid {
		openStartDate = baseDate.AddDate(0, 1, 0)
	}
	lastVisibleDate := openStartDate.AddDate(0, months-1, 0)
	purchasesByKey := make(map[string][]expenses.Expense)
	monthTotals := make(map[string]*InstallmentMonthSummary)

	for _, installment := range installments {
		key := installment.SeriesID
		if key == "" {
			key = legacyInstallmentKey(installment)
		}
		purchasesByKey[key] = append(purchasesByKey[key], installment)

		installmentDate := monthDate(installment.Year, installment.Month)
		if installmentDate.Before(openStartDate) || installmentDate.After(lastVisibleDate) {
			continue
		}

		monthKey := monthMapKey(installment.Year, installment.Month)
		monthSummary := monthTotals[monthKey]
		if monthSummary == nil {
			monthSummary = &InstallmentMonthSummary{
				Month:        installment.Month,
				Year:         installment.Year,
				Installments: []InstallmentItemSummary{},
			}
			monthTotals[monthKey] = monthSummary
		}

		monthSummary.Total += installment.Amount
		monthSummary.Installments = append(monthSummary.Installments, installmentItemSummary(installment, categoryNames))
	}

	purchases := make([]InstallmentPurchaseSummary, 0, len(purchasesByKey))
	summary := InstallmentCommitmentsSummary{}

	for seriesKey, purchaseInstallments := range purchasesByKey {
		sortInstallments(purchaseInstallments)
		purchase := buildPurchaseSummary(seriesKey, purchaseInstallments, categoryNames, openStartDate)

		summary.OriginalTotal += purchase.OriginalTotal
		summary.PaidTotal += purchase.PaidTotal
		summary.RemainingTotal += purchase.RemainingTotal
		summary.PaidInstallments += purchase.PaidInstallments
		summary.RemainingInstallments += purchase.RemainingInstallments

		purchases = append(purchases, purchase)
	}

	sort.Slice(purchases, func(i, j int) bool {
		if purchases[i].RemainingTotal != purchases[j].RemainingTotal {
			return purchases[i].RemainingTotal > purchases[j].RemainingTotal
		}
		return purchases[i].Description < purchases[j].Description
	})

	timeline := make([]InstallmentMonthSummary, 0, months)
	for i := 0; i < months; i++ {
		currentDate := openStartDate.AddDate(0, i, 0)
		month := int(currentDate.Month())
		year := currentDate.Year()
		monthSummary := monthTotals[monthMapKey(year, month)]
		if monthSummary == nil {
			monthSummary = &InstallmentMonthSummary{
				Month:        month,
				Year:         year,
				Installments: []InstallmentItemSummary{},
			}
		}

		sort.Slice(monthSummary.Installments, func(i, j int) bool {
			if monthSummary.Installments[i].Amount != monthSummary.Installments[j].Amount {
				return monthSummary.Installments[i].Amount > monthSummary.Installments[j].Amount
			}
			return monthSummary.Installments[i].Description < monthSummary.Installments[j].Description
		})

		monthSummary.Total = roundMoney(monthSummary.Total)
		timeline = append(timeline, *monthSummary)
	}

	var heaviestMonth *InstallmentHeaviestMonth
	for _, monthSummary := range timeline {
		if monthSummary.Total <= 0 {
			continue
		}
		if heaviestMonth == nil || monthSummary.Total > heaviestMonth.Total {
			heaviestMonth = &InstallmentHeaviestMonth{
				Month: monthSummary.Month,
				Year:  monthSummary.Year,
				Total: monthSummary.Total,
			}
		}
	}

	summary.OriginalTotal = roundMoney(summary.OriginalTotal)
	summary.PaidTotal = roundMoney(summary.PaidTotal)
	summary.RemainingTotal = roundMoney(summary.RemainingTotal)
	summary.TotalPurchases = len(purchases)
	summary.HeaviestMonth = heaviestMonth

	return InstallmentCommitmentsResponse{
		BaseMonth: baseMonth,
		BaseYear:  baseYear,
		Months:    months,
		Summary:   summary,
		Purchases: purchases,
		Timeline:  timeline,
	}
}

func buildPurchaseSummary(seriesKey string, installments []expenses.Expense, categoryNames map[uint]string, baseDate time.Time) InstallmentPurchaseSummary {
	first := installments[0]
	last := installments[len(installments)-1]
	totalInstallments := first.Installments
	if totalInstallments <= 0 {
		totalInstallments = len(installments)
	}

	installmentAmount := first.Amount
	originalTotal := installmentAmount * float64(totalInstallments)
	var paidTotal float64
	var remainingTotal float64
	var paidInstallments int
	var remainingInstallments int
	var nextInstallment *InstallmentItemSummary

	for _, installment := range installments {
		installmentDate := monthDate(installment.Year, installment.Month)
		if installmentDate.Before(baseDate) {
			paidTotal += installment.Amount
			paidInstallments++
			continue
		}

		remainingTotal += installment.Amount
		remainingInstallments++
		if nextInstallment == nil {
			item := installmentItemSummary(installment, categoryNames)
			nextInstallment = &item
		}
	}

	return InstallmentPurchaseSummary{
		SeriesID:              seriesKey,
		Description:           first.Description,
		CategoryID:            first.CategoryID,
		CategoryName:          categoryNames[first.CategoryID],
		PaymentSource:         first.PaymentSource,
		InstallmentAmount:     roundMoney(installmentAmount),
		OriginalTotal:         roundMoney(originalTotal),
		PaidTotal:             roundMoney(paidTotal),
		RemainingTotal:        roundMoney(remainingTotal),
		PaidInstallments:      paidInstallments,
		RemainingInstallments: remainingInstallments,
		TotalInstallments:     totalInstallments,
		FirstMonth:            first.Month,
		FirstYear:             first.Year,
		LastMonth:             last.Month,
		LastYear:              last.Year,
		NextInstallment:       nextInstallment,
	}
}

func installmentItemSummary(installment expenses.Expense, categoryNames map[uint]string) InstallmentItemSummary {
	return InstallmentItemSummary{
		ID:                 installment.ID,
		SeriesID:           installment.SeriesID,
		Description:        installment.Description,
		CategoryID:         installment.CategoryID,
		CategoryName:       categoryNames[installment.CategoryID],
		PaymentSource:      installment.PaymentSource,
		Amount:             roundMoney(installment.Amount),
		Month:              installment.Month,
		Year:               installment.Year,
		CurrentInstallment: installment.CurrentInstall,
		TotalInstallments:  installment.Installments,
	}
}

func sortInstallments(installments []expenses.Expense) {
	sort.Slice(installments, func(i, j int) bool {
		left := monthDate(installments[i].Year, installments[i].Month)
		right := monthDate(installments[j].Year, installments[j].Month)
		if !left.Equal(right) {
			return left.Before(right)
		}
		return installments[i].CurrentInstall < installments[j].CurrentInstall
	})
}

func legacyInstallmentKey(expense expenses.Expense) string {
	return strings.Join([]string{
		expense.Description,
		strconv.FormatUint(uint64(expense.CategoryID), 10),
		expense.PaymentSource,
		strconv.FormatFloat(expense.Amount, 'f', 2, 64),
		strconv.Itoa(expense.Installments),
	}, "|")
}

func monthDate(year int, month int) time.Time {
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
}

func monthMapKey(year int, month int) string {
	return strconv.Itoa(year) + "-" + strconv.Itoa(month)
}

func roundMoney(value float64) float64 {
	rounded, _ := strconv.ParseFloat(strconv.FormatFloat(value, 'f', 2, 64), 64)
	return rounded
}
