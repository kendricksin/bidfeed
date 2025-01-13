// bidfeed/cmd/main.go

package main

import (
	"log"
	"os"
	"path/filepath"

	"bidfeed/pkg/database"
	"bidfeed/pkg/models"

	"gopkg.in/yaml.v2"
)

// loadConfig reads and parses the config file
func loadConfig(configPath string) (*models.Config, error) {
	log.Printf("Loading config from: %s\n", configPath)

	file, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Error reading config file: %v\n", err)
		return nil, err
	}

	var config models.Config
	if err := yaml.Unmarshal(file, &config); err != nil {
		log.Printf("Error parsing config file: %v\n", err)
		return nil, err
	}

	return &config, nil
}

// setupLogger configures the application logger
func setupLogger(logPath string) (*log.Logger, error) {
	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(
		logPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, err
	}

	return log.New(logFile, "", log.LstdFlags), nil
}

func main() {
	// Get current working directory for debugging
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %v\n", err)
	}
	log.Printf("Current working directory: %s\n", cwd)

	// Load configuration
	config, err := loadConfig("config/config.yml")
	if err != nil {
		log.Printf("Failed to load configuration from %s: %v\n", filepath.Join(cwd, "config/config.yml"), err)
		log.Fatal(err)
	}

	// Setup logger
	logger, err := setupLogger(config.Logging.Path)
	if err != nil {
		log.Fatal("Failed to setup logger:", err)
	}
	logger.Println("Starting EGP Pipeline...")

	// Initialize database
	db, err := database.InitDB(config.Database.Path)
	if err != nil {
		logger.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create temp directory if it doesn't exist
	if err := os.MkdirAll(config.PDF.TempDir, 0755); err != nil {
		logger.Fatal("Failed to create temp directory:", err)
	}

	logger.Println("Initialization completed successfully")

	// TODO: Initialize and start pipeline components
	// 1. Feed collector
	// 2. PDF processor
	// 3. Data extractor
}
