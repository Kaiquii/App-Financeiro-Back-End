package auth

import (
	"Sobra_Ai_Back-end/internal/assistant"
	"Sobra_Ai_Back-end/internal/categories"
	"Sobra_Ai_Back-end/internal/database"
	"Sobra_Ai_Back-end/internal/expenses"
	"Sobra_Ai_Back-end/internal/incomes"
	"Sobra_Ai_Back-end/internal/uploads"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	authGroup := rg.Group("/auth")
	{
		authGroup.POST("/register", register)
		authGroup.POST("/login", login)
		authGroup.POST("/forgot-password", forgotPassword)
		authGroup.POST("/reset-password", resetPassword)

		protected := authGroup.Group("/")
		protected.Use(AuthMiddleware())
		{
			protected.GET("/users", getUsers)
		}
	}

	adminGroup := rg.Group("/admin")
	adminGroup.Use(AuthMiddleware(), AdminMiddleware())
	{
		adminGroup.DELETE("/users/:id", deleteUserByAdmin)
		adminGroup.PATCH("/users/:id/revoke-access", revokeUserAccessByAdmin)
		adminGroup.PATCH("/users/:id/restore-access", restoreUserAccessByAdmin)
	}
}

func register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	req.Email = strings.ToLower(req.Email)

	var existingUser User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Este e-mail já está cadastrado"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar a senha"})
		return
	}

	newUser := User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
		Role:     "user",
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar no banco de dados"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Usuário " + req.Name + " criado com sucesso!",
		"user_id": newUser.ID,
	})
}

func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	req.Email = strings.ToLower(req.Email)

	var user User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "E-mail ou senha incorretos"})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "E-mail ou senha incorretos"})
		return
	}

	if user.AccessBlocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Acesso revogado. Entre em contato com o suporte."})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})

	secret := []byte(os.Getenv("JWT_SECRET"))
	tokenString, err := token.SignedString(secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login realizado com sucesso!",
		"token":   tokenString,
		"user": gin.H{
			"name":           user.Name,
			"email":          user.Email,
			"role":           user.Role,
			"avatar_url":     user.AvatarURL,
			"access_blocked": user.AccessBlocked,
		},
	})
}

func getUsers(c *gin.Context) {
	var usersList []User

	database.DB.Find(&usersList)

	c.JSON(http.StatusOK, gin.H{"total": len(usersList), "users": usersList})
}

func deleteUserByAdmin(c *gin.Context) {
	id := c.Param("id")

	var user User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado"})
		return
	}

	if user.Role == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Não é permitido deletar outro administrador"})
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao iniciar exclusao do usuario"})
		return
	}

	if err := deleteUserData(tx, user.ID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar dados do usuario"})
		return
	}

	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar usuario"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao confirmar exclusao do usuario"})
		return
	}

	if err := deleteUserAvatar(user.ID); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Usuario e dados deletados com sucesso, mas houve erro ao remover a foto de perfil",
			"warning": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usuario e todos os dados dele foram deletados com sucesso"})
}

func deleteUserData(tx *gorm.DB, userID uint) error {
	deletions := []func() error{
		func() error { return tx.Where("user_id = ?", userID).Delete(&assistant.Message{}).Error },
		func() error { return tx.Where("user_id = ?", userID).Delete(&assistant.Conversation{}).Error },
		func() error { return tx.Where("user_id = ?", userID).Delete(&expenses.Expense{}).Error },
		func() error { return tx.Where("user_id = ?", userID).Delete(&incomes.Income{}).Error },
		func() error { return tx.Where("user_id = ?", userID).Delete(&categories.Category{}).Error },
		func() error { return tx.Where("user_id = ?", userID).Delete(&PasswordResetToken{}).Error },
	}

	for _, deleteFn := range deletions {
		if err := deleteFn(); err != nil {
			return err
		}
	}

	return nil
}

func deleteUserAvatar(userID uint) error {
	storage, err := uploads.NewStorage()
	if err != nil {
		return err
	}

	userIDText := strconv.FormatUint(uint64(userID), 10)
	return storage.Delete(uploads.UserAvatarKey(userIDText))
}

func revokeUserAccessByAdmin(c *gin.Context) {
	adminID, ok := currentAdminID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Administrador nao identificado"})
		return
	}

	user, ok := findManagedUser(c)
	if !ok {
		return
	}

	if user.ID == adminID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Nao e permitido revogar o proprio acesso"})
		return
	}

	if user.Role == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Nao e permitido revogar acesso de outro administrador"})
		return
	}

	now := time.Now()
	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"access_blocked":    true,
		"access_blocked_at": &now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao revogar acesso do usuario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Acesso do usuario revogado com sucesso",
		"user":    accessUserResponse(user.ID, user.Name, user.Email, user.Role, true, &now),
	})
}

func restoreUserAccessByAdmin(c *gin.Context) {
	user, ok := findManagedUser(c)
	if !ok {
		return
	}

	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"access_blocked":    false,
		"access_blocked_at": nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao liberar acesso do usuario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Acesso do usuario liberado com sucesso",
		"user":    accessUserResponse(user.ID, user.Name, user.Email, user.Role, false, nil),
	})
}

func currentAdminID(c *gin.Context) (uint, bool) {
	adminIDObj, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	adminID, ok := adminIDObj.(uint)
	return adminID, ok
}

func findManagedUser(c *gin.Context) (User, bool) {
	id := c.Param("id")

	var user User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario nao encontrado"})
		return user, false
	}

	return user, true
}

func accessUserResponse(id uint, name string, email string, role string, blocked bool, blockedAt *time.Time) gin.H {
	return gin.H{
		"id":                id,
		"name":              name,
		"email":             email,
		"role":              role,
		"access_blocked":    blocked,
		"access_blocked_at": blockedAt,
	}
}

func generateResetCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func sendPasswordResetEmail(to string, name string, code string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, host)

	subject := "Codigo para redefinir sua senha"
	htmlBody := passwordResetEmailHTML(name, code)

	message := []byte(
		"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
			"\r\n" +
			htmlBody + "\r\n",
	)

	return smtp.SendMail(host+":"+port, auth, from, []string{to}, message)
}

func forgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados invalidos: " + err.Error()})
		return
	}

	req.Email = strings.ToLower(req.Email)

	var user User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Se o e-mail existir, um codigo sera enviado."})
		return
	}

	code, err := generateResetCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar codigo"})
		return
	}

	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar codigo"})
		return
	}

	resetToken := PasswordResetToken{
		UserID:    user.ID,
		Email:     user.Email,
		CodeHash:  string(codeHash),
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Used:      false,
	}

	if err := database.DB.Create(&resetToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar codigo"})
		return
	}

	if err := sendPasswordResetEmail(user.Email, user.Name, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao enviar e-mail"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Se o e-mail existir, um codigo sera enviado."})
}

func resetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados invalidos: " + err.Error()})
		return
	}

	req.Email = strings.ToLower(req.Email)

	var token PasswordResetToken
	if err := database.DB.
		Where("email = ? AND used = false AND expires_at > ?", req.Email, time.Now()).
		Order("created_at desc").
		First(&token).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Codigo invalido ou expirado"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(token.CodeHash), []byte(req.Code)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Codigo invalido ou expirado"})
		return
	}

	var user User
	if err := database.DB.First(&user, token.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario nao encontrado"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar nova senha"})
		return
	}

	tx := database.DB.Begin()

	user.Password = string(hashedPassword)
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar senha"})
		return
	}

	token.Used = true
	if err := tx.Save(&token).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao invalidar codigo"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Senha atualizada com sucesso!"})
}
