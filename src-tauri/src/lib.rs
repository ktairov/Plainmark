use std::sync::atomic::Ordering;
use tauri::{Emitter, Manager, WindowEvent};
use tauri_plugin_window_state::{AppHandleExt, StateFlags};

mod crypto;
mod error;
mod menu;
mod state;
mod watcher;
mod workspace;

use crypto::{decrypt_text, encrypt_text};
use menu::{
    build_app_menu, sync_menu_state, sync_recent_folders, MenuLabels,
    set_color_mode_menu_label as update_color_mode_menu,
    set_encrypt_selection_enabled as update_encrypt_selection_menu,
};
use error::{AppError, AppResult};
use state::{normalize_language, LanguageState, LifecycleState, MenuLabelsState, RootState};
use watcher::{start_watcher, WatcherState};
use workspace::{
    absolute_note_path, create_folder, create_note, get_config, get_trash_info, initial_language,
    initial_menu_labels, initial_notes_root, move_path, persist_language, read_note, rename_path,
    reveal_note, save_note, scan_workspace, set_notes_root, trash_path,
};

#[cfg(target_os = "macos")]
use tauri::RunEvent as MacRunEvent;
#[cfg(target_os = "macos")]
use tauri_plugin_decorum::WebviewWindowExt;

#[tauri::command]
fn hide_main_window(app: tauri::AppHandle) -> AppResult<()> {
    app.save_window_state(StateFlags::all())
        .map_err(|error| AppError::new("windowStateFailed", error.to_string()))?;
    if let Some(window) = app.get_webview_window("main") {
        window
            .hide()
            .map_err(|error| AppError::new("windowFailed", error.to_string()))?
    }
    Ok(())
}

#[tauri::command]
fn reveal_main_window(app: tauri::AppHandle, lifecycle: tauri::State<LifecycleState>) -> AppResult<()> {
    lifecycle.window_revealed.store(true, Ordering::SeqCst);
    show(&app);
    Ok(())
}

#[tauri::command]
fn exit_app(app: tauri::AppHandle, lifecycle: tauri::State<LifecycleState>) {
    lifecycle.allow_exit.store(true, Ordering::SeqCst);
    app.exit(0);
}

#[tauri::command]
fn set_encrypt_selection_enabled(app: tauri::AppHandle, enabled: bool) -> AppResult<()> {
    update_encrypt_selection_menu(&app, enabled)
}

#[tauri::command]
fn set_color_mode_menu_label(
    app: tauri::AppHandle,
    labels: tauri::State<MenuLabelsState>,
    night_mode: bool,
) -> AppResult<()> {
    update_color_mode_menu(&app, night_mode, &labels.current())
}

#[tauri::command]
fn preview_app_language(
    app: tauri::AppHandle,
    language_state: tauri::State<LanguageState>,
    menu_labels: tauri::State<MenuLabelsState>,
    language: String,
    labels: std::collections::HashMap<String, String>,
) -> AppResult<()> {
    let language = normalize_language(&language);
    let labels = MenuLabels::new(labels);
    if language_state.get() == language && menu_labels.current() == labels {
        return Ok(());
    }
    let menu = build_app_menu(&app, &labels)
        .map_err(|error| AppError::new("menuBuildFailed", error.to_string()))?;
    app.set_menu(menu)
        .map_err(|error| AppError::new("menuBuildFailed", error.to_string()))?;
    menu_labels.set(labels)?;
    language_state.set(language)?;
    Ok(())
}

#[tauri::command]
fn set_app_language(
    app: tauri::AppHandle,
    root: tauri::State<RootState>,
    language_state: tauri::State<LanguageState>,
    menu_labels: tauri::State<MenuLabelsState>,
    language: String,
    labels: std::collections::HashMap<String, String>,
) -> AppResult<()> {
    let language = normalize_language(&language);
    persist_language(&app, &root.get()?, &language, labels.clone())?;
    preview_app_language(app, language_state, menu_labels, language, labels)
}

fn show(app: &tauri::AppHandle) {
    #[cfg(target_os = "macos")]
    let _ = app.show();
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.unminimize();
        let _ = window.show();
        let _ = window.set_focus();
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_decorum::init())
        .manage(WatcherState::default())
        .manage(LifecycleState::default())
        .plugin(tauri_plugin_clipboard_manager::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_window_state::Builder::default().build())
        .invoke_handler(tauri::generate_handler![
            get_config,
            set_notes_root,
            encrypt_text,
            decrypt_text,
            scan_workspace,
            read_note,
            save_note,
            create_note,
            create_folder,
            rename_path,
            move_path,
            get_trash_info,
            trash_path,
            absolute_note_path,
            reveal_note,
            sync_menu_state,
            sync_recent_folders,
            set_encrypt_selection_enabled,
            set_color_mode_menu_label,
            preview_app_language,
            set_app_language,
            hide_main_window,
            reveal_main_window,
            exit_app
        ])
        .on_window_event(|window, event| {
            #[cfg(target_os = "macos")]
            if let WindowEvent::Focused(true) = event {
                if let Some(webview_window) = window
                    .app_handle()
                    .get_webview_window(window.label())
                {
                    let _ = webview_window.set_traffic_lights_inset(9.0, 22.0);
                }
            }
            if let WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = window.emit("window-close-requested", ());
            }
        })
        .on_menu_event(|app, event| {
            let id = event.id().as_ref();
            if matches!(id, "plainmark.view-preview" | "plainmark.view-editor" | "plainmark.view-split") {
                let _ = menu::select_view_mode(app, id);
            }
            if id.starts_with("plainmark.") {
                let _ = app.emit("app-menu", id.to_string());
            }
        })
        .setup(|app| {
            let initial_language = initial_language(app.handle());
            let labels = MenuLabels::new(initial_menu_labels(app.handle()));
            app.set_menu(build_app_menu(app, &labels)?)?;
            app.manage(LanguageState::new(initial_language));
            app.manage(MenuLabelsState::new(labels));
            let root = initial_notes_root(app.handle());
            app.manage(RootState::new(root.clone()));
            // The window is created hidden (see tauri.conf.json). The frontend
            // reveals it after the first themed frame; this timer only covers a
            // frontend that fails to boot.
            let handle = app.handle().clone();
            std::thread::spawn(move || {
                std::thread::sleep(std::time::Duration::from_secs(2));
                if !handle
                    .state::<LifecycleState>()
                    .window_revealed
                    .load(Ordering::SeqCst)
                {
                    show(&handle);
                }
            });
            let watcher = start_watcher(app.handle().clone(), root)
                .map_err(|error| AppError::new("watcherStartFailed", error.to_string()))?;
            *app.state::<WatcherState>()
                .watcher
                .lock()
                .map_err(|_| AppError::new("watcherLock", "Watcher lock is unavailable."))? = Some(watcher);
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("build failed")
        .run(|app, event| {
            #[cfg(target_os = "macos")]
            if let tauri::RunEvent::Ready = &event {
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.set_traffic_lights_inset(9.0, 22.0);
                }
            }
            if let tauri::RunEvent::ExitRequested { api, .. } = &event {
                let lifecycle = app.state::<LifecycleState>();
                if !lifecycle.allow_exit.load(Ordering::SeqCst) {
                    api.prevent_exit();
                    let _ = app.emit("app-exit-requested", ());
                }
            }
            #[cfg(target_os = "macos")]
            if let MacRunEvent::Reopen { .. } = event {
                show(app)
            }
            let _ = (&app, &event);
        });
}
