// bidfeed/pkg/models/models.gos

package models

import "time"

// Config holds all configuration settings
type Config struct {
	Database struct {
		Path           string `yaml:"path"`
		MaxConnections int    `yaml:"max_connections"`
		TimeoutSeconds int    `yaml:"timeout_seconds"`
	} `yaml:"database"`

	Departments []struct {
		ID   string `yaml:"id"`
		Name string `yaml:"name"`
	} `yaml:"departments"`

	Feed struct {
		BaseURL        string `yaml:"base_url"`
		TimeoutSeconds int    `yaml:"timeout_seconds"`
		AllowedTimes   []struct {
			Start string `yaml:"start"`
			End   string `yaml:"end"`
		} `yaml:"allowed_times"`
		MaxEntries   int `yaml:"max_entries"`
		LookbackDays int `yaml:"lookback_days"`
	} `yaml:"feed"`

	Keywords struct {
		Include   []string `yaml:"include"`
		Exclude   []string `yaml:"exclude"`
		MinBudget float64  `yaml:"min_budget"`
	} `yaml:"keywords"`

	PDF struct {
		TempDir           string `yaml:"temp_dir"`
		CleanupAfterHours int    `yaml:"cleanup_after_hours"`
		MaxSizeMB         int    `yaml:"max_size_mb"`
		TimeoutSeconds    int    `yaml:"timeout_seconds"`
	} `yaml:"pdf"`

	Errors struct {
		MaxRetries        int `yaml:"max_retries"`
		RetryDelaySeconds int `yaml:"retry_delay_seconds"`
		AlertThreshold    int `yaml:"alert_threshold"`
	} `yaml:"errors"`

	Logging struct {
		Path       string `yaml:"path"`
		Level      string `yaml:"level"`
		MaxSizeMB  int    `yaml:"max_size_mb"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAgeDays int    `yaml:"max_age_days"`
	} `yaml:"logging"`

	Processing struct {
		BatchSize           int `yaml:"batch_size"`
		ConcurrentDownloads int `yaml:"concurrent_downloads"`
		PollIntervalSeconds int `yaml:"poll_interval_seconds"`
	} `yaml:"processing"`
}

// FeedEntry represents a row in the feed_entries table
type FeedEntry struct {
	ID          string    `json:"id"`
	DeptID      string    `json:"dept_id"`
	Title       string    `json:"title"`
	PDFURL      string    `json:"pdf_url"`
	PublishDate time.Time `json:"publish_date"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Project represents a row in the projects table
type Project struct {
	ID          string    `json:"id"`
	FeedEntryID string    `json:"feed_entry_id"`
	Title       string    `json:"title"`
	DeptID      string    `json:"dept_id"`
	Budget      float64   `json:"budget"`
	PDFContent  string    `json:"pdf_content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProcessingError represents a row in the processing_errors table
type ProcessingError struct {
	ID         string    `json:"id"`
	Source     string    `json:"source"`
	ErrorType  string    `json:"error_type"`
	Message    string    `json:"message"`
	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// PDFContent represents structured content extracted from a PDF
type PDFContent struct {
	Budget         float64        `json:"budget"`
	Specifications string         `json:"specifications"`
	Duration       Duration       `json:"duration"`
	SubmissionInfo SubmissionInfo `json:"submission_info"`
	ContactInfo    ContactInfo    `json:"contact_info"`
}

// EntryStatus represents the possible states of a feed entry
const (
	StatusNew        = "new"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusFiltered   = "filtered"
)

// ErrorType represents types of processing errors
const (
	ErrorTypeFetch    = "fetch_error"
	ErrorTypeParse    = "parse_error"
	ErrorTypeDownload = "download_error"
	ErrorTypeExtract  = "extraction_error"
	ErrorTypeDatabase = "database_error"
)

// Duration represents a time period in years and months
type Duration struct {
	Years  int `json:"years"`
	Months int `json:"months"`
}

// SubmissionInfo contains bid submission date and time
type SubmissionInfo struct {
	Date string `json:"date"`
	Time string `json:"time"`
}

// ContactInfo contains contact details
type ContactInfo struct {
	Phone string `json:"phone"`
	Email string `json:"email"`
}
