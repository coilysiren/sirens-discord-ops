# sirens-discord-ops

Discord-native admin control panel for the Sirens game servers. v1 wires up
eco; the per-game registration shape leaves room for factorio and friends.

## What it does

The bot pins one message per game in `#admin-control` with one button per
verb. Pressing a button shells out to `coily gaming <game> <verb>` on
kai-server and posts the start + completion lines to `#admin-audit`,
verbatim coily output included. Coily is the API. The bot has no awareness
of how coily implements those verbs.

v1 ships with eco only:

- Restart
- Status
- Stop
- Start

## Auth

Every button press checks the actor's role membership against
`ADMIN_ROLE_ID`. No per-verb gating in v1. Non-admins get an ephemeral
"not authorized" and no audit-channel write.

## Concurrency

If two admins press Restart at once, both invocations run. Coily's
underlying behavior is the safety net. Documented as known-acceptable.

## Configuration

All four required values live in SSM under `/sirens-discord-ops/` and are
loaded into the systemd unit via `EnvironmentFile=/etc/sirens-discord-ops.env`:

| Env var            | SSM path                                  |
|--------------------|-------------------------------------------|
| `DISCORD_TOKEN`    | `/sirens-discord-ops/discord_token`       |
| `ADMIN_CHANNEL_ID` | `/sirens-discord-ops/admin_channel_id`    |
| `AUDIT_CHANNEL_ID` | `/sirens-discord-ops/audit_channel_id`    |
| `ADMIN_ROLE_ID`    | `/sirens-discord-ops/admin_role_id`       |

Optional:

- `COILY_BIN` - path to the coily binary. Defaults to `coily` (PATH lookup).

The `scripts/install.sh` script writes `/etc/sirens-discord-ops.env` from
SSM as part of the deploy. See "Production deploy" below.

## Local development

Point at Sirens Echo (the test server), export the four env vars, and run:

```sh
make run
```

The bot will post a pinned control panel in the configured admin channel on
first connect. Subsequent runs reuse and edit the existing pinned message
in place, so the verb list stays current without duplicate panels.

## Production deploy

Native systemd unit on kai-server (not k3s). The bot shells out to `coily`
directly, so it runs as the same user that scripts/install-coily.sh
configured (`kai`).

git-push / git-pull workflow, mirroring coilysiren/infrastructure:

```sh
# Workstation
git push

# kai-server
cd ~/projects/coilysiren/sirens-discord-ops
git pull
bash scripts/install.sh
```

The script builds the binary in place (Go via Linuxbrew), installs the
binary and unit file, renders `/etc/sirens-discord-ops.env` from SSM, and
restarts the unit. Idempotent: re-run after every pull that touches the
bot, the unit file, or the SSM-backed env.

## Adding a new game

Edit `internal/bot/games.go`, append a `Game{}` entry, redeploy. The bot will post a new
pinned panel for the new game on next start. Channels and admin role stay
the same. Per-game verb lists keep new games from being locked into eco's
exact four-verb mold.

## Out of scope (v1)

Anything outside `coily gaming eco {restart,status,stop,start}`: mod
install/uninstall, save backup/restore, log tailing, in-game player count,
version pinning, whitelist edits. Anything beyond eco. Concurrency
protection. Tighter coily integration (in-process call instead of
`exec.Command`). Web UI, mobile, REST API, observability dashboards.
