#!/usr/bin/bash
# install.sh - bring up (or refresh) sirens-discord-ops on kai-server.
# Idempotent: safe to re-run after pulls that change the bot, the unit
# file, or the SSM-backed env values.
#
# Workflow:
#   workstation: git push
#   kai-server:  cd ~/projects/coilysiren/sirens-discord-ops && git pull && bash scripts/install.sh
#
# Prereqs:
#   - go on PATH (Linuxbrew: `brew install go`).
#   - coily on PATH (already installed via scripts/install-coily.sh in
#     coilysiren/infrastructure).
#   - The four /sirens-discord-ops/* SSM params already exist.
#
# Run as the `kai` user from the repo checkout. Sudo is invoked per-step.

set -euo pipefail

# Source brew's shellenv when running non-interactively so coily and go
# (both installed via Linuxbrew) land on PATH.
if [ -x /home/linuxbrew/.linuxbrew/bin/brew ]; then
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
fi

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="/etc/sirens-discord-ops.env"
BIN_PATH="/usr/local/bin/sirens-discord-ops"
UNIT_FILE="/etc/systemd/system/sirens-discord-ops.service"

echo "==> go build"
cd "${REPO_DIR}"
mkdir -p bin
go build -o bin/sirens-discord-ops ./cmd/sirens-discord-ops

echo "==> install binary + unit"
sudo install -m 0755 "${REPO_DIR}/bin/sirens-discord-ops" "${BIN_PATH}"
sudo install -m 0644 "${REPO_DIR}/systemd/sirens-discord-ops.service" "${UNIT_FILE}"

echo "==> ${ENV_FILE} (from SSM)"
# Render via a tmpfile so a partial fetch never leaves a half-written env
# in place. Permissions land at 0600 root:root before the rename.
TOKEN_VAL="$(coily aws ssm get-parameter --name /sirens-discord-ops/discord_token    --with-decryption --query Parameter.Value --output text)"
ADMIN_CH="$(coily aws ssm get-parameter --name /sirens-discord-ops/admin_channel_id --with-decryption --query Parameter.Value --output text)"
AUDIT_CH="$(coily aws ssm get-parameter --name /sirens-discord-ops/audit_channel_id --with-decryption --query Parameter.Value --output text)"
ADMIN_ROLE="$(coily aws ssm get-parameter --name /sirens-discord-ops/admin_role_id  --with-decryption --query Parameter.Value --output text)"
TMP="$(sudo mktemp /etc/sirens-discord-ops.env.XXXXXX)"
sudo chmod 0600 "${TMP}"
sudo tee "${TMP}" >/dev/null <<EOF
DISCORD_TOKEN=${TOKEN_VAL}
ADMIN_CHANNEL_ID=${ADMIN_CH}
AUDIT_CHANNEL_ID=${AUDIT_CH}
ADMIN_ROLE_ID=${ADMIN_ROLE}
EOF
sudo mv "${TMP}" "${ENV_FILE}"
unset TOKEN_VAL ADMIN_CH AUDIT_CH ADMIN_ROLE

echo "==> systemd: daemon-reload + enable --now (restart if already running)"
sudo systemctl daemon-reload
sudo systemctl enable sirens-discord-ops
sudo systemctl restart sirens-discord-ops

echo
echo "==> first-run status"
sudo systemctl --no-pager status sirens-discord-ops | head -15 || true
echo
echo "Verify with:"
echo "  sudo journalctl -u sirens-discord-ops -n 20 --no-pager -f"
