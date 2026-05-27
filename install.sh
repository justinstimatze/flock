#!/usr/bin/env bash
# install.sh — build flock and install it as a systemd *user* service (no root).
# Usage: ./install.sh [--uninstall]
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"
BIN="$BIN_DIR/flock"
UNIT_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
UNIT="$UNIT_DIR/flock.service"

if [[ "${1:-}" == "--uninstall" ]]; then
	systemctl --user disable --now flock.service 2>/dev/null || true
	rm -f "$UNIT" "$BIN"
	systemctl --user daemon-reload || true
	echo "flock: uninstalled."
	echo "note: your 'input' group membership was NOT changed. That grant lets all"
	echo "      your processes read the keyboard. To revoke it:"
	echo "          sudo gpasswd -d \"$USER\" input"
	exit 0
fi

# preflight
command -v go >/dev/null || { echo "flock: Go toolchain not found (needed to build)." >&2; exit 1; }
if ! id -nG "$USER" | tr ' ' '\n' | grep -qx input; then
	echo "flock: you are not in the 'input' group. Run, then log out and back in:" >&2
	echo "    sudo usermod -aG input \"$USER\"" >&2
	exit 1
fi
if ! { command -v pactl >/dev/null && pactl info >/dev/null 2>&1; }; then
	echo "flock: warning — no PulseAudio/PipeWire server detected; flock will be" >&2
	echo "       silent until one is running (stock desktops have one by default)." >&2
fi

# build + install the binary
echo "flock: building..."
(cd "$REPO" && go build -o "$REPO/flock" .)
mkdir -p "$BIN_DIR"
install -m 0755 "$REPO/flock" "$BIN"

# install the user service, pointing at the installed binary. awk gsub (not a
# sed s|| substitution) so path metacharacters cannot corrupt the unit file.
mkdir -p "$UNIT_DIR"
awk -v bin="$BIN" '{ gsub(/__FLOCK_BIN__/, bin); print }' "$REPO/systemd/flock.service" >"$UNIT"
systemctl --user daemon-reload
systemctl --user enable --now flock.service

echo "flock: installed and honking. 🪿"
echo "  binary:  $BIN"
echo "  status:  systemctl --user status flock.service"
echo "  stop:    systemctl --user stop flock.service"
echo "  remove:  $REPO/install.sh --uninstall"
