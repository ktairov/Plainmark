# Tauri 2 + Rust → Go + MyGo: an LLM migration guide

Migrate user workflows with **functional parity**. Rebuild the UI using native controls; preserve visual details where they affect usability. Plainmark is a reference implementation, not an automatic conversion template.

## Inspect and define the contract

Read the frontend, Rust commands, events, plugins, capabilities, menus, and persisted formats. Record **user action → behavior and failure cases → Go owner → verification**. Preserve the original in Git and capture fixtures before rewriting.

Plainmark's original is `cb2275f`:

```sh
git show cb2275f:src/api.ts
git show cb2275f:src-tauri/src/workspace.rs
git show cb2275f:src-tauri/src/crypto.rs
```

Existing Markdown folders open directly. Do not invent a data importer or settings/session migration unless the product requires one.

## Map responsibilities

| Existing responsibility | Go/MyGo reference |
| --- | --- |
| Rust file commands, validation, saving | [workspace](src/internal/workspace/) |
| Frontend tabs, drafts and conflicts | [session](src/internal/session/) |
| React UI, hooks and IPC orchestration | [desktop](src/internal/desktop/) |
| Tauri file-change events | [watcher](src/internal/workspace/watch.go) |
| Browser Markdown and KaTeX | [preview](src/internal/preview/), MyGo Markdown, [math adapter](src/internal/mathrender/) |
| Settings and session/window restore | [settings](src/internal/settings/), [persistence](src/internal/desktop/persistence.go) |
| Versioned Rust encrypted payloads | [cryptoblock](src/internal/cryptoblock/) |
| Menus, dialogs, clipboard and packaging | MyGo APIs, platform adapters, [mygo.json](src/mygo.json) |

Keep domain logic independent of widgets. Replace `invoke` with typed Go calls and listeners with owned goroutines/channels, rather than recreating an IPC layer.

## Implement in working slices

1. **Prove the stack.** Pin dependencies. Build an editor/preview prototype with Unicode, clipboard and representative formulas. Markdown math syntax is not a complete renderer: Plainmark adds baseline-aware TeX-to-SVG rendering.
2. **Port the core.** Test paths/symlinks, scanning, file operations, atomic saves, revision conflicts, watching and encryption against original fixtures.
3. **Add workflows.** Keep buffers, undo, dirty state, modes and scrolling per document. Implement navigation, independent new-tab requests, save/discard/close, and external-change handling.
4. **Connect native services.** Add menus, shortcuts, dialogs, themes, settings and restart restoration. Plainmark persists preferences/navigation, not drafts, passwords or decrypted blocks.
5. **Refine and deliver.** Reuse assets/translations where appropriate. Verify rendering and interactions, then test the packaged app outside the repository. Remove legacy sources only after acceptance; retain their Git reference and fixtures.

## Avoid known migration failures

- **UI ownership:** mutate UI state through `Window.Update`; move I/O off the UI thread. Use stable element identities, cache parsing/rendering, and copy full drafts only when `Buffer.Version()` changes.
- **Cancellation:** supersede obsolete current-note replacements, not independent new-tab reads. Reject old-workspace results, prevent background focus stealing, and stop/join watchers when switching roots.
- **Persistence:** queued writes are not successful saves. Confirm disk writes, retry failures, and flush on shutdown. Preserve versioned encryption formats; cross-test both languages. Go strings do not provide Rust `zeroize` guarantees.
- **Geometry:** test actual input, visibility and screenshots. Search reveal targets must survive rebuilds. Map split scrolling by source lines/rendered blocks without feedback loops. Check font advances, box-drawing continuity and list markers; search must preserve code-block dimensions.
- **Platforms:** in MyGo v0.7.4, `ui.Cmd` maps to Command on macOS and Control elsewhere; menus use `CmdOrCtrl`. Put Settings in a regular Linux/Windows menu. Restrict macOS Ctrl-click context behavior to macOS. Native system menus are not CSS-styled components.
- **Themes:** separate persisted appearance from temporary window overrides, and keep receiving system theme events.

## Verify and hand off

From `src/`:

```sh
go test ./...
go vet ./...
go test -race ./internal/desktop
```

Use [crypto_compat.py](src/scripts/crypto_compat.py) for Rust ↔ Go compatibility. Build with `go build -tags e2e -o build/plainmark ./cmd/plainmark`, then run relevant [native scripts](src/scripts/) sequentially with temporary notes and an isolated `PLAINMARK_CONFIG_DIR`. Exercise pointer/keyboard input, not just state setters.

Package through `make build` or `go run ./tools/buildapp` to reserve the next build number. Follow [AGENTS.md](AGENTS.md): Linux/Windows builds require an explicit request. Reference versions are MyGo v0.7.4 and `go-tex/math` v0.50.0; re-check APIs when choosing different versions.

Report workflows implemented, checks actually run, and remaining gaps. Successful cross-compilation does not prove native GUI parity. Plainmark has macOS GUI verification; Linux and Windows still require target-system validation.
