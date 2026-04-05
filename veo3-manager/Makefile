APP_NAME = Veo3Manager
VERSION = 1.0.0
LDFLAGS = -s -w

.PHONY: all windows macos-arm64 macos-amd64 linux clean

## Build all platforms (run on respective OS or CI)
all: windows macos-arm64 macos-amd64 linux

## Windows build (run on Windows)
windows:
	wails build -clean -trimpath -ldflags "$(LDFLAGS)"

## Windows build with NSIS installer
windows-installer:
	wails build -clean -trimpath -ldflags "$(LDFLAGS)" -nsis

## macOS Apple Silicon (run on macOS)
macos-arm64:
	wails build -clean -trimpath -ldflags "$(LDFLAGS)" -platform darwin/arm64

## macOS Intel (run on macOS)
macos-amd64:
	wails build -clean -trimpath -ldflags "$(LDFLAGS)" -platform darwin/amd64

## Linux (run on Linux)
linux:
	wails build -clean -trimpath -ldflags "$(LDFLAGS)" -platform linux/amd64

## Clean build artifacts
clean:
	rm -rf build/bin/*

## Dev mode
dev:
	wails dev
