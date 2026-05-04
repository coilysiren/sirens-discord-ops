.PHONY: build vet test tidy run install

BIN := bin/sirens-discord-ops

build:
	@mkdir -p bin
	go build -o $(BIN) .

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy

# Local dev run against Sirens Echo. Export DISCORD_TOKEN, ADMIN_CHANNEL_ID,
# AUDIT_CHANNEL_ID, ADMIN_ROLE_ID first (or wrap in a direnv .envrc).
run:
	go run .

# Cross-compile for kai-server (linux/amd64) and stage to /usr/local/bin.
# Run from a workstation; assumes ssh access. Adjust SSH target as needed.
install: build
	GOOS=linux GOARCH=amd64 go build -o $(BIN) .
	scp $(BIN) kai-server:/tmp/sirens-discord-ops
	ssh kai-server 'sudo install -m 0755 /tmp/sirens-discord-ops /usr/local/bin/sirens-discord-ops && sudo systemctl restart sirens-discord-ops'
