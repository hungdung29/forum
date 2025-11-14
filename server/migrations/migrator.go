package migrations

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version   string
	Name      string
	UpSQL     string
	DownSQL   string
	AppliedAt time.Time
}

// Migrator handles database migrations
type Migrator struct {
	db            *sql.DB
	migrationsDir string
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *sql.DB, migrationsDir string) *Migrator {
	return &Migrator{
		db:            db,
		migrationsDir: migrationsDir,
	}
}

// InitMigrationsTable creates the migrations tracking table
func (m *Migrator) InitMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	return nil
}

// GetAppliedMigrations returns list of applied migration versions
func (m *Migrator) GetAppliedMigrations() (map[string]time.Time, error) {
	applied := make(map[string]time.Time)
	
	rows, err := m.db.Query("SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var version string
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		applied[version] = appliedAt
	}
	
	return applied, nil
}

// GetPendingMigrations returns migrations that haven't been applied
func (m *Migrator) GetPendingMigrations() ([]Migration, error) {
	// Get applied migrations
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}
	
	// Read all migration files
	files, err := ioutil.ReadDir(m.migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}
	
	var pending []Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".up.sql") {
			continue
		}
		
		// Extract version from filename (e.g., "001_create_users.up.sql" -> "001")
		version := strings.Split(file.Name(), "_")[0]
		
		// Skip if already applied
		if _, exists := applied[version]; exists {
			continue
		}
		
		// Read migration content
		upPath := filepath.Join(m.migrationsDir, file.Name())
		upSQL, err := ioutil.ReadFile(upPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", file.Name(), err)
		}
		
		// Extract name
		nameParts := strings.Split(file.Name(), "_")
		name := strings.TrimSuffix(strings.Join(nameParts[1:], "_"), ".up.sql")
		
		pending = append(pending, Migration{
			Version: version,
			Name:    name,
			UpSQL:   string(upSQL),
		})
	}
	
	// Sort by version
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})
	
	return pending, nil
}

// Up applies all pending migrations
func (m *Migrator) Up() error {
	pending, err := m.GetPendingMigrations()
	if err != nil {
		return err
	}
	
	if len(pending) == 0 {
		fmt.Println("No pending migrations")
		return nil
	}
	
	for _, migration := range pending {
		fmt.Printf("Applying migration %s: %s...\n", migration.Version, migration.Name)
		
		// Start transaction
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		
		// Execute migration
		if _, err := tx.Exec(migration.UpSQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
		}
		
		// Record migration
		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			migration.Version, migration.Name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}
		
		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
		}
		
		fmt.Printf("✓ Migration %s applied successfully\n", migration.Version)
	}
	
	fmt.Printf("\nApplied %d migration(s)\n", len(pending))
	return nil
}

// Down rolls back the last applied migration
func (m *Migrator) Down() error {
	// Get last applied migration
	row := m.db.QueryRow("SELECT version, name FROM schema_migrations ORDER BY version DESC LIMIT 1")
	
	var version, name string
	if err := row.Scan(&version, &name); err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No migrations to rollback")
			return nil
		}
		return fmt.Errorf("failed to get last migration: %w", err)
	}
	
	// Read down migration file
	downFile := fmt.Sprintf("%s_%s.down.sql", version, name)
	downPath := filepath.Join(m.migrationsDir, downFile)
	downSQL, err := ioutil.ReadFile(downPath)
	if err != nil {
		return fmt.Errorf("failed to read down migration %s: %w", downFile, err)
	}
	
	fmt.Printf("Rolling back migration %s: %s...\n", version, name)
	
	// Start transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	
	// Execute down migration
	if _, err := tx.Exec(string(downSQL)); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to rollback migration %s: %w", version, err)
	}
	
	// Remove migration record
	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record %s: %w", version, err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback %s: %w", version, err)
	}
	
	fmt.Printf("✓ Migration %s rolled back successfully\n", version)
	return nil
}

// Status shows current migration status
func (m *Migrator) Status() error {
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}
	
	pending, err := m.GetPendingMigrations()
	if err != nil {
		return err
	}
	
	fmt.Println("\nMigration Status:")
	fmt.Println("=================")
	fmt.Printf("Applied: %d\n", len(applied))
	fmt.Printf("Pending: %d\n\n", len(pending))
	
	if len(applied) > 0 {
		fmt.Println("Applied Migrations:")
		// Get sorted versions
		var versions []string
		for v := range applied {
			versions = append(versions, v)
		}
		sort.Strings(versions)
		
		for _, v := range versions {
			fmt.Printf("  ✓ %s (applied at %s)\n", v, applied[v].Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}
	
	if len(pending) > 0 {
		fmt.Println("Pending Migrations:")
		for _, m := range pending {
			fmt.Printf("  ○ %s: %s\n", m.Version, m.Name)
		}
	}
	
	return nil
}
