#!/usr/bin/env python3
"""Cross-check the unchanged Rust crypto implementation against Go, both ways.

The original implementation is read from Git commit cb2275f. The temporary
Rust harness removes Tauri command attributes and provides AppError.
"""
import json
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
LEGACY_COMMIT = "cb2275f"


def legacy_file(path):
    return subprocess.check_output(["git", "show", f"{LEGACY_COMMIT}:{path}"], cwd=ROOT.parent)


def main():
    with tempfile.TemporaryDirectory(prefix="plainmark-crypto-") as temp:
        work = Path(temp)
        (work / "src").mkdir()
        (work / "Cargo.toml").write_text('''[package]
name = "plainmark-crypto-compat"
version = "0.1.0"
edition = "2021"
[dependencies]
argon2 = "0.5"
base64 = "0.22"
chacha20poly1305 = "0.10"
zeroize = "1.8"
''')
        (work / "Cargo.lock").write_bytes(legacy_file("src-tauri/Cargo.lock"))
        source = legacy_file("src-tauri/src/crypto.rs").decode()
        source = source.replace("#[tauri::command]\n", "")
        harness = '''#![allow(dead_code)]
mod error {
    #[derive(Debug)]
    pub struct AppError { pub code: String, pub message: String }
    pub type AppResult<T> = Result<T, AppError>;
    impl AppError {
        pub fn new(code: &str, message: &str) -> Self {
            Self { code: code.into(), message: message.into() }
        }
    }
}
'''
        harness += source
        harness += '''
fn main() {
    let args: Vec<String> = std::env::args().collect();
    let result = match args[1].as_str() {
        "encrypt" => encrypt_text_value(args[2].clone(), args[3].clone()),
        "decrypt" => decrypt_text_value(args[2].clone(), args[3].clone()),
        _ => panic!("unknown mode"),
    };
    print!("{}", result.unwrap());
}
'''
        (work / "src/main.rs").write_text(harness)
        env = {**os.environ, "CARGO_TARGET_DIR": str(work / "target")}
        subprocess.run(["cargo", "test", "--quiet"], cwd=work, env=env, check=True)
        subprocess.run(["cargo", "build", "--quiet"], cwd=work, env=env, check=True)
        subprocess.run(["go", "build", "-o", str(work / "go-compat"), "./tools/cryptocompat"], cwd=ROOT, check=True)

        def rust(mode, text, password):
            return subprocess.check_output([str(work / "target/debug/plainmark-crypto-compat"), mode, text, password], text=True)

        def go(mode, text, password):
            return json.loads(subprocess.check_output([str(work / "go-compat")], input=json.dumps({"Mode": mode, "Text": text, "Password": password}), text=True))

        original, password = "# Секрет\nUnicode ☀ and public test data", "тестовый пароль"
        assert rust("decrypt", go("encrypt", original, password), password) == original
        assert go("decrypt", rust("encrypt", original, password), password) == original
        print("PASS: original Rust v1 tests; Go → Rust and Rust → Go random Unicode round trips")


if __name__ == "__main__":
    main()
