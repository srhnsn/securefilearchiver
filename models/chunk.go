package models

// Chunk represents a part of a file.
type Chunk struct {
	Name string `json:"n"`
	Size uint64 `json:"s"`
}
