package main

import (
	"log"
	"os"

	"App_Financeiro_Back-end/internal/auth"
	"App_Financeiro_Back-end/internal/database"

	"github.com/gin-gonic/gin"
)

func main() {
	database.Connect()
	err := database.DB.AutoMigrate(&auth.User{})
	if err != nil {
		log.Fatalf("Erro ao rodar migrações: %v", err)
	}

	router := gin.Default()

	api := router.Group("/api")
	{
		auth.RegisterRoutes(api)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor rodando na porta %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}
