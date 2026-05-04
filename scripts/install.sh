#!/usr/bin/bash
# install.sh - one-time bootstrap of sirens-discord-ops on kai-server.
#
# Drops the unit files and sudoers fragment into place, then enables the
# main service + the auto-update timer. After this runs once, ongoing
# updates flow via:
#
#   workstation: git push
#   kai-server:  the timer fires within 5min, sees origin/main has moved,
#                and restarts the unit. start.sh pulls + builds + execs.
#
# Re-running this script is safe (idempotent) but only needed when the
# unit files or sudoers fragment in this repo change. Code-only updates
# do not require re-running install.
#
# Run as the `kai` user from the repo checkout. Sudo is invoked per-step.

set -euo pipefail

if [ -x /home/linuxbrew/.linuxbrew/bin/brew ]; then
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
fi

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> sudoers"
sudo install -m 0440 -o root -g root \
  "${REPO_DIR}/sudoers/kai-sirens-discord-ops" \
  /etc/sudoers.d/kai-sirens-discord-ops
sudo visudo -cf /etc/sudoers.d/kai-sirens-discord-ops

echo "==> systemd unit files"
sudo install -m 0644 "${REPO_DIR}/systemd/sirens-discord-ops.service"        /etc/systemd/system/
sudo install -m 0644 "${REPO_DIR}/systemd/sirens-discord-ops-update.service" /etc/systemd/system/
sudo install -m 0644 "${REPO_DIR}/systemd/sirens-discord-ops-update.timer"   /etc/systemd/system/

echo "==> daemon-reload + enable --now"
sudo systemctl daemon-reload
sudo systemctl enable --now sirens-discord-ops.service
sudo systemctl enable --now sirens-discord-ops-update.timer

echo
echo "==> status"
sudo systemctl --no-pager status sirens-discord-ops.service | head -10 || true
echo
sudo systemctl --no-pager status sirens-discord-ops-update.timer | head -10 || true
echo
echo "Verify with:"
echo "  sudo journalctl -u sirens-discord-ops -n 20 --no-pager -f"
echo "  sudo systemctl list-timers sirens-discord-ops-update.timer"
