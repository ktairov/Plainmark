use std::path::Path;

mod ops;
mod paths;
mod save;
mod scan;
mod settings;
#[cfg(test)]
mod tests;

use crate::error::{AppError, AppResult};
use crate::state::RootState;
use crate::watcher::{start_watcher, WatcherState};
use ops::{create_folder_in, create_note_in, move_in, rename_in, trash_in, trash_info_in};
pub(crate) use paths::{relative, validate_relative};
use save::save_to;
use scan::{read_from, scan_workspace_from};

pub(crate) use settings::{initial_language, initial_menu_labels, initial_notes_root, persist_language};
use settings::persist_notes_root;

#[derive(Debug, serde::Serialize)]
pub(crate) struct WorkspaceConfig {
    root: String,
}

#[derive(Debug, serde::Serialize, Clone, PartialEq, Eq)]
pub(crate) struct Heading {
    level: usize,
    text: String,
}

#[derive(Debug, serde::Serialize, Clone)]
pub(crate) struct NoteEntry {
    path: String,
    title: String,
    folder: String,
    headings: Vec<Heading>,
    modified_ms: u64,
    size: u64,
}

#[derive(Debug, serde::Serialize, Clone)]
pub(crate) struct FolderEntry {
    path: String,
    name: String,
}

#[derive(Debug, serde::Serialize, Clone)]
pub(crate) struct WorkspaceSnapshot {
    notes: Vec<NoteEntry>,
    folders: Vec<FolderEntry>,
}

#[derive(Debug, serde::Serialize, Clone)]
pub(crate) struct NoteContent {
    path: String,
    title: String,
    markdown: String,
    headings: Vec<Heading>,
    modified_ms: u64,
}

#[derive(Debug, serde::Serialize, Clone)]
pub(crate) struct PathOperationResult {
    path: String,
    kind: String,
}

#[derive(Debug, serde::Serialize, Clone)]
pub(crate) struct TrashInfo {
    path: String,
    name: String,
    nested_files: usize,
    is_directory: bool,
}

#[tauri::command]
pub(crate) fn get_config(root: tauri::State<RootState>) -> AppResult<WorkspaceConfig> {
    Ok(WorkspaceConfig {
        root: root.get()?.to_string_lossy().into(),
    })
}

#[tauri::command]
pub(crate) fn scan_workspace(root: tauri::State<RootState>) -> AppResult<WorkspaceSnapshot> {
    scan_workspace_from(&root.get()?)
}

#[tauri::command]
pub(crate) fn read_note(root: tauri::State<RootState>, path: String) -> AppResult<NoteContent> {
    read_from(&root.get()?, path)
}

#[tauri::command]
pub(crate) fn absolute_note_path(root: tauri::State<RootState>, path: String) -> AppResult<String> {
    Ok(paths::existing(&root.get()?, &path)?.to_string_lossy().into_owned())
}

#[tauri::command]
pub(crate) fn reveal_note(app: tauri::AppHandle, root: tauri::State<RootState>, path: String) -> AppResult<()> {
    use tauri_plugin_opener::OpenerExt;
    app.opener()
        .reveal_item_in_dir(paths::existing(&root.get()?, &path)?)
        .map_err(|error| AppError::new("revealFailed", error.to_string()))
}

#[tauri::command]
pub(crate) fn save_note(
    root: tauri::State<RootState>,
    path: String,
    markdown: String,
    expected_modified_ms: u64,
    force: Option<bool>,
) -> AppResult<NoteContent> {
    let root = root.get()?;
    save_to(&root, path, markdown, expected_modified_ms, force.unwrap_or(false))
}

#[tauri::command]
pub(crate) fn create_note(
    root: tauri::State<RootState>,
    parent: String,
    file_name: String,
    title: Option<String>,
) -> AppResult<PathOperationResult> {
    create_note_in(&root.get()?, parent, file_name, title)
}

#[tauri::command]
pub(crate) fn create_folder(
    root: tauri::State<RootState>,
    parent: String,
    folder_name: String,
) -> AppResult<PathOperationResult> {
    create_folder_in(&root.get()?, parent, folder_name)
}

#[tauri::command]
pub(crate) fn rename_path(
    root: tauri::State<RootState>,
    path: String,
    new_name: String,
) -> AppResult<PathOperationResult> {
    rename_in(&root.get()?, path, new_name)
}

#[tauri::command]
pub(crate) fn move_path(
    root: tauri::State<RootState>,
    path: String,
    destination_folder: String,
) -> AppResult<PathOperationResult> {
    move_in(&root.get()?, path, destination_folder)
}

#[tauri::command]
pub(crate) fn get_trash_info(root: tauri::State<RootState>, path: String) -> AppResult<TrashInfo> {
    trash_info_in(&root.get()?, path)
}

#[tauri::command]
pub(crate) fn trash_path(root: tauri::State<RootState>, path: String) -> AppResult<PathOperationResult> {
    trash_in(&root.get()?, path)
}

#[tauri::command]
pub(crate) fn set_notes_root(
    app: tauri::AppHandle,
    root: tauri::State<RootState>,
    watcher: tauri::State<WatcherState>,
    path: String,
) -> AppResult<WorkspaceConfig> {
    let selected = paths::validate_notes_root(Path::new(&path))?;
    let next_watcher = start_watcher(app.clone(), selected.clone())
        .map_err(|error| AppError::new("watcherStartFailed", error.to_string()))?;
    persist_notes_root(&app, &selected)?;
    root.set(selected.clone())?;
    *watcher
        .watcher
        .lock()
        .map_err(|_| AppError::new("watcherLock", "Watcher lock is unavailable."))? = Some(next_watcher);
    Ok(WorkspaceConfig {
        root: selected.to_string_lossy().into_owned(),
    })
}

#[cfg(test)]
mod ipc_tests;
