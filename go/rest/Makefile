# ============================================================================
# Local Development Commands
# ============================================================================

.PHONY: setup
setup: tools tidy

.PHONY: tools
tools:
	@go install tool

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: test
test:
	@go test -v ./...

.PHONY: lint
lint:
	@go tool golangci-lint run

.PHONY: fmt
fmt:
	@go fmt ./...

.PHONY: ogen
ogen:
	@go tool ogen --target internal/oas \
		-package oas \
		--clean \
		api/openapi.yaml

.PHONY: hadolint
hadolint:
	@docker run --rm -i hadolint/hadolint < build/Dockerfile

# ============================================================================
# Docker Commands
# ============================================================================

.PHONY: build
build:
	@docker build --no-cache -f build/Dockerfile -t go-rest-server .

.PHONY: run
run:
	@docker run --rm -p 8080:8080 --env-file .env go-rest-server
