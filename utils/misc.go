package utils

import (
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
