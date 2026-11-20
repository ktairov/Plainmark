#!/usr/bin/env python3
"""Check window zoom from tab titles and the whole tab strip using pointer input."""
import os
from pathlib import Path
import tempfile

from native_client import NativeApp

ROOT = Path(__file__).resolve().parents[1]


def bounds(app):
    return app.rpc("windows")["windows"][0]["bounds"]


def node(app, label):
    return next(n for n in app.tree() if n.get("label") == label)


def click(app, x, y, count=1):
    app.rpc("input", events=[{"type": "click", "x": x, "y": y, "clicks": count}])
    app.rpc("idle", timeout=10000)


def main():
    binary = Path(os.environ.get("PLAINMARK_BINARY", ROOT / "build/plainmark")).resolve()
    with tempfile.TemporaryDirectory(prefix="plainmark-titlebar-") as temp:
        base = Path(temp)
        notes = base / "notes"
        notes.mkdir()
        (notes / "a.md").write_text("# First\n\nFirst document")
        (notes / "b.md").write_text("# Second\n\nSecond document")
        app = NativeApp(binary, base, base / "config", notes)
        normal = {"x": 93, "y": 81, "width": 1060, "height": 740}
        try:
            app.window_command(Bounds=normal)
            app.wait(lambda: bounds(app) == normal, "normal bounds did not apply")
            app.click("note:b.md", ["cmd"])
            app.wait(lambda: any(n.get("label") == "Tab Second" for n in app.tree()), "second tab missing")

            def tab_point():
                box = node(app, "Tab First")["bounds"]
                return box["x"] + box["width"] / 2, box["y"] + box["height"] / 2

            click(app, *tab_point())
            assert "checked" in node(app, "Tab First").get("states", []), "single click did not select the tab"
            assert bounds(app) == normal, "single click resized the window"
            for point in (
                tab_point,
                lambda: (next(n for n in app.tree() if n.get("key") == "tab-strip")["bounds"]["x"] + 500, 20),
                lambda: (node(app, "Tab First")["bounds"]["x"] + 20, 3),
            ):
                click(app, *point(), 2)
                app.wait(lambda: bounds(app)["width"] > normal["width"], "header double click did not maximize")
                click(app, *point(), 2)
                app.wait(lambda: bounds(app) == normal, "second double click did not restore the window")
            assert len([n for n in app.tree() if n.get("label", "").startswith("Tab ")]) == 2, "double click changed the tabs"
            app.quit()
        finally:
            app.close()
    print("PASS: single-click tab selection, double-click tab title, empty strip and header padding maximize/restore")


if __name__ == "__main__":
    main()
