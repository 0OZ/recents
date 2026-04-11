# recently

**A live symlink trail of the git repos you actually work in.**

Every `cd` into a git repo drops a symlink in `~/.recently/current/`. The oldest falls off once you hit the cap. No history to dig through, no fuzzy finder to launch - just a directory you can `ls`, `tab`, or point your editor at.

Under the hood: a tiny Go daemon on a unix socket and a one-line shell hook that fires on `PWD` changes. That's the whole thing.

## install

```
bash <(curl -fsSL https://raw.githubusercontent.com/0OZ/recents/main/install.sh)
```

or clone and run:

```
git clone https://github.com/0OZ/recents recently
cd recently
./install.sh
```

Builds the binary into `~/.local/bin`, drops a hook into your shell (fish / bash / zsh), and registers the daemon with launchd on macOS or `systemd --user` on Linux.

## use

```
recently              # run the daemon
recently <path>       # record a path (what the shell hook calls)
recently -list        # show current entries
recently -clear       # wipe
recently -scan [dir]  # one-shot scan: walk dir (or cwd) for git repos, link recents into <dir>/recent/
recently -D …         # enable debug logging (alias: -debug)
recently -h           # all flags
```

## example - live tracking

```
$ cd ~/code/api-server
$ cd ~/code/web-ui
$ cd ~/notes            # not a git repo, ignored
$ cd ~/code/cli-tool

$ tree ~/.recently/current
~/.recently/current
├── cli-tool   -> ~/code/cli-tool
├── web-ui     -> ~/code/web-ui
└── api-server -> ~/code/api-server
```

Keep hopping between repos and the oldest entry drops once you hit `RECENTLY_MAX`.

## example - one-shot scan

Point `-scan` at a directory that holds many projects and it walks the tree,
ranks every git repo it finds by last activity (`.git/HEAD` mtime), and drops
symlinks to the top `RECENTLY_MAX` into `<dir>/recent/`.

```
$ recently -scan ~/code
scanned /Users/you/code - found 24 repo(s), linked top 9 into /Users/you/code/recent

$ tree ~/code/recent
~/code/recent
├── api-server      -> ~/code/backend/api-server
├── auth-service    -> ~/code/backend/auth-service
├── cli-tool        -> ~/code/tools/cli-tool
├── data-pipeline   -> ~/code/backend/data-pipeline
├── dotfiles        -> ~/code/dotfiles
├── mobile-app      -> ~/code/mobile/mobile-app
├── web-ui          -> ~/code/frontend/web-ui
├── worker-queue    -> ~/code/backend/worker-queue
└── www-site        -> ~/code/frontend/www-site
```

It's a one-shot snapshot - no daemon, no shell hook required. Re-run whenever
you want the list refreshed.

## debug

Add `-D` (or `-debug`) to any invocation to surface everything the program
would otherwise swallow - walk errors, stat failures, pruned store entries,
socket dial errors, trim decisions.

```
$ recently -D -scan ~/code
2026/04/11 10:01:02.903052 debug: debug logging enabled
2026/04/11 10:01:02.997345 debug: found repo ~/code/archived/old-proto (last activity 2024-03-17T21:45:53+01:00)
...
2026/04/11 10:01:03.292082 debug: trimming 15 repo(s) beyond max=9
scanned ~/code - found 24 repo(s), linked top 9 into ~/code/recent
```

## configure

Every setting has an env var and a matching `-flag`. Flags win over env vars.

```
-root     RECENTLY_ROOT       only track repos under this path prefix   default: everywhere
-max      RECENTLY_MAX        max number of recent entries              default 9
-link     RECENTLY_LINK_DIR   directory to write symlinks into          default ~/.recently/current
-store    RECENTLY_STORE      path to JSON store                        default ~/.recently/store.json
-socket   RECENTLY_SOCKET     unix socket path                          default /tmp/recently-<uid>.sock
```

Action flags (no env var):

```
-list             list current recent entries and exit
-clear            clear all recent entries and exit
-scan             scan a directory tree and link the most recent repos into <dir>/recent/
-debug, -D        enable debug logging
-version          print version and exit
-h                show all flags
```

## uninstall

```
./install.sh --uninstall
```
