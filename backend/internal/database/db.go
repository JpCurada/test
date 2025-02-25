package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/ISKOnnect/iskonnect-web/internal/config"
	_ "github.com/lib/pq"
)

func Connect(cfg config.DatabaseConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Failed to open database connection: %v", err)
		return nil, err
	}

	if err = db.Ping(); err != nil {
		log.Printf("Failed to ping database: %v", err)
		return nil, err
	}

	log.Printf("Successfully connected to database: %s", cfg.DBName)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	return db, nil
}
