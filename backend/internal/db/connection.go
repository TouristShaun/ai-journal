package db

import (
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewConnection(host, port, user, password, dbname string) (*DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable",
		host, port, user, dbname)
	if password != "" {
		psqlInfo += fmt.Sprintf(" password=%s", password)
	}
	
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Register pgvector
	// Note: pgvector-go types are registered automatically when used
	
	log.Println("Database connection established")
	
	return &DB{db}, nil
}

func (db *DB) RunMigrations() error {
	log.Println("Running database migrations...")
	
	// Run initial schema
	_, err := db.Exec(CreateTablesSQL)
	if err != nil {
		return fmt.Errorf("failed to run initial migrations: %w", err)
	}
	
	// Run processing tracker migration
	_, err = db.Exec(AddProcessingTrackerSQL)
	if err != nil {
		return fmt.Errorf("failed to run processing tracker migration: %w", err)
	}
	
	log.Println("Migrations completed successfully")
	return nil
}