#!/usr/bin/bash
# auto-update.sh - poll origin/main and `systemctl restart sirens-discord-ops`
# only when the remote has moved. Run by sirens-discord-ops-update.timer.
#
# The actual git pull happens inside start.sh on restart, so this script
# only decides whether a restart is warranted. Idempotent: a no-op when
# origin hasn't moved.

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_DIR}"

git fetch --quiet origin main

LOCAL="$(git rev-parse HEAD)"
REMOTE="$(git rev-parse origin/main)"

if [ "${LOCAL}" = "${REMOTE}" ]; then
  exit 0
fi

echo "auto-update: ${LOCAL:0:8} -> ${REMOTE:0:8}, restarting"
sudo systemctl restart sirens-discord-ops
