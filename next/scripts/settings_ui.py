#!/usr/bin/env python3
"""Exercise the settings UI, persistence and cancellation in real native windows."""
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


def dialog_open(app):
    return any(n.get("role") == "dialog" for n in app.tree())


def capture(app, name):
    directory = os.environ.get("PLAINMARK_SNAPSHOTS")
    if directory:
        output = Path(directory)
        output.mkdir(parents=True, exist_ok=True)
        (output / name).write_bytes(base64.b64decode(app.rpc("capture")["png"]))


def main():
    binary = Path(os.environ.get("PLAINMARK_BINARY", ROOT / "build/plainmark")).resolve()
    with tempfile.TemporaryDirectory(prefix="plainmark-settings-") as temp:
        base = Path(temp)
        config = base / "config"
        notes = base / "notes"
        notes.mkdir()
        (notes / "note.md").write_text("# Settings preview\n\nFormula $x_i^2$\n")
        first = base / "first"
        first.mkdir()
        app = NativeApp(binary, first, config, notes)
        try:
            app.menu("Plainmark Go", "Settings…")
            app.wait(lambda: dialog_open(app), "application menu did not open settings")
            press(app, "settings-section-theme")
            app.wait(lambda: any(n.get("key") == "theme-light-nord" for n in app.tree()), "theme cards missing")
            press(app, "theme-light-nord")
            app.wait(lambda: any(n.get("key") == "theme-light-nord" and "checked" in n.get("states", []) for n in app.tree()), "palette did not select")
            capture(app, "settings-native-themes.png")
            press(app, "apply-settings")
            app.wait(lambda: not dialog_open(app), "Apply did not close settings")
            app.quit()
        finally:
            app.close()
        path = config / "state.json"
        state = json.loads(path.read_text())
        assert state["preferences"]["dayTheme"] == "nord", state
        assert state["preferences"]["colorMode"] == "system", "palette preview persisted a temporary mode"
        state["preferences"].update(language="ru", colorMode="light")
        state["window"].update(width=880, height=620)
        path.write_text(json.dumps(state))
        before = state["preferences"].copy()
        second = base / "second"
        second.mkdir()
        app = NativeApp(binary, second, config)
        try:
            app.menu("Plainmark Go", "Настройки…")
            app.wait(lambda: dialog_open(app), "settings did not open after restart")
            capture(app, "settings-native-ru-narrow.png")
            press(app, "settings-section-language")
            app.wait(lambda: any(n.get("key") == "settings-language-select" for n in app.tree()), "language select missing")
            capture(app, "settings-native-language.png")
            press(app, "settings-language-select")
            app.wait(lambda: any(n.get("key") == "settings-language-en" for n in app.tree()), "language options missing")
            capture(app, "settings-native-language-popup.png")
            app.rpc("input", events=[{"type": "key", "key": "Escape"}])
            app.wait(lambda: not any(n.get("key") == "settings-language-en" for n in app.tree()), "Escape did not close language popup")
            assert dialog_open(app), "closing the language popup also closed settings"
            press(app, "settings-language-select")
            app.wait(lambda: any(n.get("key") == "settings-language-en" for n in app.tree()), "language popup did not reopen")
            press(app, "settings-language-en")
            app.wait(lambda: any(n.get("role") == "dialog" and n.get("label") == "Settings" for n in app.tree()), "language preview failed")
            press(app, "cancel-settings")
            app.wait(lambda: not dialog_open(app), "Cancel did not close settings")
            app.menu("Plainmark Go", "Настройки…")
            app.wait(lambda: any(n.get("role") == "dialog" and n.get("label") == "Настройки" for n in app.tree()), "Cancel did not restore Russian")
            app.rpc("input", events=[{"type": "key", "key": "Escape"}])
            app.wait(lambda: not dialog_open(app), "Escape did not close settings")
            app.quit("Выйти из Plainmark")
        finally:
            app.close()
        assert json.loads(path.read_text())["preferences"] == before, "cancelled preview was persisted"
        print("PASS: application menu, sections, palette preview/apply/restart, language preview/cancel, Escape, Russian UI at 880px")


if __name__ == "__main__":
    main()
