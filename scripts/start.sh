#!/usr/bin/bash
# start.sh - ExecStart for sirens-discord-ops.service.
#
# Pulls the latest main, rebuilds the binary, fetches SSM-backed env vars,
# and execs into the binary as PID 1. Run by systemd; not intended to be
# run by hand.
#
# To update the bot in production:
#     workstation: git push
#     kai-server:  sudo systemctl restart sirens-discord-ops

set -euo pipefail

# Linuxbrew supplies go and coily.
if [ -x /home/linuxbrew/.linuxbrew/bin/brew ]; then
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
fi

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_DIR}"

# Fast-forward main to origin/main. ff-only avoids merge prompts in
# non-interactive systemd context. Local edits on kai-server should be
# rare; if any exist, fail loudly rather than clobber.
git fetch --quiet origin main
git checkout --quiet main
git merge --ff-only --quiet origin/main

# Build in place. `go build` is incremental, so this is a no-op when the
# source hasn't changed.
mkdir -p bin
go build -o bin/sirens-discord-ops ./cmd/sirens-discord-ops

# Fetch SSM at exec time. No env file on disk: token rotation is
# `coily aws ssm put-parameter` followed by `systemctl restart`.
export DISCORD_TOKEN="$(coily aws ssm get-parameter --name /sirens-discord-ops/discord_token    --with-decryption --query Parameter.Value --output text)"
export ADMIN_CHANNEL_ID="$(coily aws ssm get-parameter --name /sirens-discord-ops/admin_channel_id --with-decryption --query Parameter.Value --output text)"
export AUDIT_CHANNEL_ID="$(coily aws ssm get-parameter --name /sirens-discord-ops/audit_channel_id --with-decryption --query Parameter.Value --output text)"
export ADMIN_ROLE_ID="$(coily aws ssm get-parameter --name /sirens-discord-ops/admin_role_id     --with-decryption --query Parameter.Value --output text)"

exec "${REPO_DIR}/bin/sirens-discord-ops"
