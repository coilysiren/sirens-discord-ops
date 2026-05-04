.PHONY: build vet test tidy run

BIN := bin/sirens-discord-ops

build:
	@mkdir -p bin
	go build -o $(BIN) ./cmd/sirens-discord-ops

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy

# Local dev run against Sirens Echo. Export DISCORD_TOKEN, ADMIN_CHANNEL_ID,
# AUDIT_CHANNEL_ID, ADMIN_ROLE_ID first (or wrap in a direnv .envrc).
# Production deploy is `git push` here, then `git pull && bash scripts/install.sh`
# on kai-server.
run:
	go run ./cmd/sirens-discord-ops
