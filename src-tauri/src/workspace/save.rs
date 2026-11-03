use std::{fs, io::Write, path::Path};

use crate::error::{AppError, AppResult};
use super::paths::{existing, is_markdown, modified_ms};
use super::NoteContent;

pub(super) fn save_to(
    root: &Path,
    path: String,
    markdown: String,
    expected: u64,
    force: bool,
) -> AppResult<NoteContent> {
    let target = existing(root, &path)?;
    if !is_markdown(&target) {
        return Err(AppError::new("notMarkdown", "Only .md and .mdx files can be saved."));
    }
    let actual = modified_ms(&target);
    if !force && actual != expected {
        return Err(AppError::new("conflict", format!(
            "The file changed on disk (expected {expected}, found {actual})."
        ))
        .detail("expected", expected)
        .detail("found", actual));
    }
    let parent = target
        .parent()
        .ok_or_else(|| AppError::new("invalidPath", "File has no parent."))?;
    let mut temporary = tempfile::Builder::new()
        .prefix(".notes-save-")
        .tempfile_in(parent)?;
    if let Ok(metadata) = fs::metadata(&target) {
        temporary.as_file().set_permissions(metadata.permissions())?
    }
    temporary.write_all(markdown.as_bytes())?;
    temporary.as_file().sync_all()?;
    temporary
        .persist(&target)
        .map_err(|error| AppError::new("ioError", error.error.to_string()))?;
    if modified_ms(&target) <= expected {
        let next = expected.saturating_add(1);
        let timestamp = filetime::FileTime::from_unix_time(
            (next / 1_000).min(i64::MAX as u64) as i64,
            ((next % 1_000) * 1_000_000) as u32,
        );
        filetime::set_file_mtime(&target, timestamp)?;
    }
    super::scan::content(path, &target)
}
