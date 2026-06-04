#!/usr/bin/bash
# start.sh - ExecStart for sirens-discord-ops.service. Run by systemd, not by hand.
# Pulls main, rebuilds, fetches SSM env, execs the binary. See README deploy notes.

set -euo pipefail

# Linuxbrew supplies go and coily.
if [ -x /home/linuxbrew/.linuxbrew/bin/brew ]; then
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
fi

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_DIR}"

# Fast-forward main to origin/main. ff-only fails loudly rather than
# clobber any rare local edits in the non-interactive systemd context.
git fetch --quiet origin main
git checkout --quiet main
git merge --ff-only --quiet origin/main

# Build in place. `go build` is incremental, so this is a no-op when the
# source hasn't changed.
mkdir -p bin
go build -o bin/sirens-discord-ops ./cmd/sirens-discord-ops

# Fetch SSM at exec time. No env file on disk: token rotation is
# `coily ops aws ssm put-parameter` followed by `systemctl restart`.

# fetch_ssm fails loud so a broken fetch dies here, not later as a confusing
# "invalid Authorization header". See docs/deploy.md for the two-guard rationale.
fetch_ssm() {
  local key="$1"
  # Split assignment from `local` so a non-zero fetch trips set -e (a combined
  # `local value=...` would mask it); the -z check catches an empty success.
  local value
  value="$(coily ops aws ssm get-parameter --name "/sirens-discord-ops/${key}" --with-decryption --query Parameter.Value --output text)"
  if [ -z "${value}" ]; then
    echo "FATAL: SSM fetch for /sirens-discord-ops/${key} returned empty" >&2
    exit 1
  fi
  printf '%s' "${value}"
}

DISCORD_TOKEN="$(fetch_ssm discord_token)"
ADMIN_CHANNEL_ID="$(fetch_ssm admin_channel_id)"
AUDIT_CHANNEL_ID="$(fetch_ssm audit_channel_id)"
ADMIN_ROLE_ID="$(fetch_ssm admin_role_id)"
export DISCORD_TOKEN ADMIN_CHANNEL_ID AUDIT_CHANNEL_ID ADMIN_ROLE_ID

exec "${REPO_DIR}/bin/sirens-discord-ops"
