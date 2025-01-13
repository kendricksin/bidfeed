// pkg/pipeline/processor.go

package pipeline

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"bidfeed/pkg/database"
	"bidfeed/pkg/feed"
	"bidfeed/pkg/models"
	"bidfeed/pkg/pdf"
)

// Processor coordinates the pipeline operations
type Processor struct {
	config     *models.Config
	db         *database.Database
	collector  *feed.Collector
	extractor  *pdf.Extractor
	downloader *pdf.Downloader
	logger     *log.Logger
	workerPool *WorkerPool
}

// NewProcessor creates a new pipeline processor
func NewProcessor(
	config *models.Config,
	db *database.Database,
	logger *log.Logger,
) *Processor {
	return &Processor{
		config:     config,
		db:         db,
		collector:  feed.NewCollector(config),
		extractor:  pdf.NewExtractor(config),
		downloader: pdf.NewDownloader(config),
		logger:     logger,
		workerPool: NewWorkerPool(
			config.Processing.ConcurrentDownloads,
			config.Processing.BatchSize,
			logger,
		),
	}
}

// Start begins the processing pipeline
func (p *Processor) Start(ctx context.Context) error {
	p.logger.Println("Starting processing pipeline...")

	// Start the worker pool
	p.workerPool.Start(ctx)
	defer p.workerPool.Stop()

	// Process each configured department
	for _, dept := range p.config.Departments {
		if err := p.processDepartment(ctx, dept.ID); err != nil {
			p.logger.Printf("Error processing department %s: %v", dept.ID, err)
		}
	}

	return nil
}

// processDepartment handles collection and processing for a single department
func (p *Processor) processDepartment(ctx context.Context, deptID string) error {
	p.logger.Printf("Processing department: %s", deptID)

	// Collect feed entries
	entries, err := p.collector.FetchFeed(deptID)
	if err != nil {
		return fmt.Errorf("error fetching feed: %w", err)
	}

	// Filter and submit entries for processing
	for _, entry := range entries {
		if err := p.processEntry(ctx, &entry); err != nil {
			p.logError(entry.ID, "process_entry", err)
		}
	}

	return nil
}

// processEntry handles a single feed entry
func (p *Processor) processEntry(ctx context.Context, entry *models.FeedEntry) error {
	// Check if entry should be processed based on keywords
	if !p.shouldProcess(entry) {
		p.logger.Printf("Skipping entry %s due to keyword filtering", entry.ID)
		return p.db.UpdateFeedEntryStatus(entry.ID, models.StatusFiltered)
	}

	// Store entry in database
	if err := p.db.InsertFeedEntry(entry); err != nil {
		return fmt.Errorf("error storing feed entry: %w", err)
	}

	// Create and submit processing job
	job := NewProcessingJob(entry.ID, func(jobCtx context.Context) error {
		return p.processEntryContent(jobCtx, entry)
	})

	p.workerPool.Submit(job)
	return nil
}

// processEntryContent downloads and extracts content from a feed entry
func (p *Processor) processEntryContent(ctx context.Context, entry *models.FeedEntry) error {
	// Update status to processing
	if err := p.db.UpdateFeedEntryStatus(entry.ID, models.StatusProcessing); err != nil {
		return err
	}

	// Extract content from PDF
	content, err := p.extractor.ExtractFromURL(entry.PDFURL, p.config)
	if err != nil {
		p.db.UpdateFeedEntryStatus(entry.ID, models.StatusFailed)
		return fmt.Errorf("error extracting PDF content: %w", err)
	}

	// Create project record
	project := &models.Project{
		ID:          entry.ID,
		FeedEntryID: entry.ID,
		Title:       entry.Title,
		DeptID:      entry.DeptID,
		Budget:      content.Budget,
		PDFContent:  fmt.Sprintf("%+v", content), // Store structured content as string
	}

	// Store project in database
	if err := p.db.InsertProject(project); err != nil {
		p.db.UpdateFeedEntryStatus(entry.ID, models.StatusFailed)
		return fmt.Errorf("error storing project: %w", err)
	}

	// Update status to completed
	return p.db.UpdateFeedEntryStatus(entry.ID, models.StatusCompleted)
}

// shouldProcess checks if an entry matches the processing criteria
func (p *Processor) shouldProcess(entry *models.FeedEntry) bool {
	title := entry.Title

	// Check exclude keywords
	for _, keyword := range p.config.Keywords.Exclude {
		if containsIgnoreCase(title, keyword) {
			return false
		}
	}

	// Check include keywords
	if len(p.config.Keywords.Include) > 0 {
		matched := false
		for _, keyword := range p.config.Keywords.Include {
			if containsIgnoreCase(title, keyword) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// logError records a processing error to the database
func (p *Processor) logError(source, errorType string, err error) {
	error := &models.ProcessingError{
		ID:        fmt.Sprintf("%s-%d", source, time.Now().Unix()),
		Source:    source,
		ErrorType: errorType,
		Message:   err.Error(),
	}

	if err := p.db.LogError(error); err != nil {
		p.logger.Printf("Error logging error: %v", err)
	}
}

// containsIgnoreCase checks if a string contains a substring (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}
