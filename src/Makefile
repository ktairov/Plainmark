BUILD = go run ./tools/buildapp

.PHONY: build cross linux-amd64 linux-arm64 windows-amd64 windows-arm64

# Local platform. macOS produces the .app without a disk image.
build:
	$(BUILD) -skip-dmg

# Portable Windows builds; installers can be built separately with NSIS.
cross:
	$(BUILD) -platform linux/amd64,linux/arm64,windows/amd64,windows/arm64 -skip-installer

linux-amd64:
	$(BUILD) -platform linux/amd64

linux-arm64:
	$(BUILD) -platform linux/arm64

windows-amd64:
	$(BUILD) -platform windows/amd64 -skip-installer

windows-arm64:
	$(BUILD) -platform windows/arm64 -skip-installer
