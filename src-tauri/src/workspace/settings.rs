use std::{
    collections::HashMap,
    fs,
    io::Write,
    path::{Path, PathBuf},
};
use tauri::{Manager, Runtime};

use crate::error::{AppError, AppResult};
use super::paths::validate_notes_root;

#[derive(Debug, serde::Serialize, serde::Deserialize)]
pub(crate) struct AppSettings {
    notes_root: String,
    #[serde(default)]
    language: Option<String>,
    #[serde(default)]
    menu_labels: HashMap<String, String>,
}

fn default_notes_root() -> PathBuf {
    std::env::var("NOTES_ROOT")
        .map(PathBuf::from)
        .unwrap_or_else(|_| PathBuf::from(env!("NOTES_DEFAULT_ROOT")))
}

fn settings_path<R: Runtime>(app: &tauri::AppHandle<R>) -> AppResult<PathBuf> {
    let directory = app
        .path()
        .app_config_dir()
        .map_err(|error| AppError::new("settingsDirectoryUnavailable", error.to_string()))?;
    fs::create_dir_all(&directory).map_err(AppError::from)?;
    Ok(directory.join("settings.json"))
}

fn read_settings<R: Runtime>(app: &tauri::AppHandle<R>) -> Option<AppSettings> {
    let path = settings_path(app).ok()?;
    let json = fs::read_to_string(path).ok()?;
    serde_json::from_str::<AppSettings>(&json).ok()
}

pub(crate) fn initial_notes_root<R: Runtime>(app: &tauri::AppHandle<R>) -> PathBuf {
    let stored = read_settings(app)
        .map(|settings| PathBuf::from(settings.notes_root))
        .and_then(|path| validate_notes_root(&path).ok());
    stored
        .or_else(|| validate_notes_root(&default_notes_root()).ok())
        .unwrap_or_else(default_notes_root)
}

pub(crate) fn initial_language<R: Runtime>(app: &tauri::AppHandle<R>) -> String {
    read_settings(app)
        .and_then(|settings| settings.language)
        .map(|language| crate::state::normalize_language(&language))
        .unwrap_or_else(|| "en".to_string())
}

pub(crate) fn initial_menu_labels<R: Runtime>(
    app: &tauri::AppHandle<R>,
) -> HashMap<String, String> {
    read_settings(app)
        .map(|settings| settings.menu_labels)
        .unwrap_or_default()
}

fn write_settings<R: Runtime>(app: &tauri::AppHandle<R>, settings: &AppSettings) -> AppResult<()> {
    let path = settings_path(app)?;
    let parent = path
        .parent()
        .ok_or_else(|| AppError::new("settingsDirectoryUnavailable", "Settings directory is unavailable."))?;
    let mut temporary = tempfile::NamedTempFile::new_in(parent).map_err(AppError::from)?;
    serde_json::to_writer_pretty(&mut temporary, settings).map_err(|error| AppError::new("settingsWriteFailed", error.to_string()))?;
    temporary.write_all(b"\n")?;
    temporary.flush()?;
    temporary.persist(&path).map_err(|error| AppError::new("settingsWriteFailed", error.error.to_string()))?;
    Ok(())
}

pub(super) fn persist_notes_root<R: Runtime>(app: &tauri::AppHandle<R>, root: &Path) -> AppResult<()> {
    let previous = read_settings(app);
    let language = previous.as_ref().and_then(|settings| settings.language.clone());
    let menu_labels = previous.map(|settings| settings.menu_labels).unwrap_or_default();
    write_settings(
        app,
        &AppSettings {
            notes_root: root.to_string_lossy().into_owned(),
            language,
            menu_labels,
        },
    )
}

pub(crate) fn persist_language<R: Runtime>(
    app: &tauri::AppHandle<R>,
    root: &Path,
    language: &str,
    menu_labels: HashMap<String, String>,
) -> AppResult<()> {
    write_settings(
        app,
        &AppSettings {
            notes_root: root.to_string_lossy().into_owned(),
            language: Some(language.to_string()),
            menu_labels,
        },
    )
}
