package models

// File represents a file on the user's system. It consists of one or more chunks.
type File struct {
	ModificationTime JSONTime  `json:"m"`
	Size             uint64    `json:"s,omitempty"`
	DeletedAt        *JSONTime `json:"d,omitempty"`
	IsDirectory      bool      `json:"i,omitempty"`
	Chunks           []Chunk   `json:"c,omitempty"`
}
