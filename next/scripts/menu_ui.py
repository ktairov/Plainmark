#!/usr/bin/env python3
"""Check the original menu commands and themed context menu in a native window."""
import base64
import os
from pathlib import Path
import tempfile

from native_client import NativeApp

ROOT = Path(__file__).resolve().parents[1]


def main():
    binary = Path(os.environ.get("PLAINMARK_BINARY", ROOT / "build/plainmark")).resolve()
    with tempfile.TemporaryDirectory(prefix="plainmark-menu-") as temp:
        base = Path(temp)
        notes = base / "notes"
        notes.mkdir()
        (notes / "a.md").write_text("# First\n\nFirst document")
        (notes / "b.md").write_text("# Second\n\nSecond document")
        app = NativeApp(binary, base, base / "config", notes)
        try:
            app.menu("Plainmark Go", "Settings…")
            app.wait(lambda: any(n.get("role") == "dialog" for n in app.tree()), "Settings is missing from the application menu")
            app.rpc("act", key="cancel-settings", action="press")
            app.wait(lambda: not any(n.get("role") == "dialog" for n in app.tree()), "settings did not close")
            app.menu("View", "Hide Sidebar")
            visible = lambda: any(n.get("role") == "textField" and "search" in n.get("states", []) for n in app.tree())
            app.wait(lambda: not visible(), "Hide Sidebar failed")
            app.menu("View", "Show Sidebar")
            app.wait(visible, "Show Sidebar label/action failed")
            app.menu("View", "Editor")
            app.wait(lambda: any(n.get("key") == "editor" for n in app.tree()), "Editor command failed")
            app.menu("View", "Preview")
            app.rpc("idle", timeout=10000)
            app.menu("Edit", "Find in Note…")
            app.wait(lambda: any(n.get("role") == "textField" and n.get("label") == "Find" for n in app.tree()), "Find did not move to Edit menu")
            app.rpc("input", events=[{"type": "key", "key": "Escape"}])
            app.menu("View", "Preview")
            app.rpc("idle", timeout=10000)
            app.menu("View", "Preview")  # Reselecting keeps the current mode checked.
            app.rpc("idle", timeout=10000)
            node = next(n for n in app.tree() if n.get("key") == "note:b.md")
            box = node["bounds"]
            app.rpc("input", events=[{"type": "click", "x": box["x"] + box["width"] / 2, "y": box["y"] + box["height"] / 2, "button": "right"}])
            app.wait(lambda: any(n.get("role") == "menuItem" and n.get("label") == "Open in new tab" for n in app.tree()), "themed note context menu missing")
            items = [n for n in app.tree() if n.get("role") == "menuItem"]
            assert [n.get("label") for n in items] == ["Copy path", "Open in new tab", "Pin", "Rename", "Move", "Move to Trash…"], items
            output = os.environ.get("PLAINMARK_SNAPSHOTS")
            if output:
                path = Path(output)
                path.mkdir(parents=True, exist_ok=True)
                (path / "context-menu.png").write_bytes(base64.b64decode(app.rpc("capture")["png"]))
            entry = next(n for n in items if n.get("label") == "Open in new tab")
            app.rpc("act", id=entry["id"], action="press")
            app.wait(lambda: len([n for n in app.tree() if n.get("label", "").startswith("Tab ")]) == 2, "context action replaced the current tab")
            app.menu("Note", "Pin")
            app.rpc("idle", timeout=10000)
            app.menu("Note", "Unpin")
            app.menu("Window", "Previous Tab")
            app.wait(lambda: any(n.get("label") == "Tab First" and "checked" in n.get("states", []) for n in app.tree()), "Previous Tab failed")
            app.menu("Window", "Close Other Tabs")
            app.wait(lambda: len([n for n in app.tree() if n.get("label", "").startswith("Tab ")]) == 1, "Close Other Tabs failed")
            app.menu("Plainmark Go", "Settings…")
            app.wait(lambda: any(n.get("role") == "dialog" for n in app.tree()), "settings missing")
            app.rpc("act", key="settings-section-language", action="press")
            app.wait(lambda: any(n.get("key") == "settings-language-select" for n in app.tree()), "language section missing")
            app.rpc("act", key="settings-language-select", action="press")
            app.wait(lambda: any(n.get("key") == "settings-language-ru" for n in app.tree()), "language options missing")
            app.rpc("act", key="settings-language-ru", action="press")
            app.rpc("idle", timeout=10000)
            app.rpc("act", key="apply-settings", action="press")
            app.wait(lambda: not any(n.get("role") == "dialog" for n in app.tree()), "Apply did not close settings")
            app.menu("Вид", "Скрыть боковую панель")
            app.wait(lambda: not visible(), "Russian Hide Sidebar failed")
            app.menu("Вид", "Показать боковую панель")
            app.wait(visible, "Russian Show Sidebar failed")
            app.quit("Выйти из Plainmark")
        finally:
            app.close()
        print("PASS: app Settings, Edit Find, View modes/sidebar, Note Pin/Unpin, Window tab commands, themed context order/open-in-new-tab, Russian menu switch, clean Quit")


if __name__ == "__main__":
    main()
