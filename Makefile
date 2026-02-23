.PHONY: gifs proto ts-proto codegen frontend-check buf-lint test build install ci ui-build dev-backend dev-frontend dev-tmux

all: gifs

VERSION=v0.1.14
GORELEASER_ARGS ?= --skip=sign --snapshot --clean
GORELEASER_TARGET ?= --single-target

AGENT_UI_DIR=agent-ui-system
AGENT_UI_TSC_BIN=$(AGENT_UI_DIR)/node_modules/.bin/tsc
AGENT_UI_TS_PROTO_BIN=$(AGENT_UI_DIR)/node_modules/.bin/protoc-gen-ts_proto

TAPES=$(wildcard doc/vhs/*tape)
gifs: $(TAPES)
	for i in $(TAPES); do vhs < $$i; done

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v

lint:
	golangci-lint run -v

lintmax:
	golangci-lint run -v --max-same-issues=100

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude-generated -exclude=G101,G304,G301,G306 -exclude-dir=.history -exclude-dir=proto/generated/go ./...

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

test:
	go test ./... -count=1

frontend-check: $(AGENT_UI_TSC_BIN)
	pnpm -C $(AGENT_UI_DIR) run check

buf-lint:
	buf lint .

$(AGENT_UI_DIR)/node_modules/.bin/%: $(AGENT_UI_DIR)/package.json $(AGENT_UI_DIR)/pnpm-lock.yaml
	@command -v pnpm >/dev/null || (echo "pnpm is required (https://pnpm.io/installation)" && exit 1)
	pnpm -C $(AGENT_UI_DIR) install --frozen-lockfile

ts-proto: $(AGENT_UI_TS_PROTO_BIN)
	pnpm -C $(AGENT_UI_DIR) run proto

proto:
	protoc --proto_path=proto --proto_path=/usr/include \
		--go_out=proto/generated/go --go_opt=paths=source_relative \
		proto/plz_confirm/v1/*.proto

codegen: proto ts-proto

build: codegen ui-build
	go build -tags embed ./...

ci: buf-lint test frontend-check

DEV_API_ADDR ?= :3001
DEV_UI_PORT ?= 3000

.PHONY: ui-build

ui-build:
	GOWORK=off go run ./internal/server/generate_build.go

dev-backend:
	go run ./cmd/plz-confirm serve --addr "$(DEV_API_ADDR)"

dev-frontend: $(AGENT_UI_DIR)/node_modules/.bin/vite
	pnpm -C $(AGENT_UI_DIR) dev --host --port "$(DEV_UI_PORT)"

dev-tmux:
	API_ADDR="$(DEV_API_ADDR)" UI_PORT="$(DEV_UI_PORT)" bash scripts/tmux-up.sh

goreleaser:
	goreleaser release $(GORELEASER_ARGS) $(GORELEASER_TARGET)

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/plz-confirm@$(shell svu current)

bump-glazed:
	go get github.com/go-go-golems/glazed@latest
	go get github.com/go-go-golems/clay@latest
	go get github.com/go-go-golems/go-go-goja@latest
	go mod tidy

PLZ_CONFIRM_BINARY=$(shell which plz-confirm)
install: ui-build
	go build -tags embed -o ./dist/plz-confirm ./cmd/plz-confirm && \
		cp ./dist/plz-confirm $(PLZ_CONFIRM_BINARY)
