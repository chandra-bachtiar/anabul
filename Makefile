SHELL := /bin/sh

WEB_DIR := apps/web
API_DIR := apps/api
DIST_DIR := $(API_DIR)/internal/webui/dist

.PHONY: dev-db dev-api dev-web migrate sqlc test build build-web docker

dev-db:
	docker compose up -d postgres

dev-api:
	cd $(API_DIR) && go run ./cmd/server

dev-web:
	cd $(WEB_DIR) && NODE_ENV=development npm run dev

migrate:
	cd $(API_DIR) && go run ./cmd/migrate

sqlc:
	cd $(API_DIR) && sqlc generate

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
