package main

import (
	"Sobra_Ai_Back-end/internal/appversion"
	"Sobra_Ai_Back-end/internal/assistant"
	"Sobra_Ai_Back-end/internal/auth"
	"Sobra_Ai_Back-end/internal/categories"
	"Sobra_Ai_Back-end/internal/database"
	"Sobra_Ai_Back-end/internal/expenses"
	"Sobra_Ai_Back-end/internal/incomes"
	"Sobra_Ai_Back-end/internal/migrations"
	"Sobra_Ai_Back-end/internal/reports"
	"Sobra_Ai_Back-end/internal/uploads"
	"Sobra_Ai_Back-end/internal/users"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: Arquivo .env não encontrado. Usando variáveis do sistema.")
	}

	if err := auth.ConfigureBotProtectionFromEnv(); err != nil {
		log.Fatalf("Configuracao anti-bot invalida: %v", err)
	}

	database.Connect()

	if err := migrations.Validate(database.DB); err != nil {
		log.Fatalf("Banco com migrations pendentes ou incompatíveis: %v", err)
	}

	router := gin.Default()
	if err := os.MkdirAll(uploads.Dir(), 0755); err != nil {
		log.Fatalf("Erro ao preparar pasta de uploads: %v", err)
	}

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(config))
	router.Static(uploads.PublicPath, uploads.Dir())

	api := router.Group("/api")
	{
		auth.RegisterRoutes(api)
		appversion.RegisterRoutes(api)

		protected := api.Group("/")
		protected.Use(auth.AuthMiddleware())
		{
			expenses.RegisterRoutes(protected)
			incomes.RegisterRoutes(protected)
			reports.RegisterRoutes(protected)
			categories.RegisterRoutes(protected)
			users.RegisterRoutes(protected)
			assistant.RegisterRoutes(protected)
		}
	}

	log.Println("Servidor a correr na porta 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}
