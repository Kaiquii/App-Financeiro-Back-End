package reports

import (
	"Sobra_Ai_Back-end/internal/database"
	"Sobra_Ai_Back-end/internal/expenses"
	"Sobra_Ai_Back-end/internal/incomes"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	reportsGroup := rg.Group("/reports")
	{
		reportsGroup.GET("/summary", getMonthlySummary)
		reportsGroup.GET("/categories", getCategorySummary)
		reportsGroup.GET("/chart", getChartData)
		reportsGroup.GET("/yearly-summary", getYearlySummary)
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
