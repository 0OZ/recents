package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	cfg := defaultConfig()

	var (
		rootFlag    = flag.String("root", cfg.Root, "only track repos under this path prefix")
		maxFlag     = flag.Int("max", cfg.MaxRecent, "max number of recent entries")
		linkFlag    = flag.String("link", cfg.LinkDir, "directory to write symlinks into")
		storeFlag   = flag.String("store", cfg.StorePath, "path to JSON store")
		socketFlag  = flag.String("socket", cfg.SocketPath, "unix socket path")
		listFlag    = flag.Bool("list", false, "list current recent entries and exit")
		clearFlag   = flag.Bool("clear", false, "clear all recent entries and exit")
		scanFlag    = flag.Bool("scan", false, "scan a directory tree for git repos and link the most recently used into <dir>/recent/")
		versionFlag = flag.Bool("version", false, "print version and exit")
	)
	var debugFlag bool
	flag.BoolVar(&debugFlag, "debug", false, "enable debug logging")
	flag.BoolVar(&debugFlag, "D", false, "enable debug logging (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s %s - track recently visited git projects\n\n", appName, appVersion)
		fmt.Fprintln(os.Stderr, "usage:")
		fmt.Fprintf(os.Stderr, "  %s                  run daemon\n", appName)
		fmt.Fprintf(os.Stderr, "  %s <path>           record path as visited\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -list            list recent entries\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -clear           clear all entries\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -scan [dir]      scan dir (or cwd) for git repos, link recents into <dir>/recent/\n", appName)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "flags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if debugFlag {
		enableDebug()
	}

	cfg.Root = *rootFlag
	cfg.MaxRecent = *maxFlag
	cfg.LinkDir = *linkFlag
	cfg.StorePath = *storeFlag
	cfg.SocketPath = *socketFlag

	switch {
	case *versionFlag:
		fmt.Println(appName, appVersion)
	case *listFlag:
		runList(cfg)
	case *clearFlag:
		runClear(cfg)
	case *scanFlag:
		target := "."
		if flag.NArg() > 0 {
			target = flag.Arg(0)
		}
		runScan(cfg, target)
	case flag.NArg() > 0:
		runRecord(cfg, flag.Arg(0))
	default:
		if err := NewDaemon(cfg).Run(); err != nil {
			log.Fatal(err)
		}
	}
}
