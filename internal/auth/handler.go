package auth

import (
	"Sobra_Ai_Back-end/internal/database"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
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
			"name":       user.Name,
			"email":      user.Email,
			"role":       user.Role,
			"avatar_url": user.AvatarURL,
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

	if err := database.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao deletar usuário"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usuário deletado com sucesso"})
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
