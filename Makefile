.PHONY: generate migrate run build help

generate: ## Generate SQL client code (sqlc) and API types (oapi-codegen)
	go run github.com/sqlc-dev/sqlc/cmd/sqlc generate
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config oapi-codegen.yaml api/openapi.yaml

migrate: ## Apply pending dbmate migrations (requires DATABASE_URL)
	dbmate up

run: ## Run the development server
	go run ./cmd/server

build: ## Compile the server binary to bin/server
	go build -o bin/server ./cmd/server

.DEFAULT_GOAL := help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
