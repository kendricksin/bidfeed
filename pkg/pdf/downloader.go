// pkg/pdf/downloader.go

package pdf

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bidfeed/pkg/models"
)

// Downloader handles PDF file downloads with constraints
type Downloader struct {
	client        *http.Client
	maxSizeMB     int
	tempDir       string
	timeoutSecs   int
	maxRetries    int
	retryDelaySec int
}

// NewDownloader creates a new PDF downloader instance
func NewDownloader(config *models.Config) *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: time.Duration(config.PDF.TimeoutSeconds) * time.Second,
		},
		maxSizeMB:     config.PDF.MaxSizeMB,
		tempDir:       config.PDF.TempDir,
		timeoutSecs:   config.PDF.TimeoutSeconds,
		maxRetries:    config.Errors.MaxRetries,
		retryDelaySec: config.Errors.RetryDelaySeconds,
	}
}

// Download retrieves a PDF from a URL and saves it to a temporary file
func (d *Downloader) Download(url string) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(d.retryDelaySec) * time.Second)
		}

		filePath, err := d.tryDownload(url)
		if err == nil {
			return filePath, nil
		}
		lastErr = err

		// Don't retry if it's a size limit error
		if err == ErrFileTooLarge {
			return "", err
		}
	}
	return "", fmt.Errorf("failed after %d attempts: %w", d.maxRetries, lastErr)
}

// ErrFileTooLarge is returned when the PDF exceeds the size limit
var ErrFileTooLarge = fmt.Errorf("PDF file exceeds size limit")

func (d *Downloader) tryDownload(url string) (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.timeoutSecs)*time.Second)
	defer cancel()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Set headers for PDF download
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/pdf,*/*")

	// Perform the request
	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error downloading PDF: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Check Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !isPDFContentType(contentType) {
		return "", fmt.Errorf("unexpected content type: %s", contentType)
	}

	// Check Content-Length if available
	if contentLength := resp.ContentLength; contentLength > 0 {
		sizeMB := float64(contentLength) / 1024 / 1024
		if sizeMB > float64(d.maxSizeMB) {
			return "", ErrFileTooLarge
		}
	}

	// Create temporary file
	tmpFile, err := d.createTempFile()
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %w", err)
	}

	// Use LimitReader to enforce size limit during download
	maxBytes := int64(d.maxSizeMB) * 1024 * 1024
	limitReader := io.LimitReader(resp.Body, maxBytes)

	// Copy with progress tracking
	written, err := io.Copy(tmpFile, limitReader)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("error saving PDF: %w", err)
	}

	if written >= maxBytes {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", ErrFileTooLarge
	}

	return tmpFile.Name(), nil
}

// createTempFile creates a temporary file with proper permissions
func (d *Downloader) createTempFile() (*os.File, error) {
	// Ensure temp directory exists
	if err := os.MkdirAll(d.tempDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating temp directory: %w", err)
	}

	return os.CreateTemp(d.tempDir, "pdf-*.pdf")
}

// isPDFContentType checks if the content type is valid for PDF
func isPDFContentType(contentType string) bool {
	validTypes := []string{
		"application/pdf",
		"application/x-pdf",
		"application/acrobat",
		"application/vnd.pdf",
		"text/pdf",
		"text/x-pdf",
	}

	for _, valid := range validTypes {
		if contentType == valid {
			return true
		}
	}

	return false
}

// CleanupOldFiles removes temporary files older than the specified duration
func (d *Downloader) CleanupOldFiles(maxAge time.Duration) error {
	entries, err := os.ReadDir(d.tempDir)
	if err != nil {
		return fmt.Errorf("error reading temp directory: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() && isPDFFile(entry.Name()) {
			path := filepath.Join(d.tempDir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if now.Sub(info.ModTime()) > maxAge {
				os.Remove(path)
			}
		}
	}

	return nil
}

// isPDFFile checks if the filename has a .pdf extension
func isPDFFile(filename string) bool {
	return filepath.Ext(filename) == ".pdf"
}
