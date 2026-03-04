package models

// Chunk represents a contiguous block of source code, usually a function or class.
// It is used in the interval map to identify the boundaries of a given signature.
type Chunk struct {
	File  string `json:"f"`     // The relative file path to the original source file.
	Start int    `json:"start"` // The 1-indexed start line of the chunk.
	End   int    `json:"end"`   // The 1-indexed end line of the chunk.
	Sig   string `json:"sig"`   // The signature of the chunk (e.g., "func process()").
}
