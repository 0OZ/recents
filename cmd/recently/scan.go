package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const scanLinkSubdir = "recent"

type repoActivity struct {
	path  string
	mtime time.Time
}

// findGitRepos walks root and returns every directory containing a
// .git entry, annotated with a "last activity" timestamp. Walking does
// not descend into a repo once found, so submodules and vendored repos
// are ignored.
func findGitRepos(root string) ([]repoActivity, error) {
	var repos []repoActivity
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			dlog("walk %s: %v", path, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		gitPath := filepath.Join(path, gitMarkerDir)
		if _, statErr := os.Stat(gitPath); statErr == nil {
			mtime := lastActivity(gitPath)
			dlog("found repo %s (last activity %s)", path, mtime.Format(time.RFC3339))
			repos = append(repos, repoActivity{path: path, mtime: mtime})
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].mtime.After(repos[j].mtime)
	})
	return repos, nil
}

// lastActivity returns the most recent mtime we can observe for a
// repo's git state. .git/HEAD updates on commits and checkouts; the
// .git entry itself is a fallback for unusual layouts (worktrees,
// gitdir files).
func lastActivity(gitPath string) time.Time {
	headPath := filepath.Join(gitPath, "HEAD")
	if info, err := os.Stat(headPath); err == nil {
		return info.ModTime()
	} else {
		dlog("stat %s: %v", headPath, err)
	}
	if info, err := os.Stat(gitPath); err == nil {
		return info.ModTime()
	} else {
		dlog("stat %s: %v", gitPath, err)
	}
	return time.Time{}
}

func runScan(cfg Config, root string) {
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		os.Exit(1)
	}
	repos, err := findGitRepos(abs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		os.Exit(1)
	}
	if len(repos) == 0 {
		fmt.Printf("(no git repos under %s)\n", abs)
		return
	}

	keep := repos
	if len(keep) > cfg.MaxRecent {
		dlog("trimming %d repo(s) beyond max=%d", len(keep)-cfg.MaxRecent, cfg.MaxRecent)
		keep = keep[:cfg.MaxRecent]
	}

	entries := make([]Entry, len(keep))
	for i, r := range keep {
		entries[i] = Entry{Path: ProjectPath(r.path), LastVisit: r.mtime}
	}

	linkDir := filepath.Join(abs, scanLinkSubdir)
	if err := NewLinkManager(linkDir).Refresh(entries); err != nil {
		fmt.Fprintf(os.Stderr, "scan: refresh: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("scanned %s - found %d repo(s), linked top %d into %s\n",
		abs, len(repos), len(keep), linkDir)
}
