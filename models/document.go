package models

// Document represents the index which stores all metadata about the archived files.
type Document struct {
	KeyEncrypted   string            `json:"key"`
	KeyUnencrypted string            `json:"-"`
	Files          map[string]File   `json:"files"`
	DeletedFiles   map[string][]File `json:"deleted_files"`
}
