#!/usr/bin/env python3
"""Check actual move/resize, tab modes and both scroll panes across restarts."""
import json
import os
from pathlib import Path
import tempfile

from native_client import NativeApp

ROOT = Path(__file__).resolve().parents[1]


def wheel(app, x, y, distance):
    app.rpc("input", events=[{"type": "wheel", "x": x, "y": y, "dy": distance, "precise": True}])
    app.rpc("idle", timeout=10000)


def bounds(app):
    return app.rpc("windows")["windows"][0]["bounds"]


def mode(app, key):
    return any(n.get("key") == key and "checked" in n.get("states", []) for n in app.tree())


def has_position(app):
    return any(n.get("role") == "text" and n.get("label", "").startswith("Ln ") for n in app.tree())


def main():
    binary = Path(os.environ.get("PLAINMARK_BINARY", ROOT / "build/plainmark")).resolve()
    with tempfile.TemporaryDirectory(prefix="plainmark-restoration-") as temp:
        base = Path(temp)
        notes = base / "notes"
        notes.mkdir()
        for name, title in [("a", "First"), ("b", "Second")]:
            (notes / (name + ".md")).write_text("# " + title + "\n\n" + "\n\n".join(f"Paragraph {i}: readable long document for scroll restoration." for i in range(250)))
        config = base / "config"
        path = config / "state.json"
        expected_bounds = {"x": 93, "y": 81, "width": 1060, "height": 740}
        first = base / "first"
        first.mkdir()
        app = NativeApp(binary, first, config, notes)
        try:
            app.window_command(Bounds=expected_bounds)
            app.wait(lambda: bounds(app) == expected_bounds, "move/resize did not apply")
            assert not has_position(app), "preview shows line/column"
            wheel(app, 900, 400, 1300)
            app.menu("View", "Editor")
            app.wait(lambda: mode(app, "mode-pencil-line"), "editor mode did not select")
            assert has_position(app), "editor hides line/column"
            wheel(app, 900, 400, 950)
            app.click("note:b.md", ["cmd"])
            app.wait(lambda: any(n.get("label") == "Tab Second" and "checked" in n.get("states", []) for n in app.tree()), "second tab missing")
            app.menu("View", "Split View")
            app.wait(lambda: mode(app, "mode-columns-2"), "split mode missing")
            wheel(app, 460, 400, 710)
            wheel(app, 900, 400, 840)
            app.quit()
        finally:
            app.close()
        state = json.loads(path.read_text())
        expected = {t["path"]: t for t in state["roots"][str(notes.resolve())]["tabs"]}
        assert {k: state["window"][k] for k in expected_bounds} == expected_bounds, state["window"]
        assert expected["a.md"]["mode"] == "edit" and expected["b.md"]["mode"] == "split", expected
        for tab in expected.values():
            assert tab.get("previewY", 0) > 300 and tab.get("editorY", 0) > 300, expected
        second = base / "second"
        second.mkdir()
        app = NativeApp(binary, second, config)
        try:
            app.wait(lambda: bounds(app) == expected_bounds, "window geometry did not restore")
            app.wait(lambda: mode(app, "mode-columns-2"), "active split tab did not restore")
            app.click("note:a.md")
            app.wait(lambda: mode(app, "mode-pencil-line"), "inactive editor tab did not restore")
            app.menu("View", "Preview")
            app.wait(lambda: mode(app, "mode-eye"), "preview mode missing")
            assert not has_position(app), "line/column remained visible in preview"
            app.quit()
        finally:
            app.close()
        restored = json.loads(path.read_text())
        for tab in restored["roots"][str(notes.resolve())]["tabs"]:
            previous = expected[tab["path"]]
            for field in ("previewY", "editorY"):
                assert abs(tab.get(field, 0) - previous[field]) < 3, (tab["path"], field, previous, tab)
        assert restored["window"] == state["window"], (restored["window"], state["window"])
        third = base / "third"
        third.mkdir()
        app = NativeApp(binary, third, config)
        try:
            app.window_command(Maximized=True)
            app.wait(lambda: bounds(app)["width"] > expected_bounds["width"], "window did not maximize")
            maximized_bounds = bounds(app)
            app.quit()
        finally:
            app.close()
        maximized = json.loads(path.read_text())["window"]
        assert maximized.get("maximized"), maximized
        assert {k: maximized[k] for k in expected_bounds} == expected_bounds, "maximizing overwrote normal bounds"
        fourth = base / "fourth"
        fourth.mkdir()
        app = NativeApp(binary, fourth, config)
        try:
            app.wait(lambda: bounds(app) == maximized_bounds, "maximized state did not restore")
            app.window_command(Maximized=False)
            app.wait(lambda: bounds(app) == expected_bounds, "unmaximizing did not restore normal position/size")
            app.quit()
        finally:
            app.close()
        print("PASS: move/resize/maximize and restart, normal geometry after unmaximize, active/inactive tab modes, editor/preview scroll per tab, preview hides Ln/Col")


if __name__ == "__main__":
    main()
