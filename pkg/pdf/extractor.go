// pkg/pdf/extractor.go

package pdf

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"bidfeed/pkg/models"
	"bidfeed/pkg/utils"

	"github.com/ledongthuc/pdf"
)

// Extractor handles PDF content extraction and parsing
type Extractor struct {
	tempDir     string
	maxSizeMB   int
	timeoutSecs int
}

// NewExtractor creates a new PDF extractor instance
func NewExtractor(config *models.Config) *Extractor {
	return &Extractor{
		tempDir:     config.PDF.TempDir,
		maxSizeMB:   config.PDF.MaxSizeMB,
		timeoutSecs: config.PDF.TimeoutSeconds,
	}
}

// ExtractFromURL downloads and processes a PDF from a URL
func (e *Extractor) ExtractFromURL(url string, config *models.Config) (*models.PDFContent, error) {
	downloader := NewDownloader(config)

	// Download the PDF
	tmpPath, err := downloader.Download(url)
	if err != nil {
		return nil, fmt.Errorf("error downloading PDF: %w", err)
	}
	defer os.Remove(tmpPath)

	return e.ExtractFromFile(tmpPath)
}

// ExtractFromFile processes a PDF file and extracts structured content
func (e *Extractor) ExtractFromFile(filepath string) (*models.PDFContent, error) {
	f, r, err := pdf.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening PDF: %w", err)
	}
	defer f.Close()

	var content strings.Builder
	totalPages := r.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		p := r.Page(pageNum)
		if p.V.IsNull() {
			continue
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		content.WriteString(text)
	}

	return e.parseContent(content.String())
}

// parseContent extracts structured information from PDF text
func (e *Extractor) parseContent(text string) (*models.PDFContent, error) {
	result := &models.PDFContent{}

	// Extract budget
	if budget, err := e.extractBudget(text); err == nil {
		result.Budget = budget
	}

	// Extract specifications
	if specs, err := e.extractSpecs(text); err == nil {
		result.Specifications = specs
	}

	// Extract duration
	if duration, err := e.extractDuration(text); err == nil {
		result.Duration = duration
	}

	// Extract submission info
	if submission, err := e.extractSubmissionInfo(text); err == nil {
		result.SubmissionInfo = submission
	}

	// Extract contact info
	if contact, err := e.extractContactInfo(text); err == nil {
		result.ContactInfo = contact
	}

	return result, nil
}

// Regular expressions for content extraction
var (
	budgetRegex   = regexp.MustCompile(`([\d,]+\.?\d*)\s*บาท`)
	quantityRegex = regexp.MustCompile(`จำนวน\s*(\d+)`)
	yearRegex     = regexp.MustCompile(`ระยะเวลา\s*(\d+)\s*ปี`)
	monthRegex    = regexp.MustCompile(`\((\d+)\s*เดือน\)`)
	dateRegex     = regexp.MustCompile(`วันที่\s*(\d+.*\d{4})`)
	timeRegex     = regexp.MustCompile(`(\d{2}[:\.]\d{2})\s*น`)
	phoneRegex    = regexp.MustCompile(`โทรศัพท์.*?(\d[\d\-]+)`)
	emailRegex    = regexp.MustCompile(`([a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)`)
)

func (e *Extractor) extractBudget(text string) (float64, error) {
	matches := budgetRegex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return 0, fmt.Errorf("budget not found")
	}

	// Convert Thai numerals and clean amount
	amount := utils.ConvertThaiNumerals(matches[1])
	amount = utils.CleanAmount(amount)

	return strconv.ParseFloat(amount, 64)
}

func (e *Extractor) extractSpecs(text string) (string, error) {
	matches := quantityRegex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return "", fmt.Errorf("specifications not found")
	}
	return utils.ConvertThaiNumerals(matches[1]), nil
}

func (e *Extractor) extractDuration(text string) (models.Duration, error) {
	duration := models.Duration{}

	if matches := yearRegex.FindStringSubmatch(text); len(matches) > 1 {
		years, _ := strconv.Atoi(utils.ConvertThaiNumerals(matches[1]))
		duration.Years = years
	}

	if matches := monthRegex.FindStringSubmatch(text); len(matches) > 1 {
		months, _ := strconv.Atoi(utils.ConvertThaiNumerals(matches[1]))
		duration.Months = months
	}

	if duration.Years == 0 && duration.Months == 0 {
		return duration, fmt.Errorf("duration not found")
	}

	return duration, nil
}

func (e *Extractor) extractSubmissionInfo(text string) (models.SubmissionInfo, error) {
	info := models.SubmissionInfo{}

	if matches := dateRegex.FindStringSubmatch(text); len(matches) > 1 {
		info.Date = utils.ConvertThaiDate(matches[1])
	}

	if matches := timeRegex.FindStringSubmatch(text); len(matches) > 1 {
		info.Time = utils.ConvertThaiNumerals(matches[1])
	}

	if info.Date == "" && info.Time == "" {
		return info, fmt.Errorf("submission info not found")
	}

	return info, nil
}

func (e *Extractor) extractContactInfo(text string) (models.ContactInfo, error) {
	info := models.ContactInfo{}

	if matches := phoneRegex.FindStringSubmatch(text); len(matches) > 1 {
		info.Phone = utils.ConvertThaiNumerals(matches[1])
	}

	if matches := emailRegex.FindStringSubmatch(text); len(matches) > 1 {
		info.Email = matches[1]
	}

	if info.Phone == "" && info.Email == "" {
		return info, fmt.Errorf("contact info not found")
	}

	return info, nil
}
