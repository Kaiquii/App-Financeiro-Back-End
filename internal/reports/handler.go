package reports

import (
	"App_Financeiro_Back-end/internal/database"
	"App_Financeiro_Back-end/internal/expenses"
	"App_Financeiro_Back-end/internal/incomes"
	"net/http"
	"strconv"

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

	month, _ := strconv.Atoi(monthStr)
	year, _ := strconv.Atoi(yearStr)

	var incomesList []incomes.Income
	database.DB.Where("user_id = ? AND month = ? AND year = ?", userID, month, year).Find(&incomesList)

	var expensesList []expenses.Expense
	database.DB.Where("user_id = ? AND month = ? AND year = ?", userID, month, year).Find(&expensesList)

	var totalIncome, totalExpense float64
	var totalSalary, totalAdiantamento, totalRendaExtra float64

	var expensesFromSalary, expensesFromAdiantamento, expensesFromRendaExtra float64

	for _, income := range incomesList {
		totalIncome += income.Amount
		if income.Source == "Salário" {
			totalSalary += income.Amount
		} else if income.Source == "Adiantamento" {
			totalAdiantamento += income.Amount
		} else if income.Source == "Renda Extra" {
			totalRendaExtra += income.Amount
		}
	}

	for _, expense := range expensesList {
		totalExpense += expense.Amount
		if expense.PaymentSource == "Salário" {
			expensesFromSalary += expense.Amount
		} else if expense.PaymentSource == "Adiantamento" {
			expensesFromAdiantamento += expense.Amount
		} else if expense.PaymentSource == "Renda Extra" {
			expensesFromRendaExtra += expense.Amount
		}
	}

	restanteSalario := totalSalary - expensesFromSalary
	restanteAdiantamento := totalAdiantamento - expensesFromAdiantamento

	restanteRendaExtra := totalRendaExtra - expensesFromRendaExtra

	totalGeralDisponivel := totalIncome - totalExpense

	c.JSON(http.StatusOK, gin.H{
		"month":                  month,
		"year":                   year,
		"salario":                totalSalary,
		"adiantamento":           totalAdiantamento,
		"renda_extra_amt":        totalRendaExtra,
		"restante_salario":       restanteSalario,
		"restante_adiantamento":  restanteAdiantamento,
		"restante_renda_extra":   restanteRendaExtra,
		"total_expense":          totalExpense,
		"total_geral_disponivel": totalGeralDisponivel,
		"total_income":           totalIncome,
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
