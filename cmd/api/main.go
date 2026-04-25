package main

import (
	"App_Financeiro_Back-end/internal/auth"
	"App_Financeiro_Back-end/internal/categories"
	"App_Financeiro_Back-end/internal/database"
	"App_Financeiro_Back-end/internal/expenses"
	"App_Financeiro_Back-end/internal/incomes"
	"App_Financeiro_Back-end/internal/reports"
	"App_Financeiro_Back-end/internal/users"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: Arquivo .env não encontrado. Usando variáveis do sistema.")
	}

	database.Connect()

	err := database.DB.AutoMigrate(&auth.User{}, &auth.PasswordResetToken{}, &expenses.Expense{}, &incomes.Income{}, &categories.Category{})
	if err != nil {
		log.Fatalf("Erro ao rodar migrações: %v", err)
	}

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"} // Liberado para o qlq dominio
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(config))

	api := router.Group("/api")
	{
		auth.RegisterRoutes(api)

		protected := api.Group("/")
		protected.Use(auth.AuthMiddleware())
		{
			expenses.RegisterRoutes(protected)
			incomes.RegisterRoutes(protected)
			reports.RegisterRoutes(protected)
			categories.RegisterRoutes(protected)
			users.RegisterRoutes(protected)
		}
	}

	log.Println("Servidor a correr na porta 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}
