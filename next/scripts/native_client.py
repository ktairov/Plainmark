"""Small automation client shared by native integration checks."""
import json
import os
from pathlib import Path
import socket
import subprocess
import time


class NativeApp:
    def __init__(self, binary, directory, config, path=None, cwd=None):
        self.directory = Path(directory)
        self.socket_path = self.directory / "app.sock"
        self.log = (self.directory / "app.log").open("wb")
        env = {**os.environ, "MYGO_AUTOMATION": str(self.socket_path), "MYGO_LOCALE": "en", "PLAINMARK_CONFIG_DIR": str(config)}
        env.pop("NOTES_ROOT", None)
        self.process = subprocess.Popen([str(binary)] + ([str(path)] if path else []), cwd=cwd, env=env, stdout=self.log, stderr=subprocess.STDOUT)
        self.client = None
        self.stream = None
        self.request_id = 0
        try:
            deadline = time.monotonic() + 30
            while not self.socket_path.exists():
                if self.process.poll() is not None or time.monotonic() > deadline:
                    raise RuntimeError((self.directory / "app.log").read_text())
                time.sleep(.02)
            self.client = socket.socket(socket.AF_UNIX)
            self.client.settimeout(15)
            self.client.connect(str(self.socket_path))
            self.stream = self.client.makefile("rwb")
            self.wait(lambda: bool(self.rpc("windows")["windows"]), "window did not open")
            self.rpc("idle", timeout=10000)
            self.wait(lambda: any(n.get("key") == "toggle-sidebar" and "disabled" not in n.get("states", []) for n in self.tree()), "startup did not finish")
        except BaseException:
            self.close()
            raise

    def rpc(self, method, **params):
        self.request_id += 1
        self.stream.write((json.dumps({"id": self.request_id, "method": method, "params": params}) + "\n").encode())
        self.stream.flush()
        line = self.stream.readline()
        if not line:
            raise EOFError("app closed its automation connection")
        response = json.loads(line)
        if "error" in response:
            raise RuntimeError(response)
        return response["result"]

    def tree(self):
        self.rpc("idle", timeout=10000)
        return self.rpc("tree")["nodes"]

    def wait(self, predicate, message):
        deadline = time.monotonic() + 20
        while not predicate():
            if self.process.poll() is not None or time.monotonic() > deadline:
                raise AssertionError(message + "\n" + (self.directory / "app.log").read_text())
            time.sleep(.02)

    def menu(self, *path):
        self.rpc("menu", path=list(path))

    def click(self, key, mods=None):
        node = next(n for n in self.tree() if n.get("key") == key)
        box = node["bounds"]
        self.rpc("input", events=[{"type": "click", "x": box["x"] + box["width"] / 2, "y": box["y"] + box["height"] / 2, "mods": mods or []}])
        self.rpc("idle", timeout=10000)

    def quit(self, label="Quit Plainmark Go"):
        try:
            self.menu("Plainmark Go", label)
        except EOFError:
            pass  # The quit may close the socket before its answer is flushed.
        result = self.process.wait(timeout=15)
        assert result == 0, (result, (self.directory / "app.log").read_text())

    def close(self):
        if self.process.poll() is None:
            self.process.terminate()
            try:
                self.process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait(timeout=10)
        if self.stream:
            self.stream.close()
        if self.client:
            self.client.close()
        self.log.close()
