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
	var expensesFromSalary, expensesFromAdiantamento float64

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
		}
	}

	restanteSalario := totalSalary - expensesFromSalary
	restanteAdiantamento := totalAdiantamento - expensesFromAdiantamento
	totalGeralDisponivel := totalIncome - totalExpense

	c.JSON(http.StatusOK, gin.H{
		"month":                  month,
		"year":                   year,
		"total_income":           totalIncome,
		"total_expense":          totalExpense,
		"total_geral_disponivel": totalGeralDisponivel,
		"restante_salario":       restanteSalario,
		"restante_adiantamento":  restanteAdiantamento,
		"renda_extra_amt":        totalRendaExtra,
	})
}
