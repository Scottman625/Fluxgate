package db

import (
	"database/sql"
	"fmt"

	"queue-system/internal/config"

	_ "github.com/lib/pq"
)

func NewPostgresDB(cfg *config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 設定連接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}
