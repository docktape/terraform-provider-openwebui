.PHONY: build test testacc tidy fmt clean docs

BIN      ?= terraform-provider-openwebui
GO       ?= go
BIN_DIR  ?= $(CURDIR)/bin
VERSION  ?= 0.0.0-dev
LDFLAGS  ?= -X github.com/docktape/terraform-provider-openwebui/internal/provider.Version=$(VERSION)

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BIN)

test:
	$(GO) test ./...

testacc:
	TF_ACC=1 $(GO) test ./internal/provider/... -v -count=1 -timeout 30m

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

clean:
	rm -rf $(BIN_DIR)

docs:
	$(GO) generate ./...
