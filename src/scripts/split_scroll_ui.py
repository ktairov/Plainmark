#!/usr/bin/env python3
"""Check both directions of split scrolling with native wheel and thumb input."""
import json
import os
from pathlib import Path
import tempfile

from native_client import NativeApp

ROOT = Path(__file__).resolve().parents[1]


def main():
    binary = Path(os.environ.get("PLAINMARK_BINARY", ROOT / "build/plainmark")).resolve()
    with tempfile.TemporaryDirectory(prefix="plainmark-split-") as temp:
        base = Path(temp)
        notes = base / "notes"
        notes.mkdir()
        source = "".join(f"# Section {i:02}\n\nParagraph with **formatting** and some text.\n\n" for i in range(80))
        note = notes / "split.md"
        note.write_text(source)
        config = base / "config"
        app = NativeApp(binary, base, config, notes)
        try:
            app.click("note:split.md")
            app.menu("View", "Split View")
            app.wait(lambda: any(n.get("key") == "editor" for n in app.tree()), "split editor missing")

            def positions():
                state = json.loads((config / "state.json").read_text())
                tab = state["roots"][state["root"]]["tabs"][0]
                return tab.get("editorY", 0), tab.get("previewY", 0)

            def wheel(x, y, dy):
                app.rpc("input", events=[{"type": "wheel", "x": x, "y": y, "dy": dy, "precise": True}])
                app.rpc("idle", timeout=10000)

            editor = next(n["bounds"] for n in app.tree() if n.get("key") == "editor")
            right = next(n["bounds"] for n in app.tree() if n.get("role") == "scroll" and n["bounds"]["x"] >= editor["x"] + editor["width"] and n["bounds"]["height"] > 200)
            ex, ey = editor["x"] + 60, editor["y"] + 200
            px, py = right["x"] + 80, right["y"] + 200
            app.wait(lambda: (config / "state.json").exists(), "state was not saved")
            wheel(ex, ey, 800)
            app.wait(lambda: min(positions()) > 400, "left wheel did not scroll both panes")
            previous = positions()
            wheel(px, py, 600)
            app.wait(lambda: all(a > b + 200 for a, b in zip(positions(), previous)), "right wheel did not scroll both panes")

            for pane in (editor, right):
                wheel(px, py, -100000)
                app.wait(lambda: max(positions()) < 1, "top did not synchronize")
                x, y = pane["x"] + pane["width"] - 5, pane["y"] + 12
                app.rpc("input", events=[
                    {"type": "pointerMove", "x": x, "y": y},
                    {"type": "pointerDown", "x": x, "y": y},
                    {"type": "pointerMove", "x": x, "y": y + 180},
                    {"type": "pointerUp", "x": x, "y": y + 180},
                ])
                app.rpc("idle", timeout=10000)
                app.wait(lambda: min(positions()) > 400, "scrollbar drag did not scroll both panes")
            assert note.read_text() == source, "scrolling changed the note"
            app.quit()
        finally:
            app.close()
    print("PASS: split wheel and scrollbar drag synchronize both panes; note unchanged")


if __name__ == "__main__":
    main()
