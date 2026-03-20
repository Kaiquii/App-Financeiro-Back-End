package database

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=4343 dbname=app_financeiro port=5432 sslmode=disable TimeZone=America/Sao_Paulo"
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Falha ao conectar no banco de dados: %v", err)
	}

	log.Println("Conexão com o banco de dados estabelecida!")
}
