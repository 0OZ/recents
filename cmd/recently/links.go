package main

import (
	"os"
	"path/filepath"
)

// LinkManager owns the directory of symlinks that mirror the current
// recent-projects list.
type LinkManager struct {
	dir string
}

func NewLinkManager(dir string) *LinkManager {
	return &LinkManager{dir: dir}
}

// Refresh rewrites the link directory so it mirrors the given entries.
// Safe to call without holding the store mutex.
func (l *LinkManager) Refresh(entries []Entry) error {
	if err := os.MkdirAll(l.dir, dirMode); err != nil {
		return err
	}
	existing, err := os.ReadDir(l.dir)
	if err != nil {
		return err
	}
	for _, e := range existing {
		if e.Type()&os.ModeSymlink != 0 {
			_ = os.Remove(filepath.Join(l.dir, e.Name()))
		}
	}
	seen := map[string]bool{}
	for _, e := range entries {
		name := l.pickName(e.Path, seen)
		seen[name] = true
		if err := os.Symlink(string(e.Path), filepath.Join(l.dir, name)); err != nil {
			return err
		}
	}
	return nil
}

// pickName chooses a unique symlink name for p. If the bare base name
// collides, we prepend ancestor directory names one level at a time
// ("parent-base", "grandparent-parent-base", ...) until we find a free
// slot or run out of ancestors.
func (l *LinkManager) pickName(p ProjectPath, seen map[string]bool) string {
	name := p.Base()
	if !seen[name] {
		return name
	}
	dir := filepath.Dir(string(p))
	for dir != "/" && dir != "." {
		name = filepath.Base(dir) + "-" + name
		if !seen[name] {
			return name
		}
		dir = filepath.Dir(dir)
	}
	return name
}
