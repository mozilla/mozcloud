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
	@echo "Building render-diff "
	go build -o render-diff

.PHONY: lint
lint:
	@echo "Running golangci-lint"
	golangci-lint run

# Run go fmt
.PHONY: fmt
fmt:
	go fmt .

# Run go tests
.PHONY: test
test:
	go test ./... -v
