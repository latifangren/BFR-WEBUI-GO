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
	@cp -f customize.sh module.prop service.sh system.prop update.json env.example uninstall.sh .tmp/pkg-full/
	@cp -rf META-INF bin .tmp/pkg-full/
	@echo "--> Packaging $(ZIP_FULL)..."
	@rm -f $(ZIP_FULL)
	@(cd .tmp/pkg-full && zip -r ../../$(ZIP_FULL) META-INF customize.sh module.prop service.sh system.prop update.json env.example uninstall.sh bin webui)
	@rm -rf .tmp/pkg-full

$(ZIP_NOMODEM): frontend-nomodem
	@echo "--> Compiling Clean No-Modem Android ARM64 binary..."
	@mkdir -p .tmp/pkg-nomodem
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -tags nomodem -ldflags="-s -w" -o .tmp/pkg-nomodem/webui .
	@echo "--> Running zero-leak forensic audit..."
	@python3 scripts/audit_nomodem.py .tmp/pkg-nomodem/webui || python scripts/audit_nomodem.py .tmp/pkg-nomodem/webui
	@echo "--> Staging Clean No-Modem Magisk package files..."
	@cp -f customize.sh module.prop service.sh system.prop update.json env.example uninstall.sh .tmp/pkg-nomodem/
	@cp -rf META-INF bin .tmp/pkg-nomodem/
	@rm -f .tmp/pkg-nomodem/bin/band_manager.dex .tmp/pkg-nomodem/bin/sim_switcher.dex
	@echo "--> Packaging $(ZIP_NOMODEM)..."
	@rm -f $(ZIP_NOMODEM)
	@(cd .tmp/pkg-nomodem && zip -r ../../$(ZIP_NOMODEM) META-INF customize.sh module.prop service.sh system.prop update.json env.example uninstall.sh bin webui)
	@rm -rf .tmp/pkg-nomodem

frontend-full:
	@echo "--> Preparing Full Modem frontend assets..."
	@(cd frontend && VITE_HAS_VENDOR_MODEM=true bun run build)

frontend-nomodem:
	@echo "--> Preparing Clean No-Modem frontend assets..."
	@(cd frontend && VITE_HAS_VENDOR_MODEM=false bun run build)

test:
	@echo "--> Running unit tests (Full & No-Modem)..."
	go test -v ./...
	go test -v -tags nomodem ./...

clean:
	@echo "--> Cleaning build artifacts..."
	@rm -rf .tmp/pkg-full .tmp/pkg-nomodem .tmp/webui* $(ZIP_FULL) $(ZIP_NOMODEM)
