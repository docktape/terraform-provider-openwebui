
.PHONY: build test tidy fmt clean

BIN ?= terraform-provider-openwebui
GO ?= go
BIN_DIR ?= $(CURDIR)/bin
VERSION ?= 0.0.0-dev
LDFLAGS ?= -X github.com/docktape/terraform-provider-openwebui/internal/provider.Version=$(VERSION)

# Build the provider binary into ./bin. Point a Terraform dev override at $(BIN_DIR)
# to test it locally (see README.md > Local Development & Testing).
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BIN)

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

clean:
	rm -rf $(BIN_DIR)
