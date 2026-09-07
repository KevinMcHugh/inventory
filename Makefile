.PHONY: dev generate migrate bootstrap run build help

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

run: ## Run just the API server (no web, no migrations)
	go run ./cmd/server

build: ## Compile the server binary to bin/server
	go build -o bin/server ./cmd/server

.DEFAULT_GOAL := help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
