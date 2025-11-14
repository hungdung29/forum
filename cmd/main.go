package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"forum/server/config"
	"forum/server/migrations"
	"forum/server/routes"
	"forum/server/utils"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load configuration from environment
	cfg := config.LoadConfig()
	
	// Update BasePath for backward compatibility
	if cfg.App.BasePath != "" {
		config.BasePath = cfg.App.BasePath
	}

	// Connect to the database
	db, err := config.Connect()
	if err != nil {
		log.Fatal("Database connection error:", err)
	}

	// Handle database setup based on environment
	if cfg.App.BasePath != "" {
		// Running in Docker/production - run migrations automatically
		log.Println("Running database migrations...")
		migrationsDir := cfg.App.BasePath + "server/database/migrations"
		migrator := migrations.NewMigrator(db, migrationsDir)
		
		// Initialize migrations table
		if err := migrator.InitMigrationsTable(); err != nil {
			log.Fatalf("Failed to initialize migrations table: %v", err)
		}
		
		// Run pending migrations
		if err := migrator.Up(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		
		log.Println("Database setup complete.")
	} else {
		// Handle command-line flags for database setup
		if len(os.Args) > 1 {
			if err := utils.HandleFlags(os.Args[1:], db); err != nil {
				fmt.Println(err)
				utils.Usage()
				os.Exit(1)
			}
			return
		}
	}
	

	
	// Start the HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      routes.Routes(db),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine so it doesn't block
	go func() {
		log.Printf("Server starting on http://localhost:%d (Environment: %s)", 
			cfg.Server.Port, cfg.App.Environment)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()	// Wait for interrupt signal (Ctrl+C or kill command)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")

	// Give existing requests 30 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server stopped gracefully")
}
