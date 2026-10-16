SHELL := /bin/sh

WEB_DIR := apps/web
API_DIR := apps/api
DIST_DIR := $(API_DIR)/internal/webui/dist

.PHONY: dev-db dev-api dev-web migrate sqlc test build build-web docker backup restore prod-up prod-down webhook telegram-info

dev-db:
	docker compose up -d postgres

dev-api:
	cd $(API_DIR) && go run ./cmd/server

dev-web:
	cd $(WEB_DIR) && NODE_ENV=development npm run dev

migrate:
	cd $(API_DIR) && go run ./cmd/migrate

sqlc:
	# Use the official image so sqlc does not have to be installed locally.
	docker run --rm -v "$(CURDIR)/$(API_DIR):/src" -w /src sqlc/sqlc generate

# The auth and agent tests both migrate the same database; -p 1 runs one package
# at a time so the goose migrations cannot race each other.
test:
	cd $(API_DIR) && go test ./... -p 1
	cd $(WEB_DIR) && npm run check

build-web:
	cd $(WEB_DIR) && npm ci --include=dev && npm run build
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	cp -r $(WEB_DIR)/build/. $(DIST_DIR)/
	# This marker keeps the dist folder in git: `go:embed` needs the folder to
	# exist when building without the frontend (e.g. `go build ./...`).
	touch $(DIST_DIR)/.gitkeep

build: build-web
	mkdir -p bin
	cd $(API_DIR) && go build -o ../../bin/anabul ./cmd/server

docker:
	docker build -t anabul .

# Daily backup: writes a dump into backups/ and keeps the newest BACKUP_KEEP files.
backup:
	cd $(API_DIR) && BACKUP_DIR="$(abspath backups)" go run ./cmd/backup

# Restore a dump: make restore FILE=backups/anabul-20261014-070000.sql.gz
restore:
	@test -n "$(FILE)" || (echo "usage: make restore FILE=backups/anabul-....sql.gz" && exit 1)
	cd $(API_DIR) && go run ./cmd/restore "$(abspath $(FILE))"

# Run the app plus the reverse proxy with HTTPS (needs DOMAIN and a filled .env).
prod-up:
	docker compose -f docker-compose.prod.yml up -d --build

prod-down:
	docker compose -f docker-compose.prod.yml down

# Register the Telegram webhook against the public URL (needs TELEGRAM_TOKEN,
# TELEGRAM_WEBHOOK_SECRET, TELEGRAM_OWNER_CHAT_ID, and PUBLIC_URL in .env).
webhook:
	cd $(API_DIR) && go run ./cmd/telegram-webhook set

telegram-info:
	cd $(API_DIR) && go run ./cmd/telegram-webhook info
