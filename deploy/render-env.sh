#!/usr/bin/env bash
# Render /etc/sirens-discord-ops.env from SSM. Run on kai-server as root
# during deploy. Reads SecureStrings via the host's IAM role.
#
# Idempotent: rewrites the file in place. The systemd unit reads the file
# via EnvironmentFile=, so a unit restart picks up changes.
set -euo pipefail

OUT=/etc/sirens-discord-ops.env
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

fetch() {
  aws ssm get-parameter \
    --name "$1" \
    --with-decryption \
    --query Parameter.Value \
    --output text
}

cat >"$TMP" <<EOF
DISCORD_TOKEN=$(fetch /sirens-discord-ops/discord_token)
ADMIN_CHANNEL_ID=$(fetch /sirens-discord-ops/admin_channel_id)
AUDIT_CHANNEL_ID=$(fetch /sirens-discord-ops/audit_channel_id)
ADMIN_ROLE_ID=$(fetch /sirens-discord-ops/admin_role_id)
EOF

install -m 0600 -o root -g root "$TMP" "$OUT"
echo "wrote $OUT"
