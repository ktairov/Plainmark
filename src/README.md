# Plainmark — development

The Go implementation of Plainmark uses MyGo for its native desktop interface. Markdown folders open directly. The app includes a note tree, tabs, file operations, file watching and conflict handling, Markdown preview, encrypted blocks, settings, and session restoration.

For a feature overview and quick start, see the [project README](../README.md).

## Requirements

- Go 1.27.3 or newer.
- GTK 3 on Linux.
- Python 3 for the native integration scripts.
- Cargo only for the optional compatibility check against the original Rust encryption code.

Node.js, Rust, cgo, and a WebView runtime are not needed to build or run the application itself.

## Run and build

Run these commands from `src/`:

```sh
go run ./cmd/plainmark ../testdata/workspace
go run ./cmd/plainmark ../testdata/workspace/Math.md
```

Package the app for the current platform:

```sh
make build
```

On Apple Silicon:

```sh
open "build/release/darwin-arm64/Plainmark.app"
```

All packaging targets use `go run ./tools/buildapp`, which increments the third component of the version in `mygo.json` before invoking MyGo. A failed or partial build still reserves its build number.

The macOS bundle uses a local ad-hoc signature and is not notarized. Go is not required to run the packaged app.

Linux and Windows builds support AMD64 and ARM64. All targets share `mygo.json`; `make cross` builds all four targets, and `make linux-amd64`, `make linux-arm64`, `make windows-amd64`, and `make windows-arm64` build them individually. Windows targets produce portable builds by default; installers require NSIS. Cross-compilation does not replace native GUI testing on each target system.

For an unpackaged development binary:

```sh
go build -o build/plainmark ./cmd/plainmark
```

## Settings and session state

Open settings through **Plainmark → Settings…** or **Cmd+,** on macOS, and **Edit → Settings…** or **Ctrl+,** on Linux and Windows. The toolbar's day/night control temporarily changes the appearance of the current window. With System appearance selected, a system theme change clears that override and resumes following the system. Explicit Day and Night preferences remain independent of system appearance.

The default macOS configuration file is:

```text
~/Library/Application Support/Plainmark/state.json
```

Other platforms use Go's user configuration directory with the same `Plainmark/state.json` suffix. To use an isolated profile, set `PLAINMARK_CONFIG_DIR` to a directory.

The state file stores preferences and navigation, including folders, tabs, view modes, scroll positions, and window geometry. It does not store note bodies, unsaved drafts, passwords, or decrypted blocks. If the renamed profile does not exist yet, the app reads the previous Go version's profile and writes subsequent changes to the new path. Tauri session state is not imported.

## Checks

Run the Go tests and static checks:

```sh
go test ./...
go vet ./...
go test -race ./internal/desktop
```

Native UI scripts launch a separate process and use temporary notes and configuration. Build the automation-enabled binary first, then run the scripts sequentially:

```sh
go build -tags e2e -o build/plainmark ./cmd/plainmark
python3 scripts/smoke.py
python3 scripts/lifecycle.py
python3 scripts/settings_ui.py
python3 scripts/navigation.py
python3 scripts/menu_ui.py
python3 scripts/restoration.py
python3 scripts/appearance_ui.py
python3 scripts/titlebar_ui.py
python3 scripts/search_ui.py
python3 scripts/split_scroll_ui.py
```

Generate preview snapshots:

```sh
PLAINMARK_SNAPSHOTS=/tmp/plainmark-snapshots go test ./internal/desktop -run Snapshot -count=1
```

Compare the math engines; output is written to `/tmp/plainmark-mathprobe`:

```sh
go run ./tools/mathprobe
```

Check encryption compatibility with the original Rust implementation from commit `cb2275f` (requires Cargo and Git history):

```sh
python3 scripts/crypto_compat.py
```

## Dependencies and project history

MyGo is pinned to v0.7.4, and the application's math renderer, `go-tex/math`, to v0.50.0. The Luo-Studio math dependency is used only by the comparison tool and is not imported by the application.

The original Tauri/Rust/React sources are preserved in Git history at `cb2275f`. The current implementation has its own palettes, translations, icons, and fixtures. The Linux package uses GTK 3 without WebKitGTK (`linux.webview: false`).
