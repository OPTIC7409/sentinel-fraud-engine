package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	_ "github.com/lib/pq"
)

// migrationsDir returns .../database/migrations next to this file (works no matter the process cwd).
func migrationsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("could not resolve migrations directory")
	}
	return filepath.Join(filepath.Dir(file), "migrations")
}

const (
	defaultDatabaseURL = "postgres://postgres:postgres@localhost:5433/sentinel_fraud?sslmode=disable"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run migrate.go [up|down]")
		os.Exit(1)
	}

	command := os.Args[1]
	if command != "up" && command != "down" {
		log.Fatal("Command must be 'up' or 'down'")
	}

	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultDatabaseURL
		log.Printf("Using default DATABASE_URL: %s", dbURL)
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	dir := migrationsDir()
	pattern := filepath.Join(dir, fmt.Sprintf("*.%s.sql", command))
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatalf("Failed to read migration files: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("No migration files found matching %s", pattern)
	}

	// Sort files to ensure correct order
	sort.Strings(files)

	// If running down migrations, reverse the order
	if command == "down" {
		for i := len(files)/2 - 1; i >= 0; i-- {
			opp := len(files) - 1 - i
			files[i], files[opp] = files[opp], files[i]
		}
	}

	// Execute each migration
	for _, file := range files {
		log.Printf("Running migration: %s", filepath.Base(file))

		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		// Execute migration
		if _, err := db.Exec(string(content)); err != nil {
			log.Fatalf("Failed to execute migration %s: %v", file, err)
		}

		log.Printf("✓ Completed: %s", filepath.Base(file))
	}

	log.Printf("All migrations completed successfully!")
}
