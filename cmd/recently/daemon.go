package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Daemon is the long-lived application service: it owns the store and
// link manager and speaks the unix-socket protocol.
type Daemon struct {
	cfg   Config
	store *Store
	links *LinkManager
	sem   chan struct{}
}

func NewDaemon(cfg Config) *Daemon {
	return &Daemon{
		cfg:   cfg,
		store: NewStore(cfg.StorePath, cfg.MaxRecent),
		links: NewLinkManager(cfg.LinkDir),
		sem:   make(chan struct{}, maxInFlightConns),
	}
}

// Run binds the unix socket and serves requests until SIGINT/SIGTERM.
func (d *Daemon) Run() error {
	if c, err := net.DialTimeout("unix", d.cfg.SocketPath, daemonProbeTimeout); err == nil {
		c.Close()
		return fmt.Errorf("another %s daemon is already running on %s", appName, d.cfg.SocketPath)
	}

	d.store.Load()
	if err := d.links.Refresh(d.store.Snapshot()); err != nil {
		log.Printf("%s: initial link refresh: %v", appName, err)
	}

	_ = os.Remove(d.cfg.SocketPath)
	ln, err := net.Listen("unix", d.cfg.SocketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := os.Chmod(d.cfg.SocketPath, fileMode); err != nil {
		_ = ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		_ = ln.Close()
	}()

	log.Printf("%s: listening on %s (link=%s)", appName, d.cfg.SocketPath, d.cfg.LinkDir)
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				_ = os.Remove(d.cfg.SocketPath)
				return nil
			}
			return err
		}
		d.sem <- struct{}{}
		go func() {
			defer func() { <-d.sem }()
			d.handle(conn)
		}()
	}
}

func (d *Daemon) handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(handleDeadline))
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		if Command(raw) == cmdClear {
			d.clear()
			fmt.Fprintln(conn, respCleared)
			continue
		}
		if resp, ok := d.recordFromRaw(raw); ok {
			fmt.Fprintln(conn, resp)
		}
	}
}

// recordFromRaw resolves a raw client-supplied path to a git root,
// validates it against the configured Root prefix, and records it.
func (d *Daemon) recordFromRaw(raw string) (Response, bool) {
	dir, err := filepath.EvalSymlinks(raw)
	if err != nil {
		dir = raw
	}
	root, ok := findGitRoot(dir)
	if !ok {
		return "", false
	}
	p := ProjectPath(root)
	if !p.IsUnder(d.cfg.Root) {
		return "", false
	}
	snap := d.store.Record(p)
	d.persistAndRefresh(snap)
	return respRecorded(p), true
}

func (d *Daemon) clear() {
	snap := d.store.Clear()
	d.persistAndRefresh(snap)
}

// persistAndRefresh runs the two side effects (disk write + symlink
// rewrite) that every store mutation needs. Called without the store
// mutex held so slow I/O doesn't block other connections.
func (d *Daemon) persistAndRefresh(snap []Entry) {
	if err := d.store.Persist(snap); err != nil {
		log.Printf("%s: persist: %v", appName, err)
	}
	if err := d.links.Refresh(snap); err != nil {
		log.Printf("%s: refresh: %v", appName, err)
	}
}

// findGitRoot walks dir upwards until it finds a .git marker.
func findGitRoot(dir string) (string, bool) {
	for {
		if _, err := os.Stat(filepath.Join(dir, gitMarkerDir)); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
