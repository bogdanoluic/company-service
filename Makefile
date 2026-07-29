ifneq (,$(wildcard ./.env))
	include .env
	export
endif

TERN_VERSION := v2.4.1
TERN := $(CURDIR)/bin/tern
MIGRATIONS_DIR := $(CURDIR)/migrations

.PHONY: tern-install migration-new migrate-up migrate-down migrate-status

$(TERN):
	mkdir -p $(dir $(TERN))
	GOBIN=$(CURDIR)/bin go install github.com/jackc/tern/v2@$(TERN_VERSION)

tern-install: $(TERN)

migration-new: $(TERN)
	@test -n "$(name)" || \
		(echo "usage: make migration-new name=create_companies"; exit 1)
	mkdir -p $(MIGRATIONS_DIR)
	TERN_MIGRATIONS=$(MIGRATIONS_DIR) $(TERN) new $(name)

migrate-up: $(TERN)
	TERN_MIGRATIONS=$(MIGRATIONS_DIR) $(TERN) migrate

migrate-down: $(TERN)
	TERN_MIGRATIONS=$(MIGRATIONS_DIR) \
		$(TERN) migrate --destination -1

migrate-status: $(TERN)
	TERN_MIGRATIONS=$(MIGRATIONS_DIR) $(TERN) status

run:
	go run ./cmd/server

build:
	go build -o bin/company-service ./cmd/server

test:
	go test ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin

test-integration:
	go test -tags=integration ./internal/storage/postgres -count=1