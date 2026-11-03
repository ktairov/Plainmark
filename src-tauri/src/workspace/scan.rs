use std::{ffi::OsStr, fs, path::Path};
use tree_sitter::{Node, Parser};
use walkdir::WalkDir;

use crate::error::{AppError, AppResult};
use super::paths::{canon_root, existing, is_markdown, modified_ms, relative, should_descend};
use super::{FolderEntry, Heading, NoteContent, NoteEntry, WorkspaceSnapshot};

pub(super) fn heading_line(line: &str) -> Option<Heading> {
    let trimmed = line.trim_start();
    let level = trimmed.chars().take_while(|character| *character == '#').count();
    if level == 0 || level > 6 {
        return None;
    }
    let text = trimmed[level..].trim().trim_end_matches('#').trim();
    if text.is_empty() {
        None
    } else {
        Some(Heading { level, text: text.into() })
    }
}

fn walk_headings(node: Node, source: &str, out: &mut Vec<Heading>) {
    if matches!(node.kind(), "atx_heading" | "setext_heading") {
        if let Some(heading) = source
            .lines()
            .nth(node.start_position().row)
            .and_then(heading_line)
        {
            out.push(heading)
        }
    }
    let mut cursor = node.walk();
    for child in node.children(&mut cursor) {
        walk_headings(child, source, out)
    }
}

pub(super) fn parse_headings(markdown: &str) -> Vec<Heading> {
    let mut parser = Parser::new();
    if parser.set_language(&tree_sitter_md::LANGUAGE.into()).is_ok() {
        if let Some(tree) = parser.parse(markdown, None) {
            let mut headings = vec![];
            walk_headings(tree.root_node(), markdown, &mut headings);
            if !headings.is_empty() {
                return headings;
            }
        }
    }
    markdown.lines().filter_map(heading_line).collect()
}

fn strip_markdown_links(value: &str) -> String {
    let chars: Vec<char> = value.chars().collect();
    let mut output = String::with_capacity(value.len());
    let mut index = 0;
    while index < chars.len() {
        let image = chars[index] == '!' && chars.get(index + 1) == Some(&'[');
        let bracket = if image { index + 1 } else { index };
        if chars[bracket] == '[' {
            if let Some(close_offset) = chars[bracket + 1..].iter().position(|character| *character == ']') {
                let close = bracket + 1 + close_offset;
                if chars.get(close + 1) == Some(&'(') {
                    if let Some(end_offset) = chars[close + 2..].iter().position(|character| *character == ')') {
                        output.extend(chars[bracket + 1..close].iter());
                        index = close + 3 + end_offset;
                        continue;
                    }
                }
            }
        }
        output.push(chars[index]);
        index += 1;
    }
    output
}

pub(super) fn display_title(value: &str) -> String {
    let links = strip_markdown_links(value);
    let without_markers = links
        .replace("**", "")
        .replace("__", "")
        .replace("~~", "")
        .replace(['*', '`'], "");
    let mut plain = String::with_capacity(without_markers.len());
    let mut in_tag = false;
    let mut escaped = false;
    for character in without_markers.chars() {
        if escaped {
            plain.push(character);
            escaped = false;
        } else if character == '\\' {
            escaped = true;
        } else if character == '<' {
            in_tag = true;
        } else if character == '>' && in_tag {
            in_tag = false;
        } else if !in_tag && !character.is_control() {
            plain.push(character);
        }
    }
    let entities = plain
        .replace("&nbsp;", " ")
        .replace("&amp;", "&")
        .replace("&lt;", "<")
        .replace("&gt;", ">")
        .replace("&quot;", "\"")
        .replace("&#39;", "'");
    let punctuation = entities
        .replace("---", "—")
        .replace("--", "–")
        .replace("...", "…");
    punctuation.split_whitespace().collect::<Vec<_>>().join(" ")
}

pub(super) fn title_for(path: &str, markdown: &str, headings: &[Heading]) -> String {
    let title = headings
        .first()
        .map(|heading| heading.text.clone())
        .or_else(|| markdown.lines().find_map(heading_line).map(|heading| heading.text))
        .unwrap_or_else(|| {
            Path::new(path)
                .file_stem()
                .and_then(OsStr::to_str)
                .unwrap_or(path)
                .replace(['-', '_'], " ")
        });
    display_title(&title)
}

pub(super) fn content(path: String, full: &Path) -> AppResult<NoteContent> {
    let markdown = fs::read_to_string(full).map_err(AppError::from)?;
    let headings = parse_headings(&markdown);
    Ok(NoteContent {
        title: title_for(&path, &markdown, &headings),
        path,
        markdown,
        headings,
        modified_ms: modified_ms(full),
    })
}

fn note_entry(path: &Path, root: &Path) -> AppResult<NoteEntry> {
    let note_path = relative(path, root)?;
    let note = content(note_path.clone(), path)?;
    Ok(NoteEntry {
        path: note_path.clone(),
        title: note.title,
        folder: Path::new(&note_path)
            .parent()
            .map(|parent| parent.to_string_lossy().replace('\\', "/"))
            .unwrap_or_default(),
        headings: note.headings,
        modified_ms: note.modified_ms,
        size: fs::metadata(path).map_err(AppError::from)?.len(),
    })
}

pub(super) fn scan_workspace_from(root: &Path) -> AppResult<WorkspaceSnapshot> {
    let root = canon_root(root)?;
    let mut notes = vec![];
    let mut folders = vec![];
    for entry_result in WalkDir::new(&root)
        .follow_links(false)
        .into_iter()
        .filter_entry(should_descend)
    {
        let entry = entry_result.map_err(|error| AppError::new("scanFailed", error.to_string()))?;
        if entry.file_type().is_file() && is_markdown(entry.path()) {
            notes.push(note_entry(entry.path(), &root)?)
        } else if entry.depth() > 0 && entry.file_type().is_dir() {
            folders.push(FolderEntry {
                path: relative(entry.path(), &root)?,
                name: entry.file_name().to_string_lossy().into(),
            })
        }
    }
    notes.sort_by(|left, right| left.path.cmp(&right.path));
    folders.sort_by(|left, right| left.path.cmp(&right.path));
    Ok(WorkspaceSnapshot { notes, folders })
}

pub(super) fn read_from(root: &Path, path: String) -> AppResult<NoteContent> {
    let file = existing(root, &path)?;
    if !is_markdown(&file) {
        return Err(AppError::new("notMarkdown", "Path is not a Markdown file."));
    }
    content(path, &file)
}
