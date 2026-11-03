use std::{
    ffi::OsStr,
    fs,
    path::{Component, Path, PathBuf},
};
use walkdir::DirEntry;

use crate::error::{AppError, AppResult};

pub(super) fn is_hidden(entry: &DirEntry) -> bool {
    entry
        .file_name()
        .to_str()
        .map(|name| name.starts_with('.'))
        .unwrap_or(false)
}

pub(super) fn excluded(name: &OsStr) -> bool {
    let name = name.to_string_lossy();
    name.starts_with('.') || matches!(name.as_ref(), "node_modules" | "target" | "dist")
}

pub(super) fn should_descend(entry: &DirEntry) -> bool {
    entry.depth() == 0 || (!is_hidden(entry) && !excluded(entry.file_name()))
}

pub(super) fn is_markdown(path: &Path) -> bool {
    path.extension()
        .and_then(OsStr::to_str)
        .map(|extension| extension.eq_ignore_ascii_case("md") || extension.eq_ignore_ascii_case("mdx"))
        .unwrap_or(false)
}

pub(crate) fn relative(path: &Path, root: &Path) -> AppResult<String> {
    path.strip_prefix(root)
        .map(|path| path.to_string_lossy().replace('\\', "/"))
        .map_err(|error| AppError::new("pathOutsideWorkspace", error.to_string()))
}

pub(super) fn modified_ms(path: &Path) -> u64 {
    fs::metadata(path)
        .and_then(|metadata| metadata.modified())
        .ok()
        .and_then(|time| time.duration_since(std::time::UNIX_EPOCH).ok())
        .map(|duration| duration.as_millis().min(u64::MAX as u128) as u64)
        .unwrap_or(0)
}

pub(super) fn validate_notes_root(path: &Path) -> AppResult<PathBuf> {
    let root = path
        .canonicalize()
        .map_err(|error| AppError::new("cannotAccessFolder", format!("Cannot access selected folder: {error}")))?;
    if !root.is_dir() {
        return Err(AppError::new("notAFolder", "The selected path is not a folder."));
    }
    Ok(root)
}

pub(crate) fn validate_relative(path: &str, empty: bool) -> AppResult<PathBuf> {
    if path.is_empty() && empty {
        return Ok(PathBuf::new());
    }
    let parsed = Path::new(path);
    if path.trim().is_empty()
        || parsed.is_absolute()
        || parsed.components().any(|component| !matches!(component, Component::Normal(_)))
    {
        return Err(AppError::new("invalidRelativePath", "Path must be a normalized relative path inside the workspace."));
    }
    if parsed
        .components()
        .any(|component| matches!(component, Component::Normal(name) if excluded(name)))
    {
        return Err(AppError::new("excludedPath", "Hidden and excluded paths cannot be modified."));
    }
    Ok(parsed.into())
}

pub(super) fn validate_name(name: &str) -> AppResult<&str> {
    let normalized = name.trim();
    if normalized.is_empty() || matches!(normalized, "." | "..") {
        return Err(AppError::new("invalidName", "Name cannot be empty, '.' or '..'."));
    }
    if normalized.contains('/') || normalized.contains('\\') || Path::new(normalized).components().count() != 1 {
        return Err(AppError::new("nameHasSeparator", "Name cannot contain path separators."));
    }
    if excluded(OsStr::new(normalized)) {
        return Err(AppError::new("excludedName", "Hidden and excluded names are not allowed."));
    }
    Ok(normalized)
}

pub(super) fn canon_root(root: &Path) -> AppResult<PathBuf> {
    root.canonicalize()
        .map_err(|error| AppError::new("cannotAccessWorkspace", format!("Cannot access workspace root: {error}")))
}

pub(super) fn existing(root: &Path, path: &str) -> AppResult<PathBuf> {
    let relative_path = validate_relative(path, false)?;
    let root = canon_root(root)?;
    let target = root
        .join(relative_path)
        .canonicalize()
        .map_err(|error| AppError::new("pathUnavailable", error.to_string()))?;
    if !target.starts_with(&root) {
        return Err(AppError::new("pathOutsideWorkspace", "Path is outside the workspace folder."));
    }
    Ok(target)
}

pub(super) fn directory(root: &Path, path: &str) -> AppResult<PathBuf> {
    let relative_path = validate_relative(path, true)?;
    let root = canon_root(root)?;
    let target = root
        .join(relative_path)
        .canonicalize()
        .map_err(|error| AppError::new("destinationUnavailable", error.to_string()))?;
    if !target.starts_with(&root) || !target.is_dir() {
        return Err(AppError::new("destinationNotFolder", "Destination must be a folder inside the workspace."));
    }
    Ok(target)
}

pub(super) fn child(root: &Path, parent: &str, name: &str) -> AppResult<(PathBuf, PathBuf)> {
    let root = canon_root(root)?;
    let parent = directory(root.as_path(), parent)?;
    Ok((root, parent.join(validate_name(name)?)))
}
