use std::fs;

use tempfile::tempdir;

use super::ops::{create_folder_in, create_note_in, move_in, rename_in, trash_info_in};
use super::paths::{existing, validate_notes_root};
use super::save::save_to;
use super::scan::{display_title, parse_headings, read_from, scan_workspace_from, title_for};
use super::Heading;
use crate::crypto::{decrypt_text_value, encrypt_text_value};

fn fixture() -> tempfile::TempDir {
    let directory = tempdir().unwrap();
    fs::create_dir(directory.path().join("folder")).unwrap();
    fs::write(directory.path().join("folder/note.md"), "# Original\nBody").unwrap();
    directory
}

#[test]
fn headings() {
    assert_eq!(
        parse_headings("# One\n## Two"),
        vec![
            Heading {
                level: 1,
                text: "One".into()
            },
            Heading {
                level: 2,
                text: "Two".into()
            }
        ]
    );
}

#[test]
fn display_titles_are_typographic_and_plain() {
    assert_eq!(
        display_title("Engineering --- Specification"),
        "Engineering — Specification"
    );
    assert_eq!(display_title("Version 1 -- draft..."), "Version 1 – draft…");
    assert_eq!(
        display_title("**Bold** and [linked](https://example.com) `code`"),
        "Bold and linked code"
    );
    assert_eq!(
        display_title("Title&nbsp; <small>internal</small>   text"),
        "Title internal text"
    );
    assert_eq!(
        title_for("fallback.md", "# **Tree** --- Window", &[]),
        "Tree — Window"
    );
}

#[test]
fn encrypted_blocks_round_trip_and_authenticate() {
    let payload = encrypt_text_value("# Secret\nprivate text".into(), "correct horse".into())
        .expect("encryption should succeed");
    assert!(payload.starts_with("enc:v1:"));
    assert!(!payload.contains("Secret"));
    assert_eq!(
        decrypt_text_value(payload.clone(), "correct horse".into()).unwrap(),
        "# Secret\nprivate text"
    );
    assert!(decrypt_text_value(payload.clone(), "wrong password".into()).is_err());

    let mut parts: Vec<String> = payload.split(':').map(str::to_owned).collect();
    let replacement = if parts[4].starts_with('A') { "B" } else { "A" };
    parts[4].replace_range(0..1, replacement);
    assert!(decrypt_text_value(parts.join(":"), "correct horse".into()).is_err());
}

#[test]
fn path_security() {
    let directory = fixture();
    assert_eq!(
        validate_notes_root(directory.path()).unwrap(),
        directory.path().canonicalize().unwrap()
    );
    assert!(existing(directory.path(), "../x").is_err());
    assert!(create_folder_in(directory.path(), "".into(), ".x".into()).is_err());
    assert!(create_folder_in(directory.path(), "".into(), "a/b".into()).is_err());
    assert!(validate_notes_root(&directory.path().join("folder/note.md")).is_err());
    #[cfg(unix)]
    {
        std::os::unix::fs::symlink(directory.path().parent().unwrap(), directory.path().join("escape"))
            .unwrap();
        assert!(existing(directory.path(), "escape").is_err());
    }
}

#[test]
fn saving_and_conflict() {
    let directory = fixture();
    let note = read_from(directory.path(), "folder/note.md".into()).unwrap();
    let saved = save_to(
        directory.path(),
        note.path.clone(),
        "# Updated".into(),
        note.modified_ms,
        false,
    )
    .unwrap();
    assert_eq!(saved.title, "Updated");
    let conflict = save_to(directory.path(), note.path, "stale".into(), note.modified_ms, false).unwrap_err();
    assert_eq!(conflict.code, "conflict");
    assert_eq!(conflict.details.as_ref().and_then(|details| details.get("expected")), Some(&note.modified_ms.to_string()));
}

#[test]
fn saves_large_notes_and_rapid_revisions() {
    let directory = fixture();
    let first = read_from(directory.path(), "folder/note.md".into()).unwrap();
    let large = format!("# Large\n{}", "content line\n".repeat(150_000));
    let second = save_to(
        directory.path(),
        first.path,
        large.clone(),
        first.modified_ms,
        false,
    )
    .unwrap();
    let third = save_to(
        directory.path(),
        second.path,
        "# Immediate revision".into(),
        second.modified_ms,
        false,
    )
    .unwrap();
    assert_eq!(third.markdown, "# Immediate revision");
    assert!(large.len() > 1_000_000);
}

#[test]
fn crud_no_overwrite() {
    let directory = fixture();
    fs::create_dir(directory.path().join("destination")).unwrap();
    let created = create_note_in(
        directory.path(),
        "folder".into(),
        "new".into(),
        Some("Title".into()),
    )
    .unwrap();
    assert!(create_note_in(directory.path(), "folder".into(), "new.md".into(), None).is_err());
    let renamed = rename_in(directory.path(), created.path, "renamed".into()).unwrap();
    let moved = move_in(directory.path(), renamed.path, "destination".into()).unwrap();
    assert_eq!(moved.path, "destination/renamed.md");
    assert_eq!(
        fs::read_to_string(directory.path().join(moved.path)).unwrap(),
        "# Title\n"
    );
}

#[test]
fn folder_rules_and_listing() {
    let directory = fixture();
    fs::create_dir_all(directory.path().join("folder/child/empty")).unwrap();
    fs::write(directory.path().join("alpha.md"), "# Alpha").unwrap();
    assert!(move_in(directory.path(), "folder".into(), "folder/child".into()).is_err());
    let snapshot = scan_workspace_from(directory.path()).unwrap();
    assert_eq!(
        snapshot
            .notes
            .iter()
            .map(|note| note.path.as_str())
            .collect::<Vec<_>>(),
        vec!["alpha.md", "folder/note.md"]
    );
    assert!(snapshot
        .folders
        .iter()
        .any(|folder| folder.path == "folder/child/empty"));
}

#[test]
fn trash_preview() {
    let directory = fixture();
    fs::write(directory.path().join("folder/t.txt"), "x").unwrap();
    assert_eq!(
        trash_info_in(directory.path(), "folder".into())
            .unwrap()
            .nested_files,
        2
    );
}
