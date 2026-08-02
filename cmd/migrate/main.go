package main

import (
	"Sobra_Ai_Back-end/internal/database"
	"Sobra_Ai_Back-end/internal/migrations"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	if len(os.Args) < 2 {
		usage()
	}

	database.Connect()
	switch os.Args[1] {
	case "status":
		printStatus()
	case "up":
		if err := migrations.Up(database.DB); err != nil {
			log.Fatalf("Falha ao aplicar migrations: %v", err)
		}
		log.Println("Migrations aplicadas com sucesso.")
	case "baseline":
		if len(os.Args) != 3 || os.Args[2] != "--confirm-schema-reviewed" {
			log.Fatal("Baseline recusada. Revise o backup e o schema e execute: migrate baseline --confirm-schema-reviewed")
		}
		if err := migrations.Baseline(database.DB); err != nil {
			log.Fatalf("Falha ao registrar baseline: %v", err)
		}
		log.Println("Baseline registrada sem alterar as tabelas da aplicacao.")
	default:
		usage()
	}
}

func printStatus() {
	entries, err := migrations.Status(database.DB)
	if err != nil {
		log.Fatalf("Falha ao consultar migrations: %v", err)
	}
	for _, entry := range entries {
		state := "pending"
		appliedAt := "-"
		if entry.Applied {
			state = "applied"
			appliedAt = entry.AppliedAt.Format("2006-01-02 15:04:05Z07:00")
		}
		fmt.Printf("%06d  %-8s  %-32s  %s\n", entry.Version, state, entry.Name, appliedAt)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Uso: migrate status | migrate up | migrate baseline --confirm-schema-reviewed")
	os.Exit(2)
}
