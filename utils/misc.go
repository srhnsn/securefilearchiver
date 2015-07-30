package utils

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	humanRangePattern = regexp.MustCompile("(\\d+)([sihdwmy])")

	humanRangeTokens = map[string]time.Duration{
		"s": time.Second,
		"i": time.Minute,
		"h": time.Hour,
		"d": time.Hour * 24,
		"w": time.Hour * 24 * 7,
		"m": time.Hour * 24 * 30,
		"y": time.Hour * 24 * 365,
	}
)

// FileExists checks if a specified path (file or directory) exists.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// FixSlashes replaces all \ with / in a string.
func FixSlashes(input string) string {
	return strings.Replace(input, "\\", "/", -1)
}

// FormatFileSize converts file sizes in human-readable formats.
func FormatFileSize(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KiB", float64(bytes)/1024)
	}

	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MiB", float64(bytes)/1024/1024)
	}

	if bytes < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.1f GiB", float64(bytes)/1024/1024/1024)
	}

	return fmt.Sprintf("%.1f TiB", float64(bytes)/1024/1024/1024/1024)
}

// ParseHumanRange parses human time ranges into time.Durations.
func ParseHumanRange(input string) (time.Duration, error) {
	match := humanRangePattern.FindAllStringSubmatch(input, -1)

	if match == nil {
		return 0, fmt.Errorf("invalid human time range input: %s", input)
	}

	var duration time.Duration

	for _, token := range match {
		amount, err := strconv.ParseInt(token[1], 10, 64)

		if err != nil {
			Error.Panicln(err)
		}

		duration = time.Duration(amount) * parseSingleHumanRange(token[2])
	}

	return duration, nil
}

func parseSingleHumanRange(input string) time.Duration {
	duration, exists := humanRangeTokens[input]

	if !exists {
		Error.Panicf("invalid token: %s", input)
	}

	return duration
}
