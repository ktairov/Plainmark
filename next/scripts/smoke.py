#!/usr/bin/env python3
"""Exercise the real native window. Requires build/plainmark built with -tags e2e."""
import base64
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]


def main():
    with tempfile.TemporaryDirectory(prefix="plainmark-smoke-") as temp:
        directory = Path(temp)
        note = directory / "note.md"
        note.write_text("# Исходный\n\nFormula $x_i^2$\n", encoding="utf-8")
        sockpath = directory / "app.sock"
        process = subprocess.Popen(
            [str(ROOT / "build/plainmark"), str(note)],
            cwd=ROOT,
            env={**os.environ, "MYGO_AUTOMATION": str(sockpath), "MYGO_LOCALE": "en"},
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        )
        try:
            deadline = time.monotonic() + 30
            while not sockpath.exists():
                if process.poll() is not None:
                    raise RuntimeError(process.stdout.read().decode())
                if time.monotonic() > deadline:
                    raise TimeoutError("live app did not create automation socket")
                time.sleep(.05)
            with socket.socket(socket.AF_UNIX) as client:
                client.settimeout(15)
                client.connect(str(sockpath))
                stream = client.makefile("rwb")
                request_id = 0

                def rpc(method, **params):
                    nonlocal request_id
                    request_id += 1
                    stream.write((json.dumps({"id": request_id, "method": method, "params": params}) + "\n").encode())
                    stream.flush()
                    response = json.loads(stream.readline())
                    if "error" in response:
                        raise RuntimeError(response)
                    return response["result"]

                deadline = time.monotonic() + 30
                windows = rpc("windows")["windows"]
                while not windows:
                    if process.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("app did not create a window")
                    time.sleep(.05)
                    windows = rpc("windows")["windows"]
                rpc("idle", timeout=10000)
                assert len(windows) == 1 and windows[0]["kind"] == "content", windows
                nodes = rpc("tree")["nodes"]
                split = next(n for n in nodes if n.get("label") == "Split" and "press" in n.get("actions", []))
                rpc("act", id=split["id"], action="press")
                deadline = time.monotonic() + 15
                while True:
                    rpc("idle", timeout=10000)
                    nodes = rpc("tree")["nodes"]
                    if any(n.get("key") == "editor" and "disabled" not in n.get("states", []) for n in nodes):
                        break
                    if time.monotonic() > deadline:
                        raise TimeoutError("editor did not become ready")
                rpc("act", key="editor", action="focus")
                rpc("idle", timeout=10000)
                rpc("input", events=[{"type": "key", "key": "CmdOrCtrl+A"}, {"type": "text", "text": "# Проверено\n\nFormula $x_i^2$\n"}])
                rpc("idle", timeout=10000)
                rpc("input", events=[{"type": "key", "key": "CmdOrCtrl+Z"}])
                rpc("idle", timeout=10000)
                assert "Исходный" in next(n for n in rpc("tree")["nodes"] if n.get("key") == "editor")["value"]
                rpc("input", events=[{"type": "key", "key": "CmdOrCtrl+Shift+Z"}])
                rpc("idle", timeout=10000)
                rpc("act", key="save", action="press")
                deadline = time.monotonic() + 15
                while True:
                    rpc("idle", timeout=10000)
                    if "Проверено" in note.read_text(encoding="utf-8"):
                        break
                    if time.monotonic() > deadline:
                        raise TimeoutError("save did not persist the Unicode edit")
                nodes = rpc("tree")["nodes"]
                assert any(n.get("role") == "heading" and n.get("label") == "Проверено" for n in nodes)
                # Select a whole line and exercise the native edit menu/clipboard.
                rpc("act", key="editor", action="selectText", **{"from": 0, "to": 11})
                rpc("menu", path=["Edit", "Copy"])
                rpc("idle", timeout=10000)

                def wait_tree(predicate, description):
                    deadline = time.monotonic() + 15
                    while True:
                        rpc("idle", timeout=10000)
                        nodes = rpc("tree")["nodes"]
                        if predicate(nodes):
                            return nodes
                        if time.monotonic() > deadline:
                            raise TimeoutError(description)
                        time.sleep(.02)

                def editor_value(nodes):
                    return next((n.get("value", "") for n in nodes if n.get("key") == "editor"), "")

                note.write_text("# Внешний\n\nUpdated externally\n", encoding="utf-8")
                wait_tree(lambda nodes: "Внешний" in editor_value(nodes), "watcher did not reload clean document")
                rpc("act", key="editor", action="focus")
                rpc("input", events=[{"type": "key", "key": "CmdOrCtrl+A"}, {"type": "text", "text": "# Локальный\n\nKeep this draft\n"}])
                rpc("idle", timeout=10000)
                note.write_text("# Конфликт\n\nExternal conflict\n", encoding="utf-8")
                nodes = wait_tree(lambda nodes: any(n.get("label") == "File changed on disk. Your draft is kept." for n in nodes), "watcher did not report conflict")
                assert "Локальный" in editor_value(nodes), "external change destroyed draft"
                overwrite = next(n for n in nodes if n.get("label") == "Overwrite")
                rpc("act", id=overwrite["id"], action="press")
                wait_tree(lambda nodes: "Локальный" in note.read_text(encoding="utf-8") and not any(n.get("label") == "Overwrite" for n in nodes), "overwrite did not resolve conflict")
                fixture = "enc:v1:AAECAwQFBgcICQoLDA0ODw:EBESExQVFhcYGRobHB0eHyAhIiMkJSYn:kjoOIIaoKTbcrT789T4USc1ZP4jD-siNCzoQtC-I0x8dWFRt2OBpKg"
                note.write_text("# Crypto smoke\n\n```encrypted\n" + fixture + "\n```\n", encoding="utf-8")
                nodes = wait_tree(lambda nodes: any(n.get("label") == "Unlock encrypted block" for n in nodes), "encrypted preview block missing")
                unlock = next(n for n in nodes if n.get("label") == "Unlock encrypted block")
                rpc("act", id=unlock["id"], action="press")
                rpc("idle", timeout=10000)
                rpc("act", key="crypto-password", action="setValue", text="archive-pass")
                rpc("act", key="crypto-submit", action="press")
                nodes = wait_tree(lambda nodes: any(n.get("label") == "Lock" for n in nodes), "Rust fixture did not unlock in actual window")
                lock = next(n for n in nodes if n.get("label") == "Lock")
                assert "# Legacy secret\nold note" in [n.get("label") for n in nodes], "plaintext is not displayed"
                rpc("act", id=lock["id"], action="press")
                wait_tree(lambda nodes: not any(n.get("label") == "Lock" for n in nodes), "block did not lock")
                assert "# Legacy secret" not in note.read_text(encoding="utf-8"), "preview wrote plaintext to file"
                capture = rpc("capture")
                output = os.environ.get("PLAINMARK_SMOKE_CAPTURE")
                if output:
                    Path(output).write_bytes(base64.b64decode(capture["png"]))
                print("PASS: native window, Unicode, undo/redo, save, preview, Copy, watcher, conflict/overwrite, Rust fixture unlock/lock and capture")
        finally:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=10)


if __name__ == "__main__":
    main()
