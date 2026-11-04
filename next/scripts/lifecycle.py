#!/usr/bin/env python3
"""Check restart restoration and clean Quit in a real MyGo app, outside the repo."""
import json
import os
from pathlib import Path
import tempfile

from native_client import NativeApp

ROOT = Path(__file__).resolve().parents[1]


def main():
    binary = Path(os.environ.get("PLAINMARK_BINARY", str(ROOT / "build/plainmark"))).resolve()
    with tempfile.TemporaryDirectory(prefix="plainmark-lifecycle-") as temp:
        base = Path(temp)
        notes = base / "notes"
        notes.mkdir()
        (notes / "a.md").write_text("# First\n\nFirst document\n", encoding="utf-8")
        (notes / "b.md").write_text("# Second\n\nSecond document\n", encoding="utf-8")
        config = base / "config"
        first = base / "first"
        first.mkdir()
        app = NativeApp(binary, first, config, path=notes, cwd=base)
        try:
            app.wait(lambda: any(n.get("key") == "note:b.md" for n in app.tree()), "sidebar note missing")
            app.click("note:b.md", ["cmd"])
            app.wait(lambda: any(n.get("label") == "Tab Second" and "checked" in n.get("states", []) for n in app.tree()), "second tab did not activate")
            app.menu("View", "Editor")
            app.wait(lambda: any(n.get("key") == "editor" for n in app.tree()), "editor did not open")
            app.menu("Note", "Pin")
            app.menu("View", "Hide Sidebar")
            app.rpc("idle", timeout=10000)
            app.quit()
        finally:
            app.close()
        state = json.loads((config / "state.json").read_text())
        root = str(notes.resolve())
        assert state["root"] == root and state["version"] == 1, state
        assert state["roots"][root]["active"] == "b.md", state
        assert [t["path"] for t in state["roots"][root]["tabs"]] == ["a.md", "b.md"], state
        assert state["roots"][root]["tabs"][1]["mode"] == "edit", state
        assert state["roots"][root]["pinned"] == ["b.md"] and not state["sidebarVisible"], state
        assert "First document" not in (config / "state.json").read_text(), "document body leaked into settings"
        second = base / "second"
        second.mkdir()
        app = NativeApp(binary, second, config, cwd=base)
        try:
            nodes = app.tree()
            assert len([n for n in nodes if n.get("label", "").startswith("Tab ")]) == 2, nodes
            assert not any(n.get("role") == "textField" and "search" in n.get("states", []) for n in nodes), "sidebar visibility not restored"
            assert any(n.get("key") == "editor" and "Second document" in n.get("value", "") for n in nodes), "active note/mode did not restore from disk"
            app.rpc("act", key="editor", action="focus")
            app.rpc("input", events=[{"type": "key", "key": "CmdOrCtrl+A"}, {"type": "text", "text": "# Unsaved lifecycle draft"}])
            app.rpc("idle", timeout=10000)
            app.menu("Plainmark Go", "Quit Plainmark")
            app.wait(lambda: any(n.get("label") == "Unsaved changes" for n in app.tree()), "Quit did not protect dirty document")
            cancel = next(n for n in app.tree() if n.get("label") == "Cancel" and n.get("role") == "button")
            app.rpc("act", id=cancel["id"], action="press")
            app.wait(lambda: not any(n.get("label") == "Unsaved changes" for n in app.tree()), "Cancel did not close the quit prompt")
            assert app.process.poll() is None, "Cancel quit terminated the app"
            app.menu("Window", "Close All Tabs")
            app.wait(lambda: any(n.get("label") == "Unsaved changes" for n in app.tree()), "Close all did not ask about the draft")
            discard = next(n for n in app.tree() if n.get("label") == "Discard" and n.get("role") == "button")
            app.rpc("act", id=discard["id"], action="press")
            app.wait(lambda: not any(n.get("label", "").startswith("Tab ") for n in app.tree()), "tabs did not close")
            app.quit()
        finally:
            app.close()
        third = base / "third"
        third.mkdir()
        app = NativeApp(binary, third, config, cwd=base)
        try:
            assert not any(n.get("label", "").startswith("Tab ") for n in app.tree()), "empty session was not restored"
            app.quit()
        finally:
            app.close()
        assert (notes / "b.md").read_text(encoding="utf-8") == "# Second\n\nSecond document\n", "discard wrote dirty document"
        print("PASS: clean Quit, dirty Quit/Cancel, discard, metadata flush, restart without arguments, tabs/mode/pin/sidebar restoration, empty session, external cwd")


if __name__ == "__main__":
    main()
