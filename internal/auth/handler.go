package auth

import (
	"Sobra_Ai_Back-end/internal/assistant"
	"Sobra_Ai_Back-end/internal/categories"
	"Sobra_Ai_Back-end/internal/database"
	"Sobra_Ai_Back-end/internal/expenses"
	"Sobra_Ai_Back-end/internal/incomes"
	"Sobra_Ai_Back-end/internal/pagination"
	"Sobra_Ai_Back-end/internal/uploads"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"mime"
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
		authGroup.POST("/request-register-code", requestRegisterCode)
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
		adminGroup.GET("/users/metrics", getUserMetrics)
		adminGroup.DELETE("/users/:id", deleteUserByAdmin)
		adminGroup.PATCH("/users/:id/revoke-access", revokeUserAccessByAdmin)
		adminGroup.PATCH("/users/:id/restore-access", restoreUserAccessByAdmin)
	}
}

func getUserMetrics(c *gin.Context) {
	var metrics UserMetricsResponse
	err := database.DB.Model(&User{}).
		Select(`
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN access_blocked = ? THEN 1 ELSE 0 END), 0) AS active,
			COALESCE(SUM(CASE WHEN access_blocked = ? THEN 1 ELSE 0 END), 0) AS blocked,
			COALESCE(SUM(CASE WHEN role = ? THEN 1 ELSE 0 END), 0) AS admins
		`, false, true, "admin").
		Scan(&metrics).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar metricas de usuarios"})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

const (
	registrationCodeTTL          = 10 * time.Minute
	registrationRateLimitWindow  = 30 * time.Minute
	maxRegistrationCodesByEmail  = 3
	maxRegistrationCodesByIP     = 5
	passwordResetRateLimitWindow = 30 * time.Minute
	maxPasswordResetsByEmail     = 3
	maxPasswordResetsByIP        = 5
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isBlockedRegistrationEmail(email string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return true
	}

	domain := strings.ToLower(parts[1])
	blockedDomains := map[string]bool{
		"example.com":   true,
		"example.org":   true,
		"example.net":   true,
		"test.com":      true,
		"localhost.com": true,
		"invalid.com":   true,
	}

	return blockedDomains[domain]
}

func exceededRegistrationCodeLimit(email string, ipAddress string) bool {
	since := time.Now().Add(-registrationRateLimitWindow)

	var emailCount int64
	database.DB.Model(&RegistrationCode{}).
		Where("email = ? AND created_at > ?", email, since).
		Count(&emailCount)
	if emailCount >= maxRegistrationCodesByEmail {
		return true
	}

	var ipCount int64
	database.DB.Model(&RegistrationCode{}).
		Where("ip_address = ? AND created_at > ?", ipAddress, since).
		Count(&ipCount)

	return ipCount >= maxRegistrationCodesByIP
}

func exceededPasswordResetLimit(email string, ipAddress string) bool {
	since := time.Now().Add(-passwordResetRateLimitWindow)

	var emailCount int64
	database.DB.Model(&PasswordResetToken{}).
		Where("email = ? AND created_at > ?", email, since).
		Count(&emailCount)
	if emailCount >= maxPasswordResetsByEmail {
		return true
	}

	var ipCount int64
	database.DB.Model(&PasswordResetToken{}).
		Where("ip_address = ? AND created_at > ?", ipAddress, since).
		Count(&ipCount)

	return ipCount >= maxPasswordResetsByIP
}

func requestRegisterCode(c *gin.Context) {
	var req RequestRegisterCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados invalidos: " + err.Error()})
		return
	}

	email := normalizeEmail(req.Email)
	ipAddress := c.ClientIP()

	if isBlockedRegistrationEmail(email) {
		log.Printf("Cadastro bloqueado por dominio invalido email=%s ip=%s user_agent=%q", email, ipAddress, c.Request.UserAgent())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Use um e-mail valido"})
		return
	}

	var existingUser User
	if err := database.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Este e-mail ja esta cadastrado"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao verificar e-mail"})
		return
	}

	if exceededRegistrationCodeLimit(email, ipAddress) {
		log.Printf("Envio de codigo de cadastro limitado email=%s ip=%s user_agent=%q", email, ipAddress, c.Request.UserAgent())
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Muitas tentativas de cadastro. Tente novamente mais tarde."})
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

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao iniciar cadastro"})
		return
	}

	if err := tx.Model(&RegistrationCode{}).
		Where("email = ? AND used = false", email).
		Update("used", true).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao invalidar codigos antigos"})
		return
	}

	registrationCode := RegistrationCode{
		Email:     email,
		CodeHash:  string(codeHash),
		IPAddress: ipAddress,
		ExpiresAt: time.Now().Add(registrationCodeTTL),
		Used:      false,
	}

	if err := tx.Create(&registrationCode).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar codigo"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao confirmar codigo"})
		return
	}

	if err := sendRegistrationCodeEmail(email, code); err != nil {
		log.Printf("Erro ao enviar codigo de cadastro email=%s ip=%s erro=%v", email, ipAddress, err)
		if deleteErr := database.DB.Delete(&registrationCode).Error; deleteErr != nil {
			log.Printf("Erro ao remover codigo de cadastro apos falha no envio email=%s ip=%s erro=%v", email, ipAddress, deleteErr)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao enviar e-mail"})
		return
	}

	log.Printf("Codigo de cadastro enviado email=%s ip=%s user_agent=%q", email, ipAddress, c.Request.UserAgent())
	c.JSON(http.StatusOK, gin.H{"message": "Enviamos um codigo para confirmar seu e-mail."})
}

func register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = normalizeEmail(req.Email)
	req.Code = strings.TrimSpace(req.Code)

	if isBlockedRegistrationEmail(req.Email) {
		log.Printf("Cadastro bloqueado por dominio invalido email=%s ip=%s user_agent=%q", req.Email, c.ClientIP(), c.Request.UserAgent())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Use um e-mail valido"})
		return
	}

	var existingUser User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Este e-mail já está cadastrado"})
		return
	}

	var registrationCode RegistrationCode
	if err := database.DB.
		Where("email = ? AND used = false AND expires_at > ?", req.Email, time.Now()).
		Order("created_at desc").
		First(&registrationCode).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Codigo invalido ou expirado"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(registrationCode.CodeHash), []byte(req.Code)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Codigo invalido ou expirado"})
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

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao iniciar cadastro"})
		return
	}

	if err := tx.Create(&newUser).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar no banco de dados"})
		return
	}

	registrationCode.Used = true
	if err := tx.Save(&registrationCode).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao invalidar codigo"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao confirmar cadastro"})
		return
	}

	log.Printf("Usuario criado por cadastro validado id=%d email=%s ip=%s user_agent=%q", newUser.ID, newUser.Email, c.ClientIP(), c.Request.UserAgent())
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

	req.Email = normalizeEmail(req.Email)
	ipAddress := c.ClientIP()

	if loginAttempts.isBlocked(req.Email, ipAddress, time.Now()) {
		log.Printf("Login limitado email=%s ip=%s user_agent=%q", req.Email, ipAddress, c.Request.UserAgent())
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Muitas tentativas de login. Tente novamente em alguns minutos."})
		return
	}

	var user User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		loginAttempts.recordFailure(req.Email, ipAddress, time.Now())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "E-mail ou senha incorretos"})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		loginAttempts.recordFailure(req.Email, ipAddress, time.Now())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "E-mail ou senha incorretos"})
		return
	}

	loginAttempts.clear(req.Email, ipAddress)

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
	params, err := pagination.Parse(c.Request.URL.Query())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var usersList []User
	if !params.Enabled {
		if err := database.DB.Find(&usersList).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar usuarios"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"total": len(usersList), "users": usersList})
		return
	}

	var total int64
	if err := database.DB.Model(&User{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao contar usuarios"})
		return
	}
	if err := database.DB.
		Order("id asc").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&usersList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar usuarios"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":      total,
		"users":      usersList,
		"pagination": pagination.NewMetadata(params, total),
	})
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
	return sendAuthEmail(to, "Código para redefinir sua senha", passwordResetEmailHTML(name, code))
}

func sendRegistrationCodeEmail(to string, code string) error {
	return sendAuthEmail(to, "Código para criar sua conta", registrationCodeEmailHTML(code))
}

func sendAuthEmail(to string, subject string, htmlBody string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, host)

	message := []byte(
		"To: " + to + "\r\n" +
			"Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n" +
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

	req.Email = normalizeEmail(req.Email)
	ipAddress := c.ClientIP()

	var user User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Se o e-mail existir, um codigo sera enviado."})
		return
	}

	if exceededPasswordResetLimit(user.Email, ipAddress) {
		log.Printf("Envio de codigo de redefinicao limitado email=%s ip=%s user_agent=%q", user.Email, ipAddress, c.Request.UserAgent())
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Muitas tentativas. Tente novamente mais tarde."})
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
		IPAddress: ipAddress,
		CodeHash:  string(codeHash),
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Used:      false,
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao iniciar redefinicao de senha"})
		return
	}

	if err := tx.Model(&PasswordResetToken{}).
		Where("email = ? AND used = false", user.Email).
		Update("used", true).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao invalidar codigos antigos"})
		return
	}

	if err := tx.Create(&resetToken).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar codigo"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao confirmar codigo"})
		return
	}

	if err := sendPasswordResetEmail(user.Email, user.Name, code); err != nil {
		log.Printf("Erro ao enviar codigo de redefinicao de senha email=%s ip=%s erro=%v", user.Email, ipAddress, err)
		if deleteErr := database.DB.Delete(&resetToken).Error; deleteErr != nil {
			log.Printf("Erro ao remover codigo de redefinicao apos falha no envio email=%s ip=%s erro=%v", user.Email, ipAddress, deleteErr)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao enviar e-mail"})
		return
	}

	log.Printf("Codigo de redefinicao enviado email=%s ip=%s user_agent=%q", user.Email, ipAddress, c.Request.UserAgent())
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
