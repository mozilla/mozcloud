# Extract CRD JSON schemas with Datree's CRD Extractor into mozcloud/crdSchemas
# Uses your current kubectl context
# These CRDs are used by our helm CI to validate manifests /w Kubeconform
#
# Usage:
#   make extract   # clone/update CRDs-catalog and run extractor, writing directly to crdSchemas
#   make clean     # remove cloned CRDs-catalog cache

SHELL := /usr/bin/env bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c

CRDS_CATALOG_REPO ?= https://github.com/datreeio/CRDs-catalog.git
CRDS_CATALOG_REF  ?= 808cecce07adf438cde06b413250be548981321c # Pinning to a specific commit
CRDS_CATALOG_DIR  ?= .cache/CRDs-catalog

MOZCLOUD_DIR ?= ../mozcloud
TARGET_DIR   ?= $(MOZCLOUD_DIR)/crdSchemas

# Requirements
GIT      ?= git
KUBECTL  ?= kubectl
PYTHON3  ?= python3

.PHONY: clone update extract clean help
.DEFAULT_GOAL := help

help:
	@echo "Targets:"
	@echo "  make extract   - Clone/update CRDs-catalog and extract CRDs into $(TARGET_DIR)"
	@echo "  make clean     - Remove local CRDs-catalog clone"
	@echo ""
	@echo "Variables:"
	@echo "  MOZCLOUD_DIR=<path to mozcloud> (default: ../mozcloud)"
	@echo "  TARGET_DIR=$(TARGET_DIR)"
	@echo ""

clone:
	@if [ ! -d "$(CRDS_CATALOG_DIR)/.git" ]; then \
		echo "Cloning $(CRDS_CATALOG_REPO) into $(CRDS_CATALOG_DIR)"; \
		$(GIT) clone "$(CRDS_CATALOG_REPO)" "$(CRDS_CATALOG_DIR)"; \
	fi
	@cd "$(CRDS_CATALOG_DIR)"; \
		echo "Checking out pinned ref $(CRDS_CATALOG_REF)"; \
		$(GIT) fetch --depth 1 origin $(CRDS_CATALOG_REF); \
		$(GIT) checkout -q $(CRDS_CATALOG_REF)

update:
	@echo "Resetting CRDs-catalog to pinned ref $(CRDS_CATALOG_REF)"
	@cd "$(CRDS_CATALOG_DIR)"; \
		$(GIT) fetch --depth 1 origin $(CRDS_CATALOG_REF); \
		$(GIT) checkout -q $(CRDS_CATALOG_REF); \
		$(GIT) reset --hard $(CRDS_CATALOG_REF)

extract: clone
	@echo "Running CRD extractor with current kubectl context: $$($(KUBECTL) config current-context)"
	@test -f "$(CRDS_CATALOG_DIR)/Utilities/crd-extractor.sh" || { \
		echo "Extractor not found at: $(CRDS_CATALOG_DIR)/Utilities/crd-extractor.sh"; \
		echo "Contents of $(CRDS_CATALOG_DIR):"; ls -la "$(CRDS_CATALOG_DIR)"; \
		echo "Contents of $(CRDS_CATALOG_DIR)/Utilities (if present):"; ls -la "$(CRDS_CATALOG_DIR)/Utilities" || true; \
		exit 1; \
	}
	@chmod +x "$(CRDS_CATALOG_DIR)/Utilities/crd-extractor.sh"
	@mkdir -p "$(TARGET_DIR)"
	@echo "Writing directly to: $(TARGET_DIR)"
	@DATREE_CATALOG_OUTPUT_DIR="$(abspath $(TARGET_DIR))" \
		"$(CRDS_CATALOG_DIR)/Utilities/crd-extractor.sh"

clean:
	@rm -rf "$(CRDS_CATALOG_DIR)"
	@echo "Removed $(CRDS_CATALOG_DIR)"
