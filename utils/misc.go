package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// TmpSuffix is the suffix for all temporary files.
	TmpSuffix = ".tmp"
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

// MustWriteFile writes data to filename. It panics if there is an error.
func MustWriteFile(filename string, data []byte) {
	err := ioutil.WriteFile(filename, data, 0700)

	PanicIfErr(err)
}

// MustWriteFileAtomic first writes the data to a temporary file, then renames it.
func MustWriteFileAtomic(filename string, data []byte) {
	tmpFilename := filename + TmpSuffix

	MustWriteFile(tmpFilename, data)
	err := os.Rename(tmpFilename, filename)

	PanicIfErr(err)
}

// PanicIfErr panics if the argument is not nil.
func PanicIfErr(err error) {
	if err == nil {
		return
	}

	Error.Panicln(err)
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

		PanicIfErr(err)

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
