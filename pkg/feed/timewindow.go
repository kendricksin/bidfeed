// pkg/feed/timewindow.go

package feed

import (
	"fmt"
	"strings"
	"time"
)

// TimeWindow represents an allowed time period
type TimeWindow struct {
	Start time.Time
	End   time.Time
}

// isWithinAllowedTime checks if the current time falls within allowed collection windows
func (c *Collector) isWithinAllowedTime() bool {
	now := time.Now()

	for _, window := range c.config.Feed.AllowedTimes {
		start, err := parseTimeString(window.Start)
		if err != nil {
			continue
		}

		end, err := parseTimeString(window.End)
		if err != nil {
			continue
		}

		currentHour := now.Hour()
		currentMinute := now.Minute()
		startHour := start.Hour()
		startMinute := start.Minute()
		endHour := end.Hour()
		endMinute := end.Minute()

		// Handle windows that cross midnight
		if endHour < startHour || (endHour == startHour && endMinute < startMinute) {
			// Check if current time is either after start or before end
			if currentHour > startHour ||
				(currentHour == startHour && currentMinute >= startMinute) ||
				currentHour < endHour ||
				(currentHour == endHour && currentMinute <= endMinute) {
				return true
			}
		} else {
			// Normal time window comparison
			if (currentHour > startHour ||
				(currentHour == startHour && currentMinute >= startMinute)) &&
				(currentHour < endHour ||
					(currentHour == endHour && currentMinute <= endMinute)) {
				return true
			}
		}
	}

	return false
}

// parseTimeString converts a time string (HH:MM) to time.Time
func parseTimeString(timeStr string) (time.Time, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	now := time.Now()
	timeStr = fmt.Sprintf("%d-%02d-%02d %s:00",
		now.Year(), now.Month(), now.Day(), timeStr)

	return time.Parse("2006-01-02 15:04:05", timeStr)
}
