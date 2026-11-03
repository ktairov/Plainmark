use crate::workspace::{relative, validate_relative};
use notify::{Config, Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher};
use serde::Serialize;
use std::{
    collections::BTreeSet,
    path::{Path, PathBuf},
    sync::{mpsc, Mutex},
    thread,
    time::Duration,
};
use tauri::Emitter;

#[derive(Default)]
pub(crate) struct WatcherState {
    pub(crate) watcher: Mutex<Option<RecommendedWatcher>>,
}

#[derive(Debug, Serialize, Clone)]
struct WorkspaceChanged {
    paths: Vec<String>,
}

fn changed(event: &Event, root: &Path) -> Vec<String> {
    if matches!(
        event.kind,
        EventKind::Create(_) | EventKind::Modify(_) | EventKind::Remove(_)
    ) {
        event
            .paths
            .iter()
            .filter_map(|path| relative(path, root).ok())
            .filter(|path| validate_relative(path, false).is_ok())
            .collect()
    } else {
        vec![]
    }
}

pub(crate) fn start_watcher(
    app: tauri::AppHandle,
    root: PathBuf,
) -> Result<RecommendedWatcher, notify::Error> {
    let root = root.canonicalize().unwrap_or(root);
    let (tx, rx) = mpsc::channel();
    let mut watcher = RecommendedWatcher::new(tx, Config::default())?;
    watcher.watch(&root, RecursiveMode::Recursive)?;
    thread::spawn(move || {
        let mut pending = BTreeSet::new();
        loop {
            match rx.recv_timeout(Duration::from_millis(300)) {
                Ok(Ok(event)) => pending.extend(changed(&event, &root)),
                Ok(Err(error)) => {
                    let _ = app.emit("plainmark-watch-error", error.to_string());
                }
                Err(mpsc::RecvTimeoutError::Timeout) if !pending.is_empty() => {
                    let paths = pending.iter().cloned().collect();
                    pending.clear();
                    let _ = app.emit("plainmark-files-changed", WorkspaceChanged { paths });
                }
                Err(mpsc::RecvTimeoutError::Disconnected) => break,
                _ => {}
            }
        }
    });
    Ok(watcher)
}
