package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sendCommand dials the daemon, writes a single command line, and
// returns the trimmed reply.
func sendCommand(socket string, cmd string) (string, error) {
	conn, err := net.DialTimeout("unix", socket, dialTimeout)
	if err != nil {
		dlog("dial %s: %v", socket, err)
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(readTimeout))
	if _, err := fmt.Fprintln(conn, cmd); err != nil {
		dlog("write %s: %v", socket, err)
		return "", err
	}
	buf := make([]byte, responseBufferSize)
	n, err := conn.Read(buf)
	if err != nil {
		dlog("read %s: %v", socket, err)
	}
	return strings.TrimSpace(string(buf[:n])), nil
}

func runList(cfg Config) {
	store := NewStore(cfg.StorePath, cfg.MaxRecent)
	store.Load()
	entries := store.Snapshot()
	if len(entries) == 0 {
		fmt.Println("(no recent projects)")
		return
	}
	for i, e := range entries {
		age := time.Since(e.LastVisit).Round(time.Second)
		fmt.Printf("%d. %s  (%s ago)\n", i+1, e.Path, age)
	}
}

func runClear(cfg Config) {
	if _, err := sendCommand(cfg.SocketPath, string(cmdClear)); err == nil {
		fmt.Println("cleared")
		return
	}
	// Daemon not running: clean up on-disk state directly.
	_ = os.Remove(cfg.StorePath)
	existing, _ := os.ReadDir(cfg.LinkDir)
	for _, e := range existing {
		if e.Type()&os.ModeSymlink != 0 {
			_ = os.Remove(filepath.Join(cfg.LinkDir, e.Name()))
		}
	}
	fmt.Println("cleared (daemon not running)")
}

func runRecord(cfg Config, rawPath string) {
	_, _ = sendCommand(cfg.SocketPath, rawPath)
}
