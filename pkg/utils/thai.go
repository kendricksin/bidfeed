// pkg/utils/thai.go

package utils

import "strings"

// ThaiNumeralMap maps Thai numerals to Arabic numerals
var ThaiNumeralMap = map[rune]rune{
	'๐': '0',
	'๑': '1',
	'๒': '2',
	'๓': '3',
	'๔': '4',
	'๕': '5',
	'๖': '6',
	'๗': '7',
	'๘': '8',
	'๙': '9',
}

// ConvertThaiNumerals converts Thai numerals to Arabic numerals
func ConvertThaiNumerals(text string) string {
	var result strings.Builder
	for _, char := range text {
		if arabic, ok := ThaiNumeralMap[char]; ok {
			result.WriteRune(arabic)
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// CleanAmount removes commas and whitespace from monetary amounts
func CleanAmount(amount string) string {
	return strings.ReplaceAll(strings.TrimSpace(amount), ",", "")
}

// ThaiMonthMap maps Thai month names to numeric representation
var ThaiMonthMap = map[string]string{
	"มกราคม":     "01",
	"กุมภาพันธ์": "02",
	"มีนาคม":     "03",
	"เมษายน":     "04",
	"พฤษภาคม":    "05",
	"มิถุนายน":   "06",
	"กรกฎาคม":    "07",
	"สิงหาคม":    "08",
	"กันยายน":    "09",
	"ตุลาคม":     "10",
	"พฤศจิกายน":  "11",
	"ธันวาคม":    "12",
}

// ConvertThaiDate converts a Thai format date to a standardized format
func ConvertThaiDate(thaiDate string) string {
	// Convert numerals first
	arabicDate := ConvertThaiNumerals(thaiDate)

	// Replace Thai month names with numbers
	for thai, arabic := range ThaiMonthMap {
		arabicDate = strings.ReplaceAll(arabicDate, thai, arabic)
	}

	return arabicDate
}
