package models

import (
	"sort"
)

// Document represents the index which stores all metadata about the archived files.
type Document struct {
	KeyEncrypted   string            `json:"key"`
	KeyUnencrypted string            `json:"-"`
	Files          map[string]File   `json:"files"`
	DeletedFiles   map[string][]File `json:"deleted_files"`
}

// GetSortedFilesKeys returns sorted Document.Files keys.
func (doc *Document) GetSortedFilesKeys() []string {
	result := []string{}

	for key := range doc.Files {
		result = append(result, key)
	}

	sort.Strings(result)
	return result
}
