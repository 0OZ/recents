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

// isAgentWorktree reports whether path lives inside a Claude Code agent
// worktree (`.claude/worktrees/...`). These are throwaway sibling repos
// that would otherwise crowd out real projects in the MRU list.
func isAgentWorktree(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == ".claude" && parts[i+1] == "worktrees" {
			return true
		}
	}
	return false
}
