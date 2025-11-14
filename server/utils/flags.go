package utils

import (
	"database/sql"
	"fmt"
	"slices"

	"forum/server/config"
	"forum/server/migrations"
)

var ValidFlags = []string{"--migrate", "--seed", "--drop", "--migrate-up", "--migrate-down", "--migrate-status"}

func HandleFlags(flags []string, db *sql.DB) error {
	if len(flags) != 1 {
		return fmt.Errorf("expected a single flag, got %d", len(flags))
	}

	flag := flags[0]
	if !slices.Contains(ValidFlags, flag) {
		return fmt.Errorf("invalid flag: '%s'", flag)
	}

	switch flag {
	case "--migrate":
		return config.CreateTables(db)
	case "--seed":
		return config.CreateDemoData(db)
	case "--drop":
		return config.Drop()
	case "--migrate-up":
		cfg := config.LoadConfig()
		migrationsDir := cfg.App.BasePath + "server/database/migrations"
		migrator := migrations.NewMigrator(db, migrationsDir)
		if err := migrator.InitMigrationsTable(); err != nil {
			return err
		}
		return migrator.Up()
	case "--migrate-down":
		cfg := config.LoadConfig()
		migrationsDir := cfg.App.BasePath + "server/database/migrations"
		migrator := migrations.NewMigrator(db, migrationsDir)
		return migrator.Down()
	case "--migrate-status":
		cfg := config.LoadConfig()
		migrationsDir := cfg.App.BasePath + "server/database/migrations"
		migrator := migrations.NewMigrator(db, migrationsDir)
		if err := migrator.InitMigrationsTable(); err != nil {
			return err
		}
		return migrator.Status()
	}
	return nil
}

func Usage() {
	fmt.Println(`Usage: go run main.go [option]
Options:
  --migrate         Create database tables (legacy)
  --seed            Insert demo data into the database
  --drop            Drop all tables
  
  --migrate-up      Apply all pending migrations
  --migrate-down    Rollback last applied migration
  --migrate-status  Show migration status`)
}
