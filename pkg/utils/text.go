// pkg/utils/text.go

package utils

import (
	"regexp"
	"strings"
	"unicode"
)

// Common regular expressions
var (
	// Match multiple whitespace characters
	whitespaceRegex = regexp.MustCompile(`\s+`)

	// Match common monetary formats
	monetaryRegex = regexp.MustCompile(`[฿$]?\s*[\d,]+\.?\d*\s*(?:บาท|THB|USD)?`)

	// Match date patterns (Thai and Western formats)
	dateRegex = regexp.MustCompile(`(?i)\d{1,2}[-/]\d{1,2}[-/]\d{2,4}|\d{1,2}\s+(?:ม\.?ค\.?|ก\.?พ\.?|มี\.?ค\.?|เม\.?ย\.?|พ\.?ค\.?|มิ\.?ย\.?|ก\.?ค\.?|ส\.?ค\.?|ก\.?ย\.?|ต\.?ค\.?|พ\.?ย\.?|ธ\.?ค\.?)\s+\d{4}`)
)

// CleanText removes excessive whitespace and normalizes text
func CleanText(text string) string {
	// Normalize whitespace
	text = whitespaceRegex.ReplaceAllString(text, " ")

	// Trim leading/trailing whitespace
	return strings.TrimSpace(text)
}

// ExtractMonetaryValues finds all monetary values in text
func ExtractMonetaryValues(text string) []string {
	return monetaryRegex.FindAllString(text, -1)
}

// ExtractDates finds all date strings in text
func ExtractDates(text string) []string {
	return dateRegex.FindAllString(text, -1)
}

// RemoveSpecialCharacters removes non-alphanumeric characters except specified ones
func RemoveSpecialCharacters(text string, keep string) string {
	keepSet := make(map[rune]bool)
	for _, r := range keep {
		keepSet[r] = true
	}

	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || keepSet[r] {
			return r
		}
		return -1
	}, text)
}

// NormalizeSpaces replaces consecutive spaces with a single space
func NormalizeSpaces(text string) string {
	return whitespaceRegex.ReplaceAllString(text, " ")
}

// TruncateText truncates text to specified length with ellipsis
func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Try to truncate at word boundary
	if idx := strings.LastIndex(text[:maxLength], " "); idx > 0 {
		return text[:idx] + "..."
	}

	return text[:maxLength] + "..."
}

// StripHTML removes HTML tags from text
func StripHTML(text string) string {
	// Remove HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text = tagRegex.ReplaceAllString(text, "")

	// Replace HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	return CleanText(text)
}

// SplitIntoWords splits text into words, handling Thai and English text
func SplitIntoWords(text string) []string {
	// Split on whitespace and punctuation
	words := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})

	// Filter out empty strings
	var result []string
	for _, word := range words {
		if word != "" {
			result = append(result, word)
		}
	}

	return result
}

// IsThaiText checks if text contains Thai characters
func IsThaiText(text string) bool {
	for _, r := range text {
		if unicode.In(r, unicode.Thai) {
			return true
		}
	}
	return false
}

// ContainsDigits checks if text contains numeric digits
func ContainsDigits(text string) bool {
	for _, r := range text {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// FormatPhoneNumber formats a Thai phone number consistently
func FormatPhoneNumber(phone string) string {
	// Remove all non-digit characters
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, phone)

	// Handle different phone number lengths
	switch len(digits) {
	case 9: // Land line
		return digits[:2] + "-" + digits[2:5] + "-" + digits[5:]
	case 10: // Mobile
		return digits[:3] + "-" + digits[3:6] + "-" + digits[6:]
	default:
		return digits
	}
}

// NormalizeNewlines converts all newline variants to \n
func NormalizeNewlines(text string) string {
	// Convert Windows style to Unix
	text = strings.ReplaceAll(text, "\r\n", "\n")
	// Convert old Mac style to Unix
	text = strings.ReplaceAll(text, "\r", "\n")
	// Normalize multiple newlines
	return regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
}
