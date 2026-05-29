.PHONY: build test testacc tidy fmt clean docs

ifeq ($(OS),Windows_NT)
    SET_TF_ACC := set TF_ACC=1&&
else
    SET_TF_ACC := TF_ACC=1
endif

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
	$(SET_TF_ACC) $(GO) test ./internal/provider/... -v -count=1 -timeout 30m

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

clean:
	rm -rf $(BIN_DIR)

docs:
	$(GO) generate ./...
