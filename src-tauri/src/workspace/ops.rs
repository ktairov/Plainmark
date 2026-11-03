use std::{ffi::OsStr, fs, io::Write, path::Path};
use walkdir::WalkDir;

use crate::error::{AppError, AppResult};
use super::paths::{canon_root, child, directory, existing, is_markdown, relative, validate_name};
use super::{PathOperationResult, TrashInfo};

pub(super) fn create_note_in(
    root: &Path,
    parent: String,
    file_name: String,
    title: Option<String>,
) -> AppResult<PathOperationResult> {
    let mut name = validate_name(&file_name)?.to_string();
    if Path::new(&name).extension().is_none() {
        name.push_str(".md")
    }
    if !is_markdown(Path::new(&name)) {
        return Err(AppError::new("notMarkdown", "A note must use .md or .mdx."));
    }
    let (root, target) = child(root, &parent, &name)?;
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&target)
        .map_err(|error| {
            if error.kind() == std::io::ErrorKind::AlreadyExists {
                AppError::new("itemExists", "An item with this name already exists.")
            } else {
                AppError::from(error)
            }
        })?;
    let heading = title.unwrap_or_else(|| {
        Path::new(&name)
            .file_stem()
            .unwrap_or_default()
            .to_string_lossy()
            .replace(['-', '_'], " ")
    });
    if !heading.trim().is_empty() {
        writeln!(file, "# {}", heading.trim()).map_err(AppError::from)?
    }
    Ok(PathOperationResult {
        path: relative(&target, &root)?,
        kind: "note".into(),
    })
}

pub(super) fn create_folder_in(
    root: &Path,
    parent: String,
    name: String,
) -> AppResult<PathOperationResult> {
    let (root, target) = child(root, &parent, &name)?;
    fs::create_dir(&target).map_err(|error| {
        if error.kind() == std::io::ErrorKind::AlreadyExists {
            AppError::new("itemExists", "An item with this name already exists.")
        } else {
            AppError::from(error)
        }
    })?;
    Ok(PathOperationResult {
        path: relative(&target, &root)?,
        kind: "folder".into(),
    })
}

pub(super) fn rename_in(root: &Path, path: String, new_name: String) -> AppResult<PathOperationResult> {
    let root = canon_root(root)?;
    let source = existing(&root, &path)?;
    if source == root {
        return Err(AppError::new("rootNotRenamable", "The workspace root cannot be renamed."));
    }
    let mut name = validate_name(&new_name)?.to_string();
    let note = source.is_file() && is_markdown(&source);
    if note && Path::new(&name).extension().is_none() {
        name.push('.');
        name.push_str(source.extension().and_then(OsStr::to_str).unwrap_or("md"))
    }
    if note && !is_markdown(Path::new(&name)) {
        return Err(AppError::new("notMarkdown", "A note must keep .md or .mdx."));
    }
    let target = source.parent().ok_or_else(|| AppError::new("missingParent", "Missing parent"))?.join(name);
    if target.exists() {
        return Err(AppError::new("itemExists", "An item with this name already exists."));
    }
    fs::rename(source, &target)?;
    Ok(PathOperationResult {
        path: relative(&target, &root)?,
        kind: if target.is_dir() { "folder" } else { "note" }.into(),
    })
}

pub(super) fn move_in(root: &Path, path: String, destination: String) -> AppResult<PathOperationResult> {
    let root = canon_root(root)?;
    let source = existing(&root, &path)?;
    let destination = directory(&root, &destination)?;
    if source == root {
        return Err(AppError::new("rootNotMovable", "The workspace root cannot be moved."));
    }
    if source.is_dir() && destination.starts_with(&source) {
        return Err(AppError::new("moveIntoDescendant", "A folder cannot be moved into itself or its descendant."));
    }
    let target = destination.join(source.file_name().ok_or_else(|| AppError::new("missingName", "Missing name"))?);
    if target.exists() {
        return Err(AppError::new("itemExistsInDestination", "An item with this name already exists in the destination."));
    }
    fs::rename(source, &target)?;
    Ok(PathOperationResult {
        path: relative(&target, &root)?,
        kind: if target.is_dir() { "folder" } else { "note" }.into(),
    })
}

pub(super) fn trash_info_in(root: &Path, path: String) -> AppResult<TrashInfo> {
    let target = existing(root, &path)?;
    if target == canon_root(root)? {
        return Err(AppError::new("rootNotDeletable", "The workspace root cannot be deleted."));
    }
    let is_directory = target.is_dir();
    let nested_files = if is_directory {
        WalkDir::new(&target)
            .follow_links(false)
            .into_iter()
            .filter_map(Result::ok)
            .filter(|entry| entry.file_type().is_file())
            .count()
    } else {
        0
    };
    Ok(TrashInfo {
        path,
        name: target.file_name().unwrap_or_default().to_string_lossy().into(),
        nested_files,
        is_directory,
    })
}

pub(super) fn trash_in(root: &Path, path: String) -> AppResult<PathOperationResult> {
    let target = existing(root, &path)?;
    if target == canon_root(root)? {
        return Err(AppError::new("rootNotDeletable", "The workspace root cannot be deleted."));
    }
    let kind = if target.is_dir() { "folder" } else { "note" };
    trash::delete(target).map_err(|error| AppError::new("trashFailed", format!("Could not move item to Trash: {error}")))?;
    Ok(PathOperationResult {
        path,
        kind: kind.into(),
    })
}
