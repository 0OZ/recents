package main

import (
	"log"
	"os"
)

var debugEnabled bool

func enableDebug() {
	debugEnabled = true
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	dlog("debug logging enabled")
}

// dlog emits a log line when -debug / -D is set. No-op otherwise so
// hot paths stay cheap in the common case.
func dlog(format string, args ...any) {
	if !debugEnabled {
		return
	}
	log.Printf("debug: "+format, args...)
}
