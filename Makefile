# vibeguard Makefile — local dev convenience.
# CI runs `go test ./...` and `vibeguard lint ./...` directly.

GO        ?= go
BIN_DIR   ?= bin
PLATFORM   = ./platform
OPERATOR   = ./operator

.PHONY: all build test lint clean demo install fmt vet check tidy

all: build

build: $(BIN_DIR)/vibeguard $(BIN_DIR)/vibeguard-mcp

$(BIN_DIR)/vibeguard:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $@ ./cmd/vibeguard

$(BIN_DIR)/vibeguard-mcp:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $@ ./cmd/vibeguard-mcp

test:
	$(GO) test ./...
	cd $(PLATFORM) && $(GO) test ./...
	cd $(OPERATOR) && $(GO) build ./...

vet:
	$(GO) vet ./...
	cd $(PLATFORM) && $(GO) vet ./...
	cd $(OPERATOR) && $(GO) vet ./...

fmt:
	$(GO) fmt ./...
	cd $(PLATFORM) && $(GO) fmt ./...
	cd $(OPERATOR) && $(GO) fmt ./...

lint: build
	$(BIN_DIR)/vibeguard lint ./...

check: fmt vet test

tidy:
	$(GO) mod tidy
	cd $(PLATFORM) && $(GO) mod tidy
	cd $(OPERATOR) && $(GO) mod tidy

# `make demo` end-to-end: validate the sample, dump IR, generate a project,
# build the generated project, lint the result.
demo: build
	@echo "==> validate"
	$(BIN_DIR)/vibeguard validate -f fixtures/sample_vibeguard.yaml
	@echo
	@echo "==> ir dump"
	$(BIN_DIR)/vibeguard ir dump -f fixtures/sample_vibeguard.yaml
	@echo
	@echo "==> generate /tmp/team-task-saas"
	rm -rf /tmp/team-task-saas
	$(BIN_DIR)/vibeguard generate -f fixtures/sample_vibeguard.yaml -o /tmp/team-task-saas
	@echo
	@echo "==> build generated project"
	cd /tmp/team-task-saas && \
		echo 'replace github.com/vibeguard/platform => $(CURDIR)/platform' >> go.mod && \
		$(GO) mod tidy 2>/dev/null && \
		$(GO) build ./... && \
		echo "  ✓ generated project compiles"
	@echo
	@echo "==> lint generated project"
	cd /tmp/team-task-saas && $(CURDIR)/$(BIN_DIR)/vibeguard lint ./... | tail -3
	@echo
	@echo "==> MCP smoke"
	printf '{"jsonrpc":"2.0","id":1,"method":"initialize"}\n{"jsonrpc":"2.0","id":2,"method":"tools/list"}\n' | $(BIN_DIR)/vibeguard-mcp | head -2

clean:
	rm -rf $(BIN_DIR)
	rm -rf /tmp/team-task-saas

install: build
	cp $(BIN_DIR)/vibeguard /usr/local/bin/vibeguard
	cp $(BIN_DIR)/vibeguard-mcp /usr/local/bin/vibeguard-mcp
