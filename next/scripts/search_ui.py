#!/usr/bin/env python3
"""Check preview search arrows with real pointer input and distant matches."""
import base64
import os
from pathlib import Path
import tempfile

from native_client import NativeApp

ROOT = Path(__file__).resolve().parents[1]


def main():
    binary = Path(os.environ.get("PLAINMARK_BINARY", ROOT / "build/plainmark")).resolve()
    with tempfile.TemporaryDirectory(prefix="plainmark-search-") as temp:
        base = Path(temp)
        notes = base / "notes"
        notes.mkdir()
        source = "# Search\n\nFirst needle here.\n\n" + "Filler paragraph.\n\n" * 100 + "Last needle here.\n"
        note = notes / "search.md"
        note.write_text(source)
        app = NativeApp(binary, base, base / "config", notes)
        try:
            app.click("note:search.md")
            app.menu("View", "Preview")
            app.rpc("idle", timeout=10000)
            app.menu("Edit", "Find in Note…")

            def field():
                return next((n for n in app.tree() if n.get("role") == "textField" and n.get("label") == "Find"), None)

            app.wait(field, "search field did not open")
            app.rpc("act", id=field()["id"], action="setValue", text="needle")

            def visible(text):
                height = app.rpc("windows")["windows"][0]["bounds"]["height"]
                return any(n.get("label") == text and n["bounds"]["height"] >= 26
                           and n["bounds"]["y"] >= 50
                           and n["bounds"]["y"] + n["bounds"]["height"] <= height
                           for n in app.tree())

            app.wait(lambda: visible("First needle here."), "initial match is off screen")
            for button, text in [("Next", "Last needle here."), ("Previous", "First needle here."), ("Previous", "Last needle here.")]:
                node = next(n for n in app.tree() if n.get("role") == "button" and n.get("label") == button)
                box = node["bounds"]
                app.rpc("input", events=[{"type": "click", "x": box["x"] + box["width"] / 2, "y": box["y"] + box["height"] / 2}])
                app.wait(lambda: visible(text), button + " did not reveal " + text)
            output = ROOT / "build/previews/search-native.png"
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_bytes(base64.b64decode(app.rpc("capture")["png"]))
            assert note.read_text() == source, "search changed the note"
            app.quit()
        finally:
            app.close()
    print("PASS: native search arrows reveal distant matches, reverse direction and wrap; note unchanged")


if __name__ == "__main__":
    main()
