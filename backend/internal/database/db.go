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
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	log.Printf("Connected to database: %s", cfg.DBName)
	return db, nil
}
