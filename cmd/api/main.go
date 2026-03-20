package main

import (
	"App_Financeiro_Back-end/internal/auth"
	"App_Financeiro_Back-end/internal/database"
	"log"

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

	log.Println("Servidor rodando na porta 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}
