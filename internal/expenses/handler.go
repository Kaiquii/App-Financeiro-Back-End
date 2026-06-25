package expenses

import (
	"Sobra_Ai_Back-end/internal/database"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const fixedExpenseHorizonMonths = 120
const maxExpenseNotesLength = 500

func normalizeExpenseType(expenseType string) string {
	switch strings.TrimSpace(strings.ToLower(expenseType)) {
	case "única", "unica":
		return "Única"
	case "parcelada":
		return "Parcelada"
	case "fixa":
		return "Fixa"
	default:
		return ""
	}
}

func buildSeriesID(userID uint) string {
	return fmt.Sprintf("expense-%d-%d", userID, time.Now().UnixNano())
}

func normalizeExpenseNotes(notes string) (string, bool) {
	normalizedNotes := strings.TrimSpace(notes)
	return normalizedNotes, len([]rune(normalizedNotes)) <= maxExpenseNotesLength
}

func RegisterRoutes(rg *gin.RouterGroup) {
	expensesGroup := rg.Group("/expenses")
	{
		expensesGroup.POST("/", createExpense)
		expensesGroup.GET("/", getExpenses)
		expensesGroup.GET("/:id", getExpenseByID)
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

	notes, ok := normalizeExpenseNotes(req.Notes)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Observações devem ter no máximo 500 caracteres"})
		return
	}

	expenseType := normalizeExpenseType(req.Type)
	if expenseType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo inválido. Use Única, Parcelada ou Fixa."})
		return
	}

	installments := req.Installments
	if installments <= 0 {
		installments = 1
	}

	loopCount := 1
	seriesID := ""

	switch expenseType {
	case "Única":
		installments = 1

	case "Parcelada":
		loopCount = installments
		seriesID = buildSeriesID(userID)

	case "Fixa":
		loopCount = fixedExpenseHorizonMonths
		installments = 0
		seriesID = buildSeriesID(userID)
	}

	for i := 1; i <= loopCount; i++ {
		installmentDate := req.Date.AddDate(0, i-1, 0)

		newExpense := Expense{
			UserID:         userID,
			SeriesID:       seriesID,
			Amount:         req.Amount,
			Description:    req.Description,
			Notes:          notes,
			CategoryID:     req.CategoryID,
			PaymentSource:  req.PaymentSource,
			Date:           installmentDate,
			Month:          int(installmentDate.Month()),
			Year:           installmentDate.Year(),
			Type:           expenseType,
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
	typeStr := c.Query("type")

	if typeStr != "" {
		normalizedType := normalizeExpenseType(typeStr)
		if normalizedType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo inválido. Use Única, Parcelada ou Fixa."})
			return
		}
		query = query.Where("type = ?", normalizedType)
	}

	if err := query.Find(&expensesList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar despesas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": len(expensesList), "expenses": expensesList})
}

func getExpenseByID(c *gin.Context) {
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

	c.JSON(http.StatusOK, expense)
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

	originalMonth := expense.Month
	originalYear := expense.Year

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

	if notesValue, exists := updateData["notes"]; exists {
		if notesValue == nil {
			updateData["notes"] = ""
		} else {
			notesText, ok := notesValue.(string)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Observações inválidas"})
				return
			}

			notes, valid := normalizeExpenseNotes(notesText)
			if !valid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Observações devem ter no máximo 500 caracteres"})
				return
			}
			updateData["notes"] = notes
		}
	}

	if typeValue, exists := updateData["type"]; exists {
		typeStr, ok := typeValue.(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo inválido. Use Única, Parcelada ou Fixa."})
			return
		}

		normalizedType := normalizeExpenseType(typeStr)
		if normalizedType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo inválido. Use Única, Parcelada ou Fixa."})
			return
		}

		updateData["type"] = normalizedType

		if normalizedType == "Única" {
			updateData["installments"] = 1
		}
		if normalizedType == "Fixa" {
			updateData["installments"] = 0
		}
	}

	if dateStr, ok := updateData["date"].(string); ok {
		if parsedDate, err := time.Parse(time.RFC3339, dateStr); err == nil {
			updateData["date"] = parsedDate
			updateData["month"] = int(parsedDate.Month())
			updateData["year"] = parsedDate.Year()
		}
	}

	if expense.Type == "Fixa" {
		updateData["installments"] = 0
	}

	if err := database.DB.Model(&expense).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar despesa"})
		return
	}

	if updateFuture {
		delete(updateData, "date")
		delete(updateData, "month")
		delete(updateData, "year")
		delete(updateData, "id")
		delete(updateData, "user_id")
		delete(updateData, "series_id")
		delete(updateData, "current_installment")

		err := applySeriesScope(database.DB.Model(&Expense{}), expense).
			Where("user_id = ? AND (year > ? OR (year = ? AND month > ?))",
				userID, originalYear, originalYear, originalMonth).
			Updates(updateData).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar despesas futuras"})
			return
		}
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
	deleteFuture := c.Query("delete_future") == "true"

	var expense Expense
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Despesa não encontrada ou não pertence a você"})
		return
	}

	if expense.Type == "Fixa" {
		deleteFuture = true
	}

	if deleteFuture {
		err := applySeriesScope(database.DB, expense).
			Where("user_id = ? AND (year > ? OR (year = ? AND month >= ?))",
				userID, expense.Year, expense.Year, expense.Month).
			Delete(&Expense{}).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar despesas em lote"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Despesa atual e todas as futuras foram removidas!"})
		return
	}

	if err := database.DB.Delete(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar despesa"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Despesa deletada com sucesso!"})
}

func applySeriesScope(query *gorm.DB, expense Expense) *gorm.DB {
	if expense.SeriesID != "" {
		return query.Where("series_id = ?", expense.SeriesID)
	}
	return query.Where("description = ?", expense.Description)
}
