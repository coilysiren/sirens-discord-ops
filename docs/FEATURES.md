# Features

Baseline of what `sirens-discord-ops` does today. Use this to evaluate scope creep or scope reduction over time. Generated 2026-05-08.

The repo is intentionally a thin Discord-button passthrough to the `coily` CLI, with audit logging. It is not a game-server admin system in its own right. Out-of-scope items (mod install, backups, log tailing, version pinning, web UI, REST API, dashboards) are called out explicitly in the README.

## Discord Bot Core

- Discord bot entry point that loads configuration and initializes the game registry.
- Role-based authorization gating for all admin commands via `ADMIN_ROLE_ID`.
- Ephemeral "not authorized" response for non-admin button clicks.
- Bot lifecycle opens a Discord session on `Run()` and blocks until SIGTERM or Interrupt.

## Control Panel UI

- Pinned control panel message posted to admin channel on bot startup.
- One panel per game (multi-game registry ready, v1 ships eco only).
- Message Components V2 rendering with semantic button colors (red for destructive, green for constructive, neutral for status).
- Large visual separators between button rows for mobile-friendly spacing.
- Auto-detection and migration of legacy Components V1 panels to V2 format.
- In-place panel editing on subsequent startups (no duplicate message accumulation).

## Button Interaction Flow

- Game-verb button pairs use custom interaction IDs in `sdo:<game>:<verb>` format.
- Two-step confirmation flow for destructive verbs (restart, stop, start).
- Ephemeral confirm/cancel prompts.
- Deferred interaction responses to operate beyond Discord's 3-second response window.

## Game Server Operations

- Eco verbs: restart, status, stop, start.
- Game registry extensible to other titles (Factorio in design, not implemented).
- Per-game verb lists (Eco's verbs are not hard-coded into the dispatcher).
- Per-game `coily` prefix mapping (e.g. `coily gaming eco restart`).
- All verbs shell out to the `coily` CLI. The bot has no direct server control.
- 5-minute timeout ceiling on each `coily` invocation.

## Audit Logging

- Dedicated audit channel records every button press start and completion.
- Actor mention (Discord user) included on each audit message.
- Full `coily` stdout and stderr captured and posted to the audit channel.
- 2000-character Discord limit handled with tail truncation plus a `(output truncated, see journalctl on kai-server)` marker.
- Command invocation line logged as-is.
- Exit code captured from each `coily` subprocess.

## Configuration

- Loads Discord token, admin channel ID, audit channel ID, and admin role ID from env.
- Optional `COILY_BIN` override (defaults to PATH lookup).
- Validates required env vars with clear missing-variable errors.
- Production secrets injected via SSM in `start.sh` (no env files on disk).

## Systemd Integration

- Main service unit `sirens-discord-ops.service` running as `kai`.
- Graceful shutdown via SIGTERM with context cancellation.
- Restart-on-failure with 5-second backoff, 5-burst-per-60-seconds limit.
- `network-online.target` dependency.

## Auto-Update

- Timer unit `sirens-discord-ops-update.timer` polls every 5 minutes.
- Lightweight `git fetch` plus revision comparison, no unnecessary restarts.
- Restarts the main service when `origin/main` advances.
- 2-minute initial delay to avoid early-boot churn.
- 30-second accuracy window on timer precision.
- SSM parameters re-fetched on every start, supporting credential rotation without re-running `install.sh`.

## Deployment

- `start.sh` handles fast-forward checkout via detached `git checkout origin/main` (no merge prompts).
- Incremental `go build` (no-op when source unchanged).
- One-time bootstrap script `install.sh` places systemd units and sudoers config.

## Sudoers

- NOPASSWD rules for auto-update timer and `coily` diagnostic commands.
- Explicit command-line argument matching for systemctl invocations.
- Covers both `/bin/systemctl` and `/usr/bin/systemctl` paths.

## Build and Dev Loop

- Makefile targets: `build`, `vet`, `test`, `tidy`, `run`.
- `bin/sirens-discord-ops` build output.
- `make run` against the Sirens Echo test server.

## Code Quality

- Pre-commit hooks: trailing whitespace, EOF fixer, YAML validation, large-file and merge-conflict detection, mixed line endings.
- Go-specific: `go mod tidy`, `go vet`, trufflehog secret scanning.
- Commit-message hook requires every commit to close a same-repo GitHub issue.
- Exempt commit prefixes: `Merge`, `Revert`, `fixup!`, `squash!`.

## Process Architecture

- One goroutine per verb invocation, `coily` runs in the background.
- No in-process concurrency protection. The `coily` layer is the safety net for simultaneous admin button presses.
- No persistent state across restarts. Panel detection works by scanning Discord messages, no database.
