package categories

import (
	"App_Financeiro_Back-end/internal/database"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	catGroup := rg.Group("/categories")
	{
		catGroup.POST("/", createCategory)
		catGroup.GET("/", getCategories)
		catGroup.PATCH("/:id", updateCategory)
		catGroup.DELETE("/:id", deleteCategory)
	}
}

func getUserID(c *gin.Context) (uint, bool) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userIDObj.(uint), true
}

func createCategory(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	newCat := Category{
		UserID: userID,
		Name:   req.Name,
	}

	if err := database.DB.Create(&newCat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar categoria"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Categoria criada!", "data": newCat})
}

func getCategories(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	var categories []Category
	if err := database.DB.Where("user_id = ?", userID).Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar categorias"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": len(categories), "categories": categories})
}

func updateCategory(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	id := c.Param("id")
	var cat Category

	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&cat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Categoria não encontrada ou não pertence a você"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	if err := database.DB.Model(&cat).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar categoria"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Categoria atualizada com sucesso!"})
}

func deleteCategory(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não identificado"})
		return
	}

	id := c.Param("id")

	var count int64
	database.DB.Table("expenses").Where("category_id = ? AND user_id = ?", id, userID).Count(&count)

	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Não é possível apagar esta categoria porque você tem " + strconv.FormatInt(count, 10) + " despesa(s) vinculada(s) a ela.",
		})
		return
	}

	var cat Category
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&cat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Categoria não encontrada"})
		return
	}

	if err := database.DB.Delete(&cat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar categoria"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Categoria deletada com sucesso!"})
}
