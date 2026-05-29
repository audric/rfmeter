# rfmeter build automation.
#
#   make            build the native (Linux) binary
#   make linux      build ./rfmeter
#   make windows    cross-build ./rfmeter.exe (no console window)
#   make all        build both
#   make run        build and run the Linux binary
#   make test       go test -race ./...
#   make vet fmt    go vet / gofmt -w
#   make icon       regenerate the Windows .ico + embedded .syso resource
#   make clean      remove build artifacts

PKG     := ./cmd/rfmeter
BIN     := rfmeter
LDFLAGS := -s -w

# GUI subsystem on Windows so no console window pops up alongside the app.
WIN_LDFLAGS := $(LDFLAGS) -H=windowsgui

ICON_DIR := build/icon
ICON_PNG := $(ICON_DIR)/rfmeter-256.png
ICON_ICO := $(ICON_DIR)/rfmeter.ico
SYSO     := cmd/rfmeter/rfmeter_windows_amd64.syso

.PHONY: all linux windows run test vet fmt icon clean

linux:
	go build -ldflags="$(LDFLAGS)" -o $(BIN) $(PKG)

windows: $(SYSO)
	GOOS=windows GOARCH=amd64 go build -ldflags="$(WIN_LDFLAGS)" -o $(BIN).exe $(PKG)

all: linux windows

run: linux
	./$(BIN)

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

# Rebuild the embedded Windows icon resource from the source artwork.
# Requires ImageMagick (`magick`); fetches akavel/rsrc via `go run`.
icon: $(SYSO)
$(SYSO): $(ICON_PNG)
	magick $(ICON_PNG) -define icon:auto-resize=256,128,64,48,32,16 $(ICON_ICO)
	go run github.com/akavel/rsrc@latest -ico $(ICON_ICO) -arch amd64 -o $(SYSO)

clean:
	rm -f $(BIN) $(BIN).exe

# Default target.
.DEFAULT_GOAL := linux
