package incomes

import (
	"App_Financeiro_Back-end/internal/database"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	incomesGroup := rg.Group("/incomes")
	{
		incomesGroup.POST("/", createIncome)
		incomesGroup.GET("/", getIncomes)
		incomesGroup.PATCH("/:id", updateIncome)
		incomesGroup.DELETE("/:id", deleteIncome)
	}
}

func getUserID(c *gin.Context) (uint, bool) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userIDObj.(uint), true
}

func createIncome(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	var req CreateIncomeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	loopCount := 1
	if req.Type == "Fixa" {
		loopCount = 60
	}

	currentMonth := req.Month
	currentYear := req.Year

	for i := 0; i < loopCount; i++ {
		newIncome := Income{
			UserID: userID,
			Source: req.Source,
			Amount: req.Amount,
			Month:  currentMonth,
			Year:   currentYear,
		}

		if err := database.DB.Create(&newIncome).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar renda"})
			return
		}

		currentMonth++
		if currentMonth > 12 {
			currentMonth = 1
			currentYear++
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Renda(s) cadastrada(s) com sucesso!"})
}

func getIncomes(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	var incomesList []Income
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

	if err := query.Find(&incomesList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar rendas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": len(incomesList), "incomes": incomesList})
}

func updateIncome(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	id := c.Param("id")
	var income Income

	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&income).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Renda não encontrada ou não pertence a você"})
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

	if err := database.DB.Model(&income).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar renda"})
		return
	}

	if updateFuture {
		database.DB.Model(&Income{}).
			Where("user_id = ? AND source = ? AND (year > ? OR (year = ? AND month > ?))",
				userID, income.Source, income.Year, income.Year, income.Month).
			Updates(updateData)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Renda(s) atualizada(s) com sucesso!"})
}

func deleteIncome(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	id := c.Param("id")
	var income Income

	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&income).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Renda não encontrada ou não pertence a você"})
		return
	}

	if err := database.DB.Delete(&income).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar renda"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Renda deletada com sucesso!"})
}
