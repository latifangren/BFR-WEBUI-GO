.PHONY: all release build-full build-nomodem frontend-full frontend-nomodem audit test clean

VERSION ?= $(shell grep '^version=' module.prop | cut -d= -f2)
ifeq ($(VERSION),)
  VERSION := v1.2.8
endif

GOOS ?= android
GOARCH ?= arm64
CGO_ENABLED ?= 0

ZIP_FULL := BFR-WEBUI-Magisk-$(VERSION).zip
ZIP_NOMODEM := BFR-WEBUI-Magisk-$(VERSION)-nomodem.zip

all: release

release: $(ZIP_FULL) $(ZIP_NOMODEM)
	@echo "========================================================"
	@echo " Release Packages Built Successfully:"
	@echo "   1. $(ZIP_FULL)"
	@echo "   2. $(ZIP_NOMODEM)"
	@echo "========================================================"

$(ZIP_FULL): frontend-full
	@echo "--> Compiling Full Android ARM64 binary..."
	@mkdir -p .tmp/pkg-full
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -ldflags="-s -w" -o .tmp/pkg-full/webui .
	@echo "--> Staging Full Magisk package files..."
	@cp -f customize.sh module.prop service.sh system.prop update.json env.example .tmp/pkg-full/
	@cp -rf bin .tmp/pkg-full/
	@echo "--> Packaging $(ZIP_FULL)..."
	@rm -f $(ZIP_FULL)
	@(cd .tmp/pkg-full && zip -r ../../$(ZIP_FULL) customize.sh module.prop service.sh system.prop update.json env.example bin webui)
	@rm -rf .tmp/pkg-full

$(ZIP_NOMODEM): frontend-nomodem
	@echo "--> Compiling Clean No-Modem Android ARM64 binary..."
	@mkdir -p .tmp/pkg-nomodem
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -tags nomodem -ldflags="-s -w" -o .tmp/pkg-nomodem/webui .
	@git checkout frontend/src/config/features.ts frontend/src/components/tabs/vendor_tabs.ts >/dev/null 2>&1 || true
	@echo "--> Running zero-leak forensic audit..."
	@python3 scripts/audit_nomodem.py .tmp/pkg-nomodem/webui || python scripts/audit_nomodem.py .tmp/pkg-nomodem/webui
	@echo "--> Staging Clean No-Modem Magisk package files..."
	@cp -f customize.sh module.prop service.sh system.prop update.json env.example .tmp/pkg-nomodem/
	@cp -rf bin .tmp/pkg-nomodem/
	@rm -f .tmp/pkg-nomodem/bin/band_manager.dex .tmp/pkg-nomodem/bin/sim_switcher.dex
	@echo "--> Packaging $(ZIP_NOMODEM)..."
	@rm -f $(ZIP_NOMODEM)
	@(cd .tmp/pkg-nomodem && zip -r ../../$(ZIP_NOMODEM) customize.sh module.prop service.sh system.prop update.json env.example bin webui)
	@rm -rf .tmp/pkg-nomodem

frontend-full:
	@echo "--> Preparing Full Modem frontend assets..."
	@git checkout frontend/src/config/features.ts frontend/src/components/tabs/vendor_tabs.ts >/dev/null 2>&1 || true
	@(cd frontend && bun run build)

frontend-nomodem:
	@echo "--> Preparing Clean No-Modem frontend assets..."
	@echo "export const HAS_VENDOR_MODEM = false;" > frontend/src/config/features.ts
	@printf "import type { Component } from 'svelte'\n\nexport const VENDOR_TABS: Record<string, Component> = {}\n" > frontend/src/components/tabs/vendor_tabs.ts
	@(cd frontend && bun run build)

test:
	@echo "--> Running unit tests (Full & No-Modem)..."
	go test -v ./...
	go test -v -tags nomodem ./...

clean:
	@echo "--> Cleaning build artifacts..."
	@rm -rf .tmp/pkg-full .tmp/pkg-nomodem .tmp/webui* $(ZIP_FULL) $(ZIP_NOMODEM)
	@git checkout frontend/src/config/features.ts frontend/src/components/tabs/vendor_tabs.ts >/dev/null 2>&1 || true
