# --------------------------------------------------------------------
# 	Makefile
# --------------------------------------------------------------------

# Targets

# Default target
.PHONY: all
all: build

# Build render-diff
.PHONY: build
build:
	go build -o render-diff

# Run golangci-lint
.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run

# Run go vet
.PHONY: vet
vet:
	go vet ./...

# Run go fmt
.PHONY: fmt
fmt:
	go fmt ./...

# Run go tests if they exist
.PHONY: test
test:
	go test ./... -v

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

## Tool versions
GOLANGCI_LINT_VERSION ?= v2.6.0

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef