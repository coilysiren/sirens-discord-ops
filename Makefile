.PHONY: build vet test tidy run

BIN := bin/sirens-discord-ops

build: ## Compile the binary into bin/sirens-discord-ops.
	@mkdir -p bin
	go build -o $(BIN) ./cmd/sirens-discord-ops

vet: ## go vet across the tree.
	go vet ./...

test: ## Run the unit test suite.
	go test ./...

tidy: ## go mod tidy.
	go mod tidy

# Local dev run against Sirens Echo. Export DISCORD_TOKEN, ADMIN_CHANNEL_ID,
# AUDIT_CHANNEL_ID, ADMIN_ROLE_ID first (or wrap in a direnv .envrc).
# Production deploy is `git push` here, then `git pull && bash scripts/install.sh`
# on kai-server.
run: ## Local dev run against Sirens Echo. Requires DISCORD_TOKEN + channel/role env vars.
	go run ./cmd/sirens-discord-ops
