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
export DISCORD_TOKEN="$(coily ops aws ssm get-parameter --name /sirens-discord-ops/discord_token    --with-decryption --query Parameter.Value --output text)"
export ADMIN_CHANNEL_ID="$(coily ops aws ssm get-parameter --name /sirens-discord-ops/admin_channel_id --with-decryption --query Parameter.Value --output text)"
export AUDIT_CHANNEL_ID="$(coily ops aws ssm get-parameter --name /sirens-discord-ops/audit_channel_id --with-decryption --query Parameter.Value --output text)"
export ADMIN_ROLE_ID="$(coily ops aws ssm get-parameter --name /sirens-discord-ops/admin_role_id     --with-decryption --query Parameter.Value --output text)"

exec "${REPO_DIR}/bin/sirens-discord-ops"
