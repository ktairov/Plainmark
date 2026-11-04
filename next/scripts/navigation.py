#!/usr/bin/env python3
"""Check ordinary and Cmd-click navigation using real native pointer events."""
import os
from pathlib import Path
import tempfile

from native_client import NativeApp

ROOT = Path(__file__).resolve().parents[1]


def tabs(app):
    return [n for n in app.tree() if n.get("label", "").startswith("Tab ")]


def active(app, title, count):
    nodes = tabs(app)
    return len(nodes) == count and any(n.get("label") == "Tab " + title and "checked" in n.get("states", []) for n in nodes)


def main():
    binary = Path(os.environ.get("PLAINMARK_BINARY", ROOT / "build/plainmark")).resolve()
    with tempfile.TemporaryDirectory(prefix="plainmark-navigation-") as temp:
        base = Path(temp)
        notes = base / "notes"
        notes.mkdir()
        for name, body in {"a": "# First\n\nFirst document", "b": "# Second\n\nFormula $x_i^2 + \\frac{1}{2}$", "c": "# Third\n\nThird document"}.items():
            (notes / (name + ".md")).write_text(body)
        app = NativeApp(binary, base, base / "config", notes)
        try:
            app.click("note:b.md")
            app.wait(lambda: active(app, "Second", 1), "plain click did not replace the clean tab")
            app.click("note:a.md", ["cmd"])
            app.wait(lambda: active(app, "First", 2), "Cmd-click did not add a tab")
            app.click("note:a.md", ["cmd"])
            assert active(app, "First", 2), "Cmd-click duplicated the current tab"
            app.click("note:c.md")
            app.wait(lambda: active(app, "Third", 2), "plain click changed the number of tabs")
            app.rpc("menu", path=["View", "Editor"])
            app.wait(lambda: any(n.get("key") == "editor" for n in app.tree()), "editor missing")
            app.rpc("act", key="editor", action="focus")
            app.rpc("input", events=[{"type": "key", "key": "CmdOrCtrl+A"}, {"type": "text", "text": "Unsaved third document"}])
            app.rpc("idle", timeout=10000)
            app.click("note:a.md")
            app.wait(lambda: active(app, "First", 3), "navigation did not retain the dirty tab")
            app.click("note:c.md")
            app.wait(lambda: any(n.get("key") == "editor" and "Unsaved third document" in n.get("value", "") for n in app.tree()), "dirty buffer was lost")
            app.rpc("menu", path=["Window", "Close All Tabs"])
            app.wait(lambda: any(n.get("label") == "Discard" for n in app.tree()), "dirty-close guard missing")
            discard = next(n for n in app.tree() if n.get("label") == "Discard")
            app.rpc("act", id=discard["id"], action="press")
            app.wait(lambda: not tabs(app), "discard did not close tabs")
            app.quit()
        finally:
            app.close()
        assert (notes / "c.md").read_text() == "# Third\n\nThird document", "discard overwrote the note"
        print("PASS: native plain click replaces, Cmd-click adds, existing tabs deduplicate, dirty draft survives, discard keeps disk unchanged")


if __name__ == "__main__":
    main()
