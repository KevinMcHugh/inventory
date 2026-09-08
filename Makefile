.PHONY: dev generate migrate bootstrap keys-list keys-create keys-rotate invites-create invites-list run build help

dev: migrate ## Migrate, then start the API and web dev server together
	@echo ">> API on :8080, web on :5173 (Ctrl+C to stop both)"
	@trap 'kill 0' EXIT INT TERM; \
	 go run ./cmd/server & \
	 (cd web && npm run dev) & \
	 wait

generate: ## Generate SQL client code (sqlc) and API types (oapi-codegen)
	go run github.com/sqlc-dev/sqlc/cmd/sqlc generate
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config oapi-codegen.yaml api/openapi.yaml

migrate: ## Apply pending dbmate migrations (requires DATABASE_URL)
	dbmate up

bootstrap: ## Create a tenant and its first api key (usage: make bootstrap NAME="Kev")
	@if [ -z "$(NAME)" ]; then echo "usage: make bootstrap NAME=<tenant name>"; exit 1; fi
	go run ./cmd/server bootstrap --tenant "$(NAME)"

keys-list: ## List active api keys for a tenant (usage: make keys-list TENANT=<id>)
	@if [ -z "$(TENANT)" ]; then echo "usage: make keys-list TENANT=<tenant id>"; exit 1; fi
	go run ./cmd/server keys list --tenant "$(TENANT)"

keys-create: ## Mint a new api key for a tenant (usage: make keys-create TENANT=<id> NAME=<label>)
	@if [ -z "$(TENANT)" ] || [ -z "$(NAME)" ]; then echo "usage: make keys-create TENANT=<id> NAME=<label>"; exit 1; fi
	go run ./cmd/server keys create --tenant "$(TENANT)" --name "$(NAME)"

keys-rotate: ## Rotate an api key by id (usage: make keys-rotate KEY_ID=<id>)
	@if [ -z "$(KEY_ID)" ]; then echo "usage: make keys-rotate KEY_ID=<key id>"; exit 1; fi
	go run ./cmd/server keys rotate --key-id "$(KEY_ID)"

invites-create: ## Mint a single-use invite (optional TENANT=<xid> links to existing tenant)
	go run ./cmd/server invites create $(if $(TENANT),--tenant "$(TENANT)") $(if $(NOTE),--note "$(NOTE)")

invites-list: ## List outstanding (unredeemed, non-expired) invites
	go run ./cmd/server invites list

run: ## Run just the API server (no web, no migrations)
	go run ./cmd/server

build: ## Compile the server binary to bin/server
	go build -o bin/server ./cmd/server

.DEFAULT_GOAL := help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
