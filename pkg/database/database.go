// bidfeed/pkg/database/database.go

package database

import (
	"database/sql"
	"os"
	"path/filepath"

	"bidfeed/pkg/models"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents our SQLite connection and operations
type Database struct {
	*sql.DB
}

// InitDB initializes the database and creates tables if they don't exist
func InitDB(dbPath string) (*Database, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create tables
	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Database{db}, nil
}

func createTables(db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS feed_entries (
			id TEXT PRIMARY KEY,
			dept_id TEXT NOT NULL,
			title TEXT NOT NULL,
			pdf_url TEXT,
			publish_date DATETIME,
			status TEXT DEFAULT 'new',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			feed_entry_id TEXT,
			title TEXT NOT NULL,
			dept_id TEXT NOT NULL,
			budget REAL,
			pdf_content TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(feed_entry_id) REFERENCES feed_entries(id)
		)`,
		`CREATE TABLE IF NOT EXISTS processing_errors (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			error_type TEXT NOT NULL,
			message TEXT,
			retry_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Indexes for better query performance
		`CREATE INDEX IF NOT EXISTS idx_feed_entries_status ON feed_entries(status)`,
		`CREATE INDEX IF NOT EXISTS idx_feed_entries_dept ON feed_entries(dept_id)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_dept ON projects(dept_id)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return err
		}
	}

	return nil
}

// InsertFeedEntry adds a new feed entry to the database
func (db *Database) InsertFeedEntry(entry *models.FeedEntry) error {
	query := `
		INSERT INTO feed_entries (id, dept_id, title, pdf_url, publish_date, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		entry.ID,
		entry.DeptID,
		entry.Title,
		entry.PDFURL,
		entry.PublishDate,
		entry.Status,
	)
	return err
}

// UpdateFeedEntryStatus updates the status of a feed entry
func (db *Database) UpdateFeedEntryStatus(id, status string) error {
	query := `
		UPDATE feed_entries 
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := db.Exec(query, status, id)
	return err
}

// GetPendingFeedEntries retrieves feed entries that need processing
func (db *Database) GetPendingFeedEntries() ([]models.FeedEntry, error) {
	query := `
		SELECT id, dept_id, title, pdf_url, publish_date, status, created_at, updated_at
		FROM feed_entries
		WHERE status = ?
		ORDER BY publish_date DESC
	`
	rows, err := db.Query(query, models.StatusNew)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.FeedEntry
	for rows.Next() {
		var entry models.FeedEntry
		err := rows.Scan(
			&entry.ID,
			&entry.DeptID,
			&entry.Title,
			&entry.PDFURL,
			&entry.PublishDate,
			&entry.Status,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// LogError records a processing error
func (db *Database) LogError(error *models.ProcessingError) error {
	query := `
		INSERT INTO processing_errors (id, source, error_type, message, retry_count)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		error.ID,
		error.Source,
		error.ErrorType,
		error.Message,
		error.RetryCount,
	)
	return err
}
