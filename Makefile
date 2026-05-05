# Extract CRD JSON schemas with Datree's CRD Extractor into mozcloud/crdSchemas
# Uses your current kubectl context and should be run against multiple clusters to collect all MozCloud CRDs
# These CRDs are used by our helm CI to validate manifests /w Kubeconform
#
# Run `make help` for available targets.

SHELL := /usr/bin/env bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c

CRDS_CATALOG_REPO ?= https://github.com/datreeio/CRDs-catalog.git
CRDS_CATALOG_REF  ?= 808cecce07adf438cde06b413250be548981321c # Pinning to a specific commit
CRDS_CATALOG_DIR  ?= .cache/CRDs-catalog

MOZCLOUD_DIR ?= .
MOZCLOUD_SCHEMAS_DIR ?= $(MOZCLOUD_DIR)/crdSchemas
TARGET_DIR   ?= $(MOZCLOUD_DIR)/.cache/crdSchemas

# Public GCS bucket — same one used by the release workflow for tool binaries.
RELEASE_BUCKET      ?= moz-fx-platform-shared-global-mozcloud-tools
CRDS_BUCKET_PREFIX  ?= crdSchemas

# Requirements
GIT      ?= git
KUBECTL  ?= kubectl
PYTHON3  ?= python3
GCLOUD   ?= gcloud

.PHONY: clone checkout_catalog extract copy clean update_crds upload_crds help
.DEFAULT_GOAL := help

help:
	@echo "Targets:"
	@echo "  make update_crds - Clone/update CRDs-catalog and copy extracted CRDs into $(MOZCLOUD_SCHEMAS_DIR)"
	@echo "  make upload_crds - Rsync $(MOZCLOUD_SCHEMAS_DIR) to gs://$(RELEASE_BUCKET)/$(CRDS_BUCKET_PREFIX)/"
	@echo "  make clean       - Remove local CRDs-catalog clone"
	@echo ""
	@echo "Requirements:"
	@echo "  - python3 (required by CRD extractor script)"
	@echo "  - kubectl (uses current kube context)"
	@echo "  - git"
	@echo "  - gcloud (required by upload_crds; auth via gcloud auth login or ADC)"
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

checkout_catalog:
	@echo "Resetting CRDs-catalog to pinned ref $(CRDS_CATALOG_REF)"
	@cd "$(CRDS_CATALOG_DIR)"; \
		$(GIT) fetch --depth 1 origin $(CRDS_CATALOG_REF); \
		$(GIT) checkout -q $(CRDS_CATALOG_REF); \
		$(GIT) reset --hard $(CRDS_CATALOG_REF)

extract:
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
	@OUTPUT_DIR="$(abspath $(TARGET_DIR))" \
		"$(CRDS_CATALOG_DIR)/Utilities/crd-extractor.sh"

copy:
	@echo "Copying CRDs from $(TARGET_DIR) into $(MOZCLOUD_SCHEMAS_DIR)"
	@cp -r $(TARGET_DIR)/ $(MOZCLOUD_SCHEMAS_DIR)

update_crds: clone extract copy

upload_crds:
	@test -d "$(MOZCLOUD_SCHEMAS_DIR)" || { \
		echo "Schemas directory not found: $(MOZCLOUD_SCHEMAS_DIR)"; \
		echo "Run 'make update_crds' first."; \
		exit 1; \
	}
	@if [ -z "$$(find "$(MOZCLOUD_SCHEMAS_DIR)" -mindepth 1 -name '*.json' -print -quit)" ]; then \
		echo "Schemas directory is empty: $(MOZCLOUD_SCHEMAS_DIR)"; \
		echo "Run 'make update_crds' first."; \
		exit 1; \
	fi
	@if [ -z "$${CI:-}" ]; then \
		echo "WARNING: This target publishes to the public production bucket"; \
		echo "  gs://$(RELEASE_BUCKET)/$(CRDS_BUCKET_PREFIX)/"; \
		echo "and is normally run by the sync-crds CI workflow on merges to main."; \
		echo "Running locally will overwrite the in-bucket schemas with whatever is"; \
		echo "in $(MOZCLOUD_SCHEMAS_DIR), and prune anything not present locally."; \
		echo ""; \
		printf "Continue? [y/N] "; \
		read -r reply; \
		case "$$reply" in y|Y|yes|YES) ;; *) echo "Aborted."; exit 1 ;; esac; \
	fi
	@echo "Syncing $(MOZCLOUD_SCHEMAS_DIR) -> gs://$(RELEASE_BUCKET)/$(CRDS_BUCKET_PREFIX)/"
	@$(GCLOUD) storage rsync --recursive --delete-unmatched-destination-objects \
		"$(MOZCLOUD_SCHEMAS_DIR)" "gs://$(RELEASE_BUCKET)/$(CRDS_BUCKET_PREFIX)/"

clean:
	@rm -rf "$(CRDS_CATALOG_DIR)"
	@echo "Removed $(CRDS_CATALOG_DIR)"
