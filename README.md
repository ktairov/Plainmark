# Plainmark

Plainmark is a desktop Markdown notes app with a native interface, built with Go and [MyGo](https://github.com/egoist/mygo). Open a folder and work directly with your Markdown files.

## Features

- **Your notes are files.** Open `.md` and `.mdx` files, browse folders, filter the note tree, pin notes, and manage files and folders.
- **Three views.** Edit Markdown, read the rendered document, or use split view with synchronized scrolling.
- **Tabs and editing.** Keep multiple notes open, use undo/redo, and search in the editor or preview with highlighted matches.
- **Rich Markdown.** Tables, task lists, footnotes, syntax-highlighted code, text diagrams, inline and display math, local images, and relative links with heading anchors.
- **External changes.** File watching reloads clean notes and shows conflicts when a file with unsaved edits changes on disk.
- **Encrypted blocks.** Encrypt selected text, unlock blocks for reading, and lock them again.
- **Personalization.** English and Russian interfaces, light/dark/system appearance, theme palettes, fonts, and reading width.
- **Session restore.** Restore folders, tabs, view modes, scroll positions, and window size and position.

## Run from source

Requires **Go 1.27.3 or newer**. Linux also requires **GTK 3**. The application uses native UI; Node.js, Rust, cgo, and a WebView runtime are not required.

```sh
git clone https://github.com/ktairov/Plainmark.git
cd Plainmark/next
go run ./cmd/plainmark
```

Choose a notes folder in the application, or pass a folder or Markdown file as an argument:

```sh
go run ./cmd/plainmark ../testdata/workspace
go run ./cmd/plainmark ../testdata/workspace/Math.md
```

## Build

From `next/`, build the app for the current platform:

```sh
make build
```

On Apple Silicon, the macOS app is written to `next/build/release/darwin-arm64/Plainmark.app`. Each packaged build increments the build number in `next/mygo.json`.

Linux and Windows targets are available for AMD64 and ARM64. See the [build guide](docs/BUILDING.md) and [Arch Linux guide](docs/LINUX_ARCH.md) for commands and packaging details; these guides are currently in Russian. Native GUI checks have been performed on macOS; Linux and Windows still need validation on their target systems.

## Development

```sh
cd next
go test ./...
go vet ./...
```

See the [developer README](next/README.md) for native UI checks, configuration, and project dependencies. The Go implementation lives in `next/`. The earlier Tauri implementation is preserved in Git history at commit `cb2275f`.

## License

[The Unlicense](LICENSE). Third-party dependencies and bundled assets retain their respective licenses, including the [Lucide icons](next/internal/icons/LICENSE).
