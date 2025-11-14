package config

import (
	"database/sql"
	"fmt"
)

func Connect() (*sql.DB, error) {
	cfg := LoadConfig()
	
	dbPath := cfg.App.BasePath + cfg.Database.Path
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool from config
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Test the database connection
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}
