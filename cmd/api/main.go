package main

import (
	"App_Financeiro_Back-end/internal/auth"
	"App_Financeiro_Back-end/internal/database"
	"App_Financeiro_Back-end/internal/expenses"
	"App_Financeiro_Back-end/internal/incomes"
	"App_Financeiro_Back-end/internal/reports"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	database.Connect()

	err := database.DB.AutoMigrate(&auth.User{}, &expenses.Expense{}, &incomes.Income{})
	if err != nil {
		log.Fatalf("Erro ao rodar migrações: %v", err)
	}

	router := gin.Default()

	api := router.Group("/api")
	{
		auth.RegisterRoutes(api)

		protected := api.Group("/")
		protected.Use(auth.AuthMiddleware())
		{
			expenses.RegisterRoutes(protected)
			incomes.RegisterRoutes(protected)
			reports.RegisterRoutes(protected)
		}
	}

	log.Println("Servidor a correr na porta 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}
