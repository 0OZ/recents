package main

import (
	"path/filepath"
	"strings"
	"time"
)

// ProjectPath is an absolute filesystem path to a git repository root.
type ProjectPath string

func (p ProjectPath) String() string { return string(p) }
func (p ProjectPath) Base() string   { return filepath.Base(string(p)) }

// IsUnder reports whether p lives inside the given root prefix.
// An empty root matches any path. Uses filepath.Rel to avoid the
// "/home/me" vs "/home/me-evil" string-prefix trap.
func (p ProjectPath) IsUnder(root string) bool {
	if root == "" {
		return true
	}
	rel, err := filepath.Rel(root, string(p))
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// Entry is a recorded visit to a project.
type Entry struct {
	Path      ProjectPath `json:"path"`
	LastVisit time.Time   `json:"last_visit"`
}
