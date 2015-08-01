package models

// File represents a file on the user's system. It consists of one or more chunks.
type File struct {
	ModificationTime JSONTime  `json:"m"`
	AddedAt          JSONTime  `json:"a"`
	DeletedAt        *JSONTime `json:"d,omitempty"`
	Size             uint64    `json:"s,omitempty"`
	IsDirectory      bool      `json:"i,omitempty"`
	Chunks           []Chunk   `json:"c,omitempty"`
}
