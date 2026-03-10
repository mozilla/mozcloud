# --------------------------------------------------------------------
# 	Makefile
# --------------------------------------------------------------------

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
GOLANGCI_LINT_VERSION ?= v2.6.0

.PHONY: all
all: build

.PHONY: build
build:
	go build -o $(LOCALBIN)/mozcloud-mcp .

.PHONY: install
install:
	go install .

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	go test ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef
