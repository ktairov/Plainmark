use std::path::Path;

fn main() {
    let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR is set");
    let notes_root = Path::new(&manifest_dir)
        .join("../../..")
        .canonicalize()
        .expect("Workspace root can be resolved from utils/generator/src-tauri");

    println!(
        "cargo:rustc-env=NOTES_DEFAULT_ROOT={}",
        notes_root.display()
    );
    tauri_build::build();
}
