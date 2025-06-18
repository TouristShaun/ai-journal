package main

import (
	"flag"
	"log"
	"os"

	"github.com/journal/internal/db"
)

func main() {
	var direction string
	flag.StringVar(&direction, "direction", "up", "Migration direction: up or down")
	flag.Parse()

	if direction != "up" && direction != "down" {
		log.Fatal("Invalid direction. Use 'up' or 'down'")
	}

	// Database configuration
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "journal_db")

	// Connect to database
	database, err := db.NewConnection(dbHost, dbPort, dbUser, dbPassword, dbName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if direction == "up" {
		if err := database.RunMigrations(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations completed successfully")
	} else {
		log.Println("Down migrations not implemented yet")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
