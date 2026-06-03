.PHONY: build test testacc tidy fmt clean docs

-include .env
export

ifeq ($(OS),Windows_NT)
    RM_BIN = if exist "$(subst /,\,$(BIN_DIR))" rmdir /s /q "$(subst /,\,$(BIN_DIR))"
    BIN_EXT = .exe
else
    RM_BIN = rm -rf $(BIN_DIR)
    BIN_EXT =
endif

BIN      ?= terraform-provider-openwebui$(BIN_EXT)
GO       ?= go
BIN_DIR  ?= $(CURDIR)/bin
VERSION  ?= 0.0.0-dev
LDFLAGS  ?= -X github.com/docktape/terraform-provider-openwebui/internal/provider.Version=$(VERSION)

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BIN)

test:
	$(GO) test ./...

testacc:
	$(GO) test ./internal/provider/... -v -count=1 -timeout 30m

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

clean:
	$(RM_BIN)

docs:
	$(GO) generate ./...
