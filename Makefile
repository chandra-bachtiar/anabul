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
	# Pakai image resmi supaya tidak perlu memasang sqlc di mesin.
	docker run --rm -v "$(CURDIR)/$(API_DIR):/src" -w /src sqlc/sqlc generate

test:
	cd $(API_DIR) && go test ./...
	cd $(WEB_DIR) && npm run check

build-web:
	cd $(WEB_DIR) && npm ci --include=dev && npm run build
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	cp -r $(WEB_DIR)/build/. $(DIST_DIR)/

build: build-web
	mkdir -p bin
	cd $(API_DIR) && go build -o ../../bin/anabul ./cmd/server

docker:
	docker build -t anabul .

# Backup harian: menyimpan dump ke backups/ dan menyisakan BACKUP_KEEP berkas terbaru.
backup:
	cd $(API_DIR) && BACKUP_DIR="$(abspath backups)" go run ./cmd/backup

# Kembalikan dump: make restore FILE=backups/anabul-20261014-070000.sql.gz
restore:
	@test -n "$(FILE)" || (echo "pakai: make restore FILE=backups/anabul-....sql.gz" && exit 1)
	cd $(API_DIR) && go run ./cmd/restore "$(abspath $(FILE))"

# Jalankan app + reverse proxy dengan HTTPS (butuh DOMAIN dan .env yang sudah diisi).
prod-up:
	docker compose -f docker-compose.prod.yml up -d --build

prod-down:
	docker compose -f docker-compose.prod.yml down

# Daftarkan webhook Telegram ke URL publik (butuh TELEGRAM_TOKEN, TELEGRAM_WEBHOOK_SECRET,
# TELEGRAM_OWNER_CHAT_ID, dan PUBLIC_URL di .env).
webhook:
	cd $(API_DIR) && go run ./cmd/telegram-webhook set

telegram-info:
	cd $(API_DIR) && go run ./cmd/telegram-webhook info
