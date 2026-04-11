package main

import (
	"os"
	"time"
)

const (
	appName    = "recently"
	appVersion = "0.1.0"
)

const (
	defaultMaxRecent     = 9
	defaultBaseDirName   = ".recently"
	defaultLinkSubdir    = "current"
	defaultStoreFilename = "store.json"
	socketPathTemplate   = "/tmp/recently-%d.sock"
)

const gitMarkerDir = ".git"

const (
	dirMode  os.FileMode = 0o755
	fileMode os.FileMode = 0o600
)

const (
	handleDeadline     = 2 * time.Second
	dialTimeout        = 300 * time.Millisecond
	readTimeout        = 500 * time.Millisecond
	daemonProbeTimeout = 100 * time.Millisecond
	responseBufferSize = 512
)

const maxInFlightConns = 32

type envVar string

const (
	envRoot    envVar = "RECENTLY_ROOT"
	envMax     envVar = "RECENTLY_MAX"
	envLinkDir envVar = "RECENTLY_LINK_DIR"
	envStore   envVar = "RECENTLY_STORE"
	envSocket  envVar = "RECENTLY_SOCKET"
)

func (e envVar) lookup() string { return os.Getenv(string(e)) }
