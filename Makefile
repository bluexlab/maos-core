OPENAPI_FILE = ./doc/openapi.yaml
OAPI_CODEGEN_CONFIG = oapi-codegen-config.yml
GENERATED_API_SERVER = pkg/api/gen.go

TOPDIR := $(strip $(dir $(realpath $(lastword $(MAKEFILE_LIST)))))

CGO_ENABLED ?= 0
ifneq (,$(wildcard $(TOPDIR)/.env))
	include $(TOPDIR)/.env
	export
endif

comma:= ,
empty:=
space:= $(empty) $(empty)

bold := $(shell tput bold)
green := $(shell tput setaf 2)
sgr0 := $(shell tput sgr0)

MODULE_NAME := $(shell go list -m)

PLATFORM ?= $(platform)
ifneq ($(PLATFORM),)
	GOOS := $(or $(word 1, $(subst /, ,$(PLATFORM))),$(shell go env GOOS))
	GOARCH := $(or $(word 2, $(subst /, ,$(PLATFORM))),$(shell go env GOARCH))
endif

BIN_SUFFIX :=
ifneq ($(or $(GOOS),$(GOARCH)),)
	GOOS ?= $(shell go env GOOS)
	GOARCH ?= $(shell go env GOARCH)
	BIN_SUFFIX := $(BIN_SUFFIX)-$(GOOS)-$(GOARCH)
endif
ifeq ($(GOOS),windows)
	BIN_SUFFIX := $(BIN_SUFFIX).exe
endif

APPS := $(patsubst app/%/,%,$(sort $(dir $(wildcard app/*/))))
GOFILES := $(shell find . -type f -name '*.go' -not -path '*/\.*' -not -path './app/*')
$(foreach app,$(APPS),\
	$(eval GOFILES_$(app) := $(shell find ./app/$(app) -type f -name '*.go' -not -path '*/\.*')))

DOCFILES := $(shell find doc -type f -name '*.yaml')

.DEFAULT: all

.PHONY: all
all: $(APPS)

.PHONY: $(APPS)
$(APPS): %: bin/%$(BIN_SUFFIX)

.SECONDEXPANSION:
bin/%: $$(GOFILES) $$(GOFILES_$$(@F)) $$(ASSET_FILES) $(GENERATED_API_SERVER)
	@printf "Building $(bold)$@$(sgr0) ... "
	@go build -o ./bin/$(@F) ./app/$(@F:$(BIN_SUFFIX)=)
	@printf "$(green)done$(sgr0)\n"

.PHONY: gosec
gosec: ## Run the golang security checker
	@gosec -exclude-dir test/mock ./...

.PHONY: generate
generate: $(GENERATED_API_SERVER)

$(GENERATED_API_SERVER): $(DOCFILES) $(OAPI_CODEGEN_CONFIG)
	@echo "Generating server code from $(OPENAPI_FILE)..."
	@mkdir -p $(@D)
	go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest --config=$(OAPI_CODEGEN_CONFIG) $(OPENAPI_FILE)
