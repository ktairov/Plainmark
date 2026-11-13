MYGO = go run github.com/egoist/mygo/cmd/mygo@v0.7.4

.PHONY: build cross linux-amd64 linux-arm64 windows-amd64 windows-arm64

# Local platform. macOS produces the .app without a disk image.
build:
	$(MYGO) build -skip-dmg

# Portable Windows builds; installers can be built separately with NSIS.
cross:
	$(MYGO) build -platform linux/amd64,linux/arm64,windows/amd64,windows/arm64 -skip-installer

linux-amd64:
	$(MYGO) build -platform linux/amd64

linux-arm64:
	$(MYGO) build -platform linux/arm64

windows-amd64:
	$(MYGO) build -platform windows/amd64 -skip-installer

windows-arm64:
	$(MYGO) build -platform windows/arm64 -skip-installer
