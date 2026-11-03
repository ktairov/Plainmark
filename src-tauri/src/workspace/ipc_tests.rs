use super::*;
use std::fs;
use tempfile::tempdir;

fn fixture() -> tempfile::TempDir {
    let directory = tempdir().unwrap();
    fs::create_dir(directory.path().join("folder")).unwrap();
    fs::write(directory.path().join("folder/note.md"), "# Original\nBody").unwrap();
    directory
}

#[test]
fn save_note_accepts_timestamp_through_tauri_ipc() {
    let directory = fixture();
    let note = read_from(directory.path(), "folder/note.md".into()).unwrap();
    let app = tauri::test::mock_builder()
        .manage(RootState::new(directory.path().to_path_buf()))
        .invoke_handler(tauri::generate_handler![save_note])
        .build(tauri::test::mock_context(tauri::test::noop_assets()))
        .unwrap();
    let webview = tauri::WebviewWindowBuilder::new(&app, "main", Default::default())
        .build()
        .unwrap();
    let response = tauri::test::get_ipc_response(
        &webview,
        tauri::webview::InvokeRequest {
            cmd: "save_note".into(),
            callback: tauri::ipc::CallbackFn(0),
            error: tauri::ipc::CallbackFn(1),
            url: "tauri://localhost".parse().unwrap(),
            body: tauri::ipc::InvokeBody::Json(serde_json::json!({
                "path": note.path,
                "markdown": "# Saved through IPC",
                "expectedModifiedMs": note.modified_ms,
                "force": false
            })),
            headers: Default::default(),
            invoke_key: tauri::test::INVOKE_KEY.into(),
        },
    )
    .expect("save_note IPC call should succeed")
    .deserialize::<serde_json::Value>()
    .unwrap();
    assert_eq!(response["title"], "Saved through IPC");
}

