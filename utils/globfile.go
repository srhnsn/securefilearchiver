package utils

import (
	"bufio"
	"os"
	"strings"

	"github.com/ryanuber/go-glob"
)

// Globfile holds a list of globs to match against.
type Globfile struct {
	globs []string
}

// NewGlobfile reads globs from filename and returns a Globfile struct.
func NewGlobfile(filename string) (Globfile, error) {
	file, err := os.Open(filename)

	if err != nil {
		return Globfile{}, err
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)

	globs := []string{}

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		globs = append(globs, line)
	}

	err = scanner.Err()

	if err != nil {
		return Globfile{}, err
	}

	return Globfile{
		globs: globs,
	}, nil
}

// Len returns the number of parsed globs.
func (g *Globfile) Len() int {
	return len(g.globs)
}

// Matches checks if path matches one of the parsed globs.
func (g *Globfile) Matches(path string) bool {
	if len(path) == 0 {
		return false
	}

	for _, pattern := range g.globs {
		if glob.Glob(pattern, path) {
			return true
		}
	}

	return false
}
