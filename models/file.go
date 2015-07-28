package models

// File represents a file on the user's system. It consists of one or more chunks.
type File struct {
	ModificationTime JSONTime `json:"m"`
	Size             uint64   `json:"s"`
	DeletedAt        JSONTime `json:"d"`

	Chunks []Chunk `json:"c"`
}
