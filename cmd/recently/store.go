package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store is the persisted, thread-safe collection of project entries.
// Mutation methods return a snapshot so callers can persist and refresh
// side effects outside the critical section.
type Store struct {
	path      string
	maxRecent int

	mu      sync.Mutex
	entries []Entry
}

func NewStore(path string, maxRecent int) *Store {
	return &Store{path: path, maxRecent: maxRecent}
}

// Load reads entries from disk. Entries whose path is missing (ENOENT)
// are pruned; entries that fail stat for other reasons (permission,
// unmounted volume, ...) are kept so a transient filesystem hiccup
// doesn't wipe the store.
func (s *Store) Load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			dlog("store read %s: %v", s.path, err)
		}
		return
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		dlog("store unmarshal %s: %v", s.path, err)
		return
	}
	kept := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if isAgentWorktree(string(e.Path)) {
			dlog("store prune agent worktree %s", e.Path)
			continue
		}
		if _, err := os.Stat(string(e.Path)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				dlog("store prune missing %s", e.Path)
				continue
			}
			dlog("store stat %s: %v", e.Path, err)
		}
		kept = append(kept, e)
	}
	s.mu.Lock()
	s.entries = kept
	s.mu.Unlock()
}

// Snapshot returns a defensive copy of the current entries.
func (s *Store) Snapshot() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *Store) snapshotLocked() []Entry {
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Record moves p to the front of the entries, truncates to maxRecent,
// and returns the resulting snapshot.
func (s *Store) Record(p ProjectPath) []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.entries[:0]
	for _, e := range s.entries {
		if e.Path != p {
			kept = append(kept, e)
		}
	}
	s.entries = append([]Entry{{Path: p, LastVisit: time.Now()}}, kept...)
	if len(s.entries) > s.maxRecent {
		s.entries = s.entries[:s.maxRecent]
	}
	return s.snapshotLocked()
}

// Clear removes all entries and returns the resulting (empty) snapshot.
func (s *Store) Clear() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
	return []Entry{}
}

// Persist writes the given snapshot to disk. The caller must not hold
// the store mutex while calling this.
func (s *Store) Persist(snap []Entry) error {
	if err := os.MkdirAll(filepath.Dir(s.path), dirMode); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, fileMode)
}
