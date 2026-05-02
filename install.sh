#!/usr/bin/env bash
#
# recently - installer
# Usage:
#   ./install.sh              install binary, shell hook, background service
#   ./install.sh --uninstall  remove everything this script installed
#
# Environment:
#   INSTALL_DIR   where to put the binary (default: ~/.local/bin)
#   RECENTLY_*    passed through to the daemon at runtime

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BIN_PATH="$INSTALL_DIR/recently"
PLIST="$HOME/Library/LaunchAgents/com.recently.daemon.plist"
SYSTEMD_UNIT="$HOME/.config/systemd/user/recently.service"
FISH_HOOK="$HOME/.config/fish/conf.d/recently.fish"
HOOK_MARKER="# recently-hook"

OS="$(uname)"

info() { printf '\033[34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[33m!!>\033[0m %s\n' "$*"; }
die()  { printf '\033[31merr\033[0m %s\n' "$*" >&2; exit 1; }

detect_shell() {
    case "$(basename "${SHELL:-}")" in
        fish) echo fish ;;
        zsh)  echo zsh ;;
        bash) echo bash ;;
        *)    echo unknown ;;
    esac
}

install_hook_fish() {
    mkdir -p "$(dirname "$FISH_HOOK")"
    cat > "$FISH_HOOK" <<EOF
$HOOK_MARKER
function _recently_track --on-variable PWD
    command $BIN_PATH \$PWD >/dev/null 2>&1 &
    disown 2>/dev/null
end
EOF
    info "fish hook → $FISH_HOOK"
}

install_hook_rc() {
    local rc="$1" hook
    hook=$(cat <<EOF

$HOOK_MARKER
_recently_track() { command $BIN_PATH "\$PWD" >/dev/null 2>&1 & disown 2>/dev/null; }
EOF
)
    if grep -qF "$HOOK_MARKER" "$rc" 2>/dev/null; then
        info "shell hook already in $rc"
        return
    fi
    case "$(basename "$rc")" in
        .bashrc)
            printf '%s\nPROMPT_COMMAND="_recently_track${PROMPT_COMMAND:+;$PROMPT_COMMAND}"\n' "$hook" >> "$rc"
            ;;
        .zshrc)
            printf '%s\nautoload -U add-zsh-hook\nadd-zsh-hook chpwd _recently_track\n' "$hook" >> "$rc"
            ;;
    esac
    info "shell hook → $rc"
}

find_existing_plist_darwin() {
    local agents_dir="$HOME/Library/LaunchAgents"
    [ -d "$agents_dir" ] || return 0
    local p
    for p in "$agents_dir"/*.plist; do
        [ -f "$p" ] || continue
        [ "$p" = "$PLIST" ] && continue
        if grep -qE '<string>[^<]*/recently</string>' "$p" 2>/dev/null; then
            printf '%s\n' "$p"
            return 0
        fi
    done
}

install_service_darwin() {
    local existing
    existing="$(find_existing_plist_darwin)"
    if [ -n "$existing" ]; then
        local label
        label="$(basename "$existing" .plist)"
        warn "existing recently plist detected: $existing"
        warn "skipping creation of $PLIST to avoid duplicate daemon"
        warn "verify ProgramArguments in $existing points to $BIN_PATH"
        launchctl kickstart -k "gui/$(id -u)/$label" 2>/dev/null \
            && info "kicked existing daemon ($label)" \
            || warn "could not kick $label; restart it manually"
        return 0
    fi

    mkdir -p "$(dirname "$PLIST")"
    cat > "$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>com.recently.daemon</string>
    <key>ProgramArguments</key>
    <array><string>$BIN_PATH</string></array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
    <key>StandardOutPath</key><string>/tmp/recently.log</string>
    <key>StandardErrorPath</key><string>/tmp/recently.log</string>
</dict>
</plist>
EOF
    launchctl unload "$PLIST" 2>/dev/null || true
    launchctl load "$PLIST"
    info "launchd agent loaded"
}

install_service_linux() {
    mkdir -p "$(dirname "$SYSTEMD_UNIT")"
    cat > "$SYSTEMD_UNIT" <<EOF
[Unit]
Description=recently - track recently visited git projects

[Service]
Type=simple
ExecStart=$BIN_PATH
Restart=on-failure

[Install]
WantedBy=default.target
EOF
    systemctl --user daemon-reload
    systemctl --user enable --now recently.service
    info "systemd user service enabled"
}

uninstall() {
    info "Uninstalling recently..."

    if [ "$OS" = "Darwin" ] && [ -f "$PLIST" ]; then
        launchctl unload "$PLIST" 2>/dev/null || true
        rm -f "$PLIST"
        info "removed launchd agent"
    fi

    if [ "$OS" = "Linux" ] && [ -f "$SYSTEMD_UNIT" ]; then
        systemctl --user disable --now recently.service 2>/dev/null || true
        rm -f "$SYSTEMD_UNIT"
        systemctl --user daemon-reload 2>/dev/null || true
        info "removed systemd unit"
    fi

    rm -f "$BIN_PATH"
    rm -f "$FISH_HOOK"
    for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
        [ -f "$rc" ] || continue
        if grep -qF "$HOOK_MARKER" "$rc"; then
            warn "edit $rc manually to remove the block marked '$HOOK_MARKER'"
        fi
    done
    info "Done. Store at ~/.recently/ and symlinks were left in place."
}

if [ "${1:-}" = "--uninstall" ]; then
    uninstall
    exit 0
fi

command -v go >/dev/null 2>&1 || die "Go is required (https://go.dev/)"

info "Building binary → $BIN_PATH"
mkdir -p "$INSTALL_DIR"
(cd "$REPO_DIR" && go build -o "$BIN_PATH" ./cmd/recently)

sh="$(detect_shell)"
case "$sh" in
    fish) install_hook_fish ;;
    bash) install_hook_rc "$HOME/.bashrc" ;;
    zsh)  install_hook_rc "$HOME/.zshrc" ;;
    *)    warn "unknown shell; install a PWD-change hook manually" ;;
esac

case "$OS" in
    Darwin) install_service_darwin ;;
    Linux)  install_service_linux ;;
    *)      warn "unknown OS '$OS'; start the daemon manually: $BIN_PATH &" ;;
esac

cat <<EOF

Done! Restart your shell (or source the rc file) to activate the hook.

  recently -list      show recent projects
  recently -clear     clear the list
  recently -h         all flags

Symlinks default to ~/.recently/current/. Override with RECENTLY_LINK_DIR.
EOF
