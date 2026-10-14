.PHONY: run build tidy test fmt vet clean \
        migrate-up migrate-down migrate-version migrate-create

APP_NAME := rogele
CMD_PATH := ./cmd/api
BIN_DIR  := ./bin

# Read DB_* vars from .env for migrations
include .env
export

MIGRATE_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

run:
	go run $(CMD_PATH)

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./... -race -cover

clean:
	rm -rf $(BIN_DIR)

migrate-up:
	migrate -path ./migrations -database "$(MIGRATE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(MIGRATE_URL)" down 1

migrate-version:
	migrate -path ./migrations -database "$(MIGRATE_URL)" version

migrate-create:
	migrate create -ext sql -dir ./migrations -seq $(name)