package main

import (
	"App_Financeiro_Back-end/internal/auth"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	api := router.Group("/api")
	{
		auth.RegisterRoutes(api)
	}

	log.Println("Iniciando o servidor na porta 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}
