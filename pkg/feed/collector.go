// pkg/feed/collector.go

package feed

import (
	"bidfeed/pkg/models"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

// RSSFeed represents the XML structure of the e-GP RSS feed
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Items []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// Collector handles fetching and processing of RSS feeds
type Collector struct {
	config     *models.Config
	httpClient *http.Client
}

// NewCollector creates a new feed collector instance
func NewCollector(config *models.Config) *Collector {
	return &Collector{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Feed.TimeoutSeconds) * time.Second,
		},
	}
}

// FetchFeed retrieves the RSS feed for a specific department
func (c *Collector) FetchFeed(deptID string) ([]models.FeedEntry, error) {
	if !c.isWithinAllowedTime() {
		return nil, fmt.Errorf("current time is outside allowed collection windows")
	}

	url := c.buildURL(deptID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers similar to Python implementation
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,th;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching feed: %w", err)
	}
	defer resp.Body.Close()

	// Handle CP874 (Windows-874) encoding
	decoder := charmap.Windows874.NewDecoder()
	reader := decoder.Reader(resp.Body)

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	feed, err := c.parseFeed(content)
	if err != nil {
		return nil, fmt.Errorf("error parsing feed: %w", err)
	}

	return c.convertToFeedEntries(feed, deptID)
}

func (c *Collector) buildURL(deptID string) string {
	baseURL := c.config.Feed.BaseURL
	if deptID != "" {
		if !strings.Contains(baseURL, "?") {
			baseURL += "?"
		} else {
			baseURL += "&"
		}
		baseURL += fmt.Sprintf("deptId=%s", deptID)
	}
	return baseURL
}

func (c *Collector) parseFeed(content []byte) (*RSSFeed, error) {
	var feed RSSFeed
	if err := xml.Unmarshal(content, &feed); err != nil {
		return nil, fmt.Errorf("error unmarshaling XML: %w", err)
	}
	return &feed, nil
}

func (c *Collector) convertToFeedEntries(feed *RSSFeed, deptID string) ([]models.FeedEntry, error) {
	var entries []models.FeedEntry

	for _, item := range feed.Channel.Items {
		pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			// Try alternative date format if standard parsing fails
			pubDate, err = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", item.PubDate)
			if err != nil {
				return nil, fmt.Errorf("error parsing date %s: %w", item.PubDate, err)
			}
		}

		entry := models.FeedEntry{
			ID:          generateEntryID(item.Link),
			DeptID:      deptID,
			Title:       item.Title,
			PDFURL:      item.Link,
			PublishDate: pubDate,
			Status:      models.StatusNew,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// generateEntryID creates a unique ID from the PDF URL
func generateEntryID(url string) string {
	// Extract the last part of the URL and clean it
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return strings.TrimSuffix(parts[len(parts)-1], ".pdf")
	}
	return url
}
