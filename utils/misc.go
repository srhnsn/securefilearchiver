package utils

import (
	"fmt"
	"os"
	"strings"
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
