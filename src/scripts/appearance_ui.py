#!/usr/bin/env python3
"""Check local theme toggles, source-sized action buttons and styled dialogs."""
import base64
import json
import os
from pathlib import Path
import tempfile

from native_client import NativeApp

ROOT = Path(__file__).resolve().parents[1]


def press(app, key):
    app.rpc("act", key=key, action="press")
    app.rpc("idle", timeout=10000)


def capture(app, name):
    if directory := os.environ.get("PLAINMARK_SNAPSHOTS"):
        path = Path(directory)
        path.mkdir(parents=True, exist_ok=True)
        (path / name).write_bytes(base64.b64decode(app.rpc("capture")["png"]))


def button_size(app, key):
    node = next(n for n in app.tree() if n.get("key") == key)
    assert node["bounds"]["height"] == 36, (key, node)


def main():
    binary = Path(os.environ.get("PLAINMARK_BINARY", ROOT / "build/plainmark")).resolve()
    with tempfile.TemporaryDirectory(prefix="plainmark-appearance-") as temp:
        base = Path(temp)
        notes = base / "notes"
        notes.mkdir()
        (notes / "a.md").write_text("# Example\n\nSome text to read and select.\n")
        config = base / "config"
        config.mkdir()
        (config / "state.json").write_text(json.dumps({"version": 1, "preferences": {"language": "en", "colorMode": "light"}}))
        first = base / "first"
        first.mkdir()
        app = NativeApp(binary, first, config, notes)
        try:
            assert not any(n.get("key") in ("settings", "more") for n in app.tree()), "removed toolbar buttons remain"
            press(app, "toggle-color")
            capture(app, "window-local-night.png")
            app.menu("Plainmark", "Settings…")
            app.wait(lambda: any(n.get("key") == "apply-settings" for n in app.tree()), "settings missing")
            for key in ("apply-settings", "cancel-settings", "reset-settings"):
                button_size(app, key)
            capture(app, "settings-original-appearance.png")
            press(app, "cancel-settings")
            app.wait(lambda: not any(n.get("role") == "dialog" for n in app.tree()), "settings did not close")
            app.menu("File", "New Note")
            app.wait(lambda: any(n.get("key") == "path-name" for n in app.tree()), "new-note dialog missing")
            button_size(app, "confirm-path")
            capture(app, "dialog-new-note.png")
            press(app, "cancel-path")
            app.wait(lambda: not any(n.get("role") == "dialog" for n in app.tree()), "new-note dialog did not close")
            app.menu("Note", "Move to Trash…")
            app.wait(lambda: any(n.get("key") == "confirm-trash" for n in app.tree()), "trash dialog missing")
            button_size(app, "confirm-trash")
            capture(app, "dialog-trash.png")
            press(app, "cancel-trash")
            app.wait(lambda: not any(n.get("role") == "dialog" for n in app.tree()), "trash dialog did not close")
            app.menu("View", "Editor")
            app.wait(lambda: any(n.get("key") == "editor" for n in app.tree()), "editor missing")
            app.rpc("act", key="editor", action="focus")
            app.rpc("input", events=[{"type": "key", "key": "CmdOrCtrl+A"}])
            app.rpc("idle", timeout=10000)
            app.menu("Note", "Encrypt Selection…")
            app.wait(lambda: any(n.get("key") == "crypto-password" for n in app.tree()), "crypto dialog missing")
            button_size(app, "crypto-submit")
            capture(app, "dialog-encrypt.png")
            press(app, "crypto-cancel")
            app.wait(lambda: not any(n.get("role") == "dialog" for n in app.tree()), "crypto dialog did not close")
            app.quit()
        finally:
            app.close()
        state = json.loads((config / "state.json").read_text())
        assert state["preferences"]["colorMode"] == "light", "toolbar toggle changed the configured mode"
        second = base / "second"
        second.mkdir()
        app = NativeApp(binary, second, config)
        try:
            capture(app, "window-configured-day-restart.png")
            app.quit()
        finally:
            app.close()
        assert list(notes.iterdir()) == [notes / "a.md"], "cancelled dialog modified the workspace"
        print("PASS: toolbar cleanup, local day/night without preference changes, 36px Apply/Cancel/actions, new-note/trash/encrypt dialogs and cancellation")


if __name__ == "__main__":
    main()
