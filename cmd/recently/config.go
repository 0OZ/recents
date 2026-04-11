package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config is the resolved runtime configuration for a recently process.
type Config struct {
	Root       string
	MaxRecent  int
	LinkDir    string
	StorePath  string
	SocketPath string
}

func defaultConfig() Config {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, defaultBaseDirName)
	return Config{
		Root:       envRoot.lookup(),
		MaxRecent:  envInt(envMax, defaultMaxRecent),
		LinkDir:    envOrDefault(envLinkDir, filepath.Join(base, defaultLinkSubdir)),
		StorePath:  envOrDefault(envStore, filepath.Join(base, defaultStoreFilename)),
		SocketPath: envOrDefault(envSocket, fmt.Sprintf(socketPathTemplate, os.Getuid())),
	}
}

func envOrDefault(e envVar, def string) string {
	if v := e.lookup(); v != "" {
		return v
	}
	return def
}

func envInt(e envVar, def int) int {
	if v := e.lookup(); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
