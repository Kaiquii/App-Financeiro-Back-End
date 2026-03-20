package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var mockDB = make(map[string]User)
var jwtSecret = []byte("minha_chave_super_secreta_app_financeiro")

func RegisterRoutes(rg *gin.RouterGroup) {
	authGroup := rg.Group("/auth")
	{
		authGroup.POST("/register", register)
		authGroup.POST("/login", login)

		protected := authGroup.Group("/")
		protected.Use(AuthMiddleware())
		{
			protected.GET("/users", getUsers)
			protected.PATCH("/users", updatePassword)
		}
	}
}

func register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	if _, exists := mockDB[req.Email]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Este e-mail já está cadastrado"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar a senha"})
		return
	}

	mockDB[req.Email] = User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Usuário " + req.Name + " criado com sucesso!"})
}

func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	user, exists := mockDB[req.Email]
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "E-mail ou senha incorretos"})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "E-mail ou senha incorretos"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login realizado com sucesso!", "token": tokenString})
}

func getUsers(c *gin.Context) {
	var usersList []User

	for _, user := range mockDB {
		usersList = append(usersList, user)
	}

	c.JSON(http.StatusOK, gin.H{"total": len(usersList), "users": usersList})
}

func updatePassword(c *gin.Context) {
	loggedEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Não foi possível identificar o usuário do token"})
		return
	}

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	emailStr := loggedEmail.(string)
	user, found := mockDB[emailStr]
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado no sistema"})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "A senha antiga está incorreta"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar a nova senha"})
		return
	}

	user.Password = string(hashedPassword)
	mockDB[emailStr] = user

	c.JSON(http.StatusOK, gin.H{"message": "Senha atualizada com sucesso!"})
}
