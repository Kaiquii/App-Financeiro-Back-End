package expenses

import (
	"App_Financeiro_Back-end/internal/database"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CreateExpenseRequest struct {
	Amount        float64   `json:"amount" binding:"required"`
	Description   string    `json:"description" binding:"required"`
	Category      string    `json:"category"`
	PaymentSource string    `json:"payment_source"`
	Date          time.Time `json:"date" binding:"required"`
	Type          string    `json:"type"`
	Installments  int       `json:"installments"`
}

func RegisterRoutes(rg *gin.RouterGroup) {
	expensesGroup := rg.Group("/expenses")
	{
		expensesGroup.POST("/", createExpense)
		expensesGroup.GET("/", getExpenses)
		expensesGroup.PATCH("/:id", updateExpense)
		expensesGroup.DELETE("/:id", deleteExpense)
	}
}

func getUserID(c *gin.Context) (uint, bool) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userIDObj.(uint), true
}

func createExpense(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	var req CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	installments := req.Installments
	if installments <= 0 {
		installments = 1
	}

	amountPerInstallment := req.Amount
	loopCount := 1

	if req.Type == "Parcelada" && installments > 1 {
		loopCount = installments
	} else if req.Type == "Fixa" {
		loopCount = 12
		installments = 12
	}

	for i := 1; i <= loopCount; i++ {
		installmentDate := req.Date.AddDate(0, i-1, 0)

		newExpense := Expense{
			UserID:         userID,
			Amount:         amountPerInstallment,
			Description:    req.Description,
			Category:       req.Category,
			PaymentSource:  req.PaymentSource,
			Date:           installmentDate,
			Month:          int(installmentDate.Month()),
			Year:           installmentDate.Year(),
			Type:           req.Type,
			Installments:   installments,
			CurrentInstall: i,
		}

		if err := database.DB.Create(&newExpense).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar parcela " + strconv.Itoa(i)})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Despesa(s) cadastrada(s) com sucesso!"})
}

func getExpenses(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	var expensesList []Expense
	query := database.DB.Where("user_id = ?", userID)
	monthStr := c.Query("month")
	yearStr := c.Query("year")

	if monthStr != "" {
		if month, err := strconv.Atoi(monthStr); err == nil {
			query = query.Where("month = ?", month)
		}
	}
	if yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			query = query.Where("year = ?", year)
		}
	}

	if err := query.Find(&expensesList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar despesas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": len(expensesList), "expenses": expensesList})
}

func updateExpense(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	id := c.Param("id")
	var expense Expense

	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Despesa não encontrada ou não pertence a você"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	updateFuture := false
	if val, exists := updateData["update_future"]; exists {
		if boolVal, ok := val.(bool); ok {
			updateFuture = boolVal
		}

		delete(updateData, "update_future")
	}

	if dateStr, ok := updateData["date"].(string); ok {
		if parsedDate, err := time.Parse(time.RFC3339, dateStr); err == nil {
			updateData["month"] = int(parsedDate.Month())
			updateData["year"] = parsedDate.Year()
		}
	}

	if err := database.DB.Model(&expense).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar despesa"})
		return
	}

	if updateFuture {
		delete(updateData, "date")
		delete(updateData, "month")
		delete(updateData, "year")

		database.DB.Model(&Expense{}).
			Where("user_id = ? AND description = ? AND (year > ? OR (year = ? AND month > ?))",
				userID, expense.Description, expense.Year, expense.Year, expense.Month).
			Updates(updateData)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Despesa atualizada com sucesso!"})
}

func deleteExpense(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	id := c.Param("id")
	var expense Expense

	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Despesa não encontrada ou não pertence a você"})
		return
	}

	if err := database.DB.Delete(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar despesa"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Despesa deletada com sucesso!"})
}
