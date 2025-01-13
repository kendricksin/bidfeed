// cmd/main.go

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"bidfeed/pkg/database"
	"bidfeed/pkg/models"
	"bidfeed/pkg/pipeline"

	"gopkg.in/yaml.v2"
)

// findConfigFile looks for config.yml in various locations
func findConfigFile() (string, error) {
	// Possible config locations
	locations := []string{
		"config/config.yml",                       // From current directory
		"../config/config.yml",                    // One level up
		filepath.Join("cmd", "config/config.yml"), // From cmd directory
	}

	// Get executable directory
	ex, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(ex)
		// Add executable directory locations
		locations = append(locations,
			filepath.Join(execDir, "config/config.yml"),
			filepath.Join(execDir, "../config/config.yml"),
		)
	}

	// Try each location
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			absPath, err := filepath.Abs(loc)
			if err != nil {
				continue
			}
			return absPath, nil
		}
	}

	return "", fmt.Errorf("config file not found in any of the expected locations")
}

// loadConfig reads and parses the config file
func loadConfig() (*models.Config, error) {
	configPath, err := findConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find config file: %w", err)
	}

	log.Printf("Loading config from: %s\n", configPath)

	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config models.Config
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Validate required paths are absolute
	config.Database.Path, err = filepath.Abs(config.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid database path: %w", err)
	}

	config.Logging.Path, err = filepath.Abs(config.Logging.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid logging path: %w", err)
	}

	config.PDF.TempDir, err = filepath.Abs(config.PDF.TempDir)
	if err != nil {
		return nil, fmt.Errorf("invalid temp directory path: %w", err)
	}

	return &config, nil
}

// setupLogger configures the application logger
func setupLogger(logPath string) (*log.Logger, error) {
	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.OpenFile(
		logPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create a multi-writer for both file and stdout
	return log.New(logFile, "", log.LstdFlags|log.Lshortfile), nil
}

func main() {
	// Setup initial stdout logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Get and log current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %v\n", err)
	}
	log.Printf("Current working directory: %s\n", cwd)

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Setup logger
	logger, err := setupLogger(config.Logging.Path)
	if err != nil {
		log.Fatalf("Logger setup error: %v", err)
	}

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

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize the processor
	processor := pipeline.NewProcessor(config, db, logger)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the pipeline in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- processor.Start(ctx)
	}()

	// Start periodic cleanup of temporary files
	go func() {
		ticker := time.NewTicker(time.Hour) // Run cleanup every hour
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupAge := time.Duration(config.PDF.CleanupAfterHours) * time.Hour
				if err := os.RemoveAll(config.PDF.TempDir); err != nil {
					logger.Printf("Error cleaning up temp directory: %v", err)
				}
				if err := os.MkdirAll(config.PDF.TempDir, 0755); err != nil {
					logger.Printf("Error recreating temp directory: %v", err)
				}
				logger.Printf("Cleaned up temporary files older than %v", cleanupAge)
			}
		}
	}()

	// Main loop
	for {
		select {
		case err := <-errChan:
			if err != nil {
				logger.Printf("Pipeline error: %v", err)
			}
			// Start a new pipeline run after the configured interval
			time.Sleep(time.Duration(config.Processing.PollIntervalSeconds) * time.Second)
			go func() {
				errChan <- processor.Start(ctx)
			}()

		case sig := <-sigChan:
			logger.Printf("Received signal: %v", sig)
			logger.Println("Initiating graceful shutdown...")

			// Cancel context to stop ongoing operations
			cancel()

			// Wait for cleanup (you might want to add a timeout here)
			time.Sleep(5 * time.Second)

			logger.Println("Shutdown completed")
			return
		}
	}
}
