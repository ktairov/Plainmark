use tauri::{
    menu::{CheckMenuItemBuilder, Menu, MenuBuilder, MenuItemBuilder, SubmenuBuilder},
    AppHandle, Runtime,
};

use std::collections::HashMap;

use crate::error::{AppError, AppResult};
use crate::state::MenuLabelsState;

/// Menu labels are translated on the frontend and pushed to Rust, so the
/// native menu has a single source of truth for every language.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct MenuLabels(HashMap<String, String>);

impl MenuLabels {
    pub(crate) fn new(labels: HashMap<String, String>) -> Self {
        Self(labels)
    }

    fn get<'a>(&'a self, key: &str, fallback: &'a str) -> &'a str {
        self.0.get(key).map(String::as_str).unwrap_or(fallback)
    }
}

const NOTE_MENU_ID: &str = "plainmark.note-menu";
const ENCRYPT_SELECTION_ID: &str = "plainmark.encrypt-selection";
const VIEW_MENU_ID: &str = "plainmark.view-menu";
const COLOR_MODE_ID: &str = "plainmark.toggle-color-mode";

pub(crate) fn build_app_menu<R: Runtime, M: tauri::Manager<R>>(
    app: &M,
    labels: &MenuLabels,
) -> tauri::Result<Menu<R>> {
    let open_file = MenuItemBuilder::with_id("plainmark.open-file", labels.get("openFile", "Open File…"))
        .accelerator("CmdOrCtrl+O")
        .build(app)?;
    let open_folder = MenuItemBuilder::with_id("plainmark.open-folder", labels.get("openFolder", "Open Folder…"))
        .accelerator("CmdOrCtrl+Shift+O")
        .build(app)?;
    let new_note = MenuItemBuilder::with_id("plainmark.new-note", labels.get("newNote", "New Note"))
        .accelerator("CmdOrCtrl+N")
        .build(app)?;
    let new_folder = MenuItemBuilder::with_id("plainmark.new-folder", labels.get("newFolder", "New Folder"))
        .accelerator("CmdOrCtrl+Shift+N")
        .build(app)?;
    let save = MenuItemBuilder::with_id("plainmark.save", labels.get("save", "Save"))
        .accelerator("CmdOrCtrl+S")
        .build(app)?;
    let close_tab = MenuItemBuilder::with_id("plainmark.close-tab", labels.get("closeTab", "Close Tab"))
        .accelerator("CmdOrCtrl+W")
        .build(app)?;
    let refresh = MenuItemBuilder::with_id("plainmark.refresh", labels.get("refresh", "Refresh"))
        .accelerator("CmdOrCtrl+R")
        .build(app)?;
    let preview = CheckMenuItemBuilder::with_id("plainmark.view-preview", labels.get("preview", "Preview"))
        .accelerator("Alt+1")
        .build(app)?;
    let editor = CheckMenuItemBuilder::with_id("plainmark.view-editor", labels.get("editor", "Editor"))
        .checked(false)
        .accelerator("Alt+2")
        .build(app)?;
    let split = CheckMenuItemBuilder::with_id("plainmark.view-split", labels.get("split", "Split View"))
        .checked(false)
        .accelerator("Alt+3")
        .build(app)?;
    let toggle_sidebar = MenuItemBuilder::with_id("plainmark.toggle-sidebar", labels.get("hideSidebar", "Hide Sidebar"))
        .accelerator("CmdOrCtrl+B")
        .build(app)?;
    let color_mode = MenuItemBuilder::with_id(COLOR_MODE_ID, labels.get("nightMode", "Night Mode")).build(app)?;
    let previous_tab = MenuItemBuilder::with_id("plainmark.previous-tab", labels.get("previousTab", "Previous Tab"))
        .accelerator("Ctrl+Shift+Tab")
        .build(app)?;
    let next_tab = MenuItemBuilder::with_id("plainmark.next-tab", labels.get("nextTab", "Next Tab"))
        .accelerator("Ctrl+Tab")
        .build(app)?;
    let copy_path = MenuItemBuilder::with_id("plainmark.copy-path", labels.get("copyPath", "Copy Path")).build(app)?;
    let copy_relative_path =
        MenuItemBuilder::with_id("plainmark.copy-relative-path", labels.get("copyRelativePath", "Copy Relative Path"))
            .build(app)?;
    let reveal = MenuItemBuilder::with_id(
        "plainmark.reveal-note",
        if cfg!(target_os = "macos") {
            labels.get("reveal", "Reveal in Finder")
        } else {
            labels.get("revealOther", "Show in File Manager")
        },
    )
    .build(app)?;
    let pin = MenuItemBuilder::with_id("plainmark.toggle-pin", labels.get("pin", "Pin")).build(app)?;
    let rename = MenuItemBuilder::with_id("plainmark.rename", labels.get("rename", "Rename…")).build(app)?;
    let move_item = MenuItemBuilder::with_id("plainmark.move", labels.get("move", "Move…")).build(app)?;
    let delete = MenuItemBuilder::with_id("plainmark.delete", labels.get("delete", "Move to Trash…")).build(app)?;
    let find = MenuItemBuilder::with_id("plainmark.find", labels.get("find", "Find in Note…"))
        .accelerator("CmdOrCtrl+F")
        .build(app)?;
    let search_notes = MenuItemBuilder::with_id("plainmark.search-notes", labels.get("searchNotes", "Search Notes…"))
        .accelerator("CmdOrCtrl+Shift+F")
        .build(app)?;
    let close_other_tabs =
        MenuItemBuilder::with_id("plainmark.close-other-tabs", labels.get("closeOtherTabs", "Close Other Tabs")).build(app)?;
    let close_all_tabs =
        MenuItemBuilder::with_id("plainmark.close-all-tabs", labels.get("closeAllTabs", "Close All Tabs")).build(app)?;
    let recent_folders = SubmenuBuilder::with_id(app, "plainmark.recent-folders", labels.get("recentFolders", "Recent Folders"))
        .item(
            &MenuItemBuilder::new(labels.get("noRecentFolders", "No Recent Folders"))
                .enabled(false)
                .build(app)?,
        )
        .build()?;
    let encrypt_selection = MenuItemBuilder::with_id(ENCRYPT_SELECTION_ID, labels.get("encryptSelection", "Encrypt Selection…"))
        .enabled(false)
        .build(app)?;
    let settings = MenuItemBuilder::with_id("plainmark.settings", labels.get("settings", "Settings…"))
        .accelerator("CmdOrCtrl+,")
        .build(app)?;
    let quit = MenuItemBuilder::with_id("plainmark.quit", labels.get("quit", "Quit Plainmark"))
        .accelerator("CmdOrCtrl+Q")
        .build(app)?;

    let app_menu = SubmenuBuilder::new(app, "Plainmark")
        .about(None)
        .separator()
        .item(&settings)
        .separator()
        .services()
        .separator()
        .hide()
        .hide_others()
        .show_all()
        .separator()
        .item(&quit)
        .build()?;
    let file_menu = SubmenuBuilder::new(app, labels.get("file", "File"))
        .items(&[&new_note, &new_folder])
        .separator()
        .items(&[&open_file, &open_folder, &recent_folders])
        .separator()
        .item(&save)
        .separator()
        .item(&close_tab)
        .build()?;
    let edit_menu = SubmenuBuilder::new(app, labels.get("edit", "Edit"))
        .undo()
        .redo()
        .separator()
        .cut()
        .copy()
        .paste()
        .select_all()
        .separator()
        .items(&[&find, &search_notes])
        .build()?;
    let view_menu = SubmenuBuilder::with_id(app, VIEW_MENU_ID, labels.get("view", "View"))
        .items(&[&preview, &editor, &split])
        .separator()
        .item(&toggle_sidebar)
        .item(&refresh)
        .item(&color_mode)
        .separator()
        .fullscreen()
        .build()?;
    let note_menu = SubmenuBuilder::with_id(app, NOTE_MENU_ID, labels.get("note", "Note"))
        .item(&pin)
        .item(&encrypt_selection)
        .separator()
        .items(&[&copy_path, &copy_relative_path, &reveal])
        .separator()
        .items(&[&rename, &move_item])
        .separator()
        .item(&delete)
        .build()?;
    let window_menu = SubmenuBuilder::new(app, labels.get("window", "Window"))
        .items(&[&previous_tab, &next_tab])
        .separator()
        .items(&[&close_other_tabs, &close_all_tabs])
        .separator()
        .minimize()
        .maximize()
        .build()?;

    MenuBuilder::new(app)
        .items(&[
            &app_menu,
            &file_menu,
            &edit_menu,
            &view_menu,
            &note_menu,
            &window_menu,
        ])
        .build()
}

#[derive(serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct MenuState {
    has_note: bool,
    dirty: bool,
    busy: bool,
    tab_count: usize,
    mode: String,
    sidebar_visible: bool,
    pinned: bool,
}

pub(crate) fn select_view_mode<R: Runtime>(
    app: &AppHandle<R>,
    selected: &str,
) -> tauri::Result<()> {
    if let Some(view) = app.menu().and_then(|menu| menu.get(VIEW_MENU_ID)) {
        if let Some(view) = view.as_submenu() {
            for id in [
                "plainmark.view-preview",
                "plainmark.view-editor",
                "plainmark.view-split",
            ] {
                if let Some(item) = view.get(id) {
                    if let Some(item) = item.as_check_menuitem() {
                        item.set_checked(id == selected)?;
                    }
                }
            }
        }
    }
    Ok(())
}

#[tauri::command]
pub(crate) fn sync_menu_state(
    app: AppHandle,
    labels: tauri::State<MenuLabelsState>,
    state: MenuState,
) -> AppResult<()> {
    let labels = labels.current();
    let menu = app.menu().ok_or_else(|| AppError::new("menuUnavailable", "Application menu is unavailable."))?;
    for submenu in menu.items().map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))? {
        let Some(submenu) = submenu.as_submenu() else {
            continue;
        };
        for item in submenu.items().map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))? {
            let id = item.id().as_ref();
            if id == "plainmark.recent-folders" {
                if let Some(recent) = item.as_submenu() {
                    recent.set_enabled(!state.busy).map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?;
                }
            }
            if let Some(check) = item.as_check_menuitem() {
                let mode = match id {
                    "plainmark.view-preview" => "preview",
                    "plainmark.view-editor" => "edit",
                    "plainmark.view-split" => "split",
                    _ => continue,
                };
                check
                    .set_checked(state.mode == mode)
                    .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?;
                check
                    .set_enabled(state.has_note && !state.busy)
                    .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?;
            }
            let Some(item) = item.as_menuitem() else {
                continue;
            };
            let enabled = match id {
                "plainmark.save" => state.has_note && state.dirty,
                "plainmark.find"
                | "plainmark.copy-path"
                | "plainmark.copy-relative-path"
                | "plainmark.reveal-note"
                | "plainmark.toggle-pin"
                | "plainmark.rename"
                | "plainmark.move"
                | "plainmark.delete" => state.has_note,
                "plainmark.close-tab" | "plainmark.close-all-tabs" => state.tab_count > 0,
                "plainmark.close-other-tabs" => state.has_note && state.tab_count > 1,
                "plainmark.previous-tab" | "plainmark.next-tab" => state.tab_count > 1,
                "plainmark.open-file"
                | "plainmark.open-folder"
                | "plainmark.new-note"
                | "plainmark.new-folder"
                | "plainmark.refresh" => true,
                "plainmark.toggle-sidebar" => {
                    item.set_text(if state.sidebar_visible {
                        labels.get("hideSidebar", "Hide Sidebar")
                    } else {
                        labels.get("showSidebar", "Show Sidebar")
                    })
                    .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?;
                    continue;
                }
                _ => continue,
            };
            item.set_enabled(enabled && !state.busy)
                .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?;
            if id == "plainmark.toggle-pin" {
                item.set_text(if state.pinned {
                    labels.get("unpin", "Unpin")
                } else {
                    labels.get("pin", "Pin")
                })
                    .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?;
            }
        }
    }
    Ok(())
}

#[tauri::command]
pub(crate) fn sync_recent_folders(
    app: AppHandle,
    labels: tauri::State<MenuLabelsState>,
    paths: Vec<String>,
) -> AppResult<()> {
    let labels = labels.current();
    let menu = app.menu().ok_or_else(|| AppError::new("menuUnavailable", "Application menu is unavailable."))?;
    let recent = menu
        .items()
        .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?
        .into_iter()
        .filter_map(|item| item.as_submenu().cloned())
        .find_map(|submenu| submenu.get("plainmark.recent-folders"))
        .and_then(|item| item.as_submenu().cloned())
        .ok_or_else(|| AppError::new("recentFoldersMenuUnavailable", "Recent Folders menu is unavailable."))?;
    while recent.remove_at(0).map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?.is_some() {}
    if paths.is_empty() {
        recent
            .append(
                &MenuItemBuilder::new(labels.get("noRecentFolders", "No Recent Folders"))
                    .enabled(false)
                    .build(&app)
                    .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?,
            )
            .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?;
    }
    for path in paths.into_iter().take(5) {
        let item = MenuItemBuilder::with_id(
            format!("plainmark.recent-folder:{path}"),
            path.replace('&', "&&"),
        )
        .build(&app)
        .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?;
        recent.append(&item).map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))?;
    }
    Ok(())
}

pub(crate) fn set_color_mode_menu_label<R: Runtime>(
    app: &AppHandle<R>,
    night_mode: bool,
    labels: &MenuLabels,
) -> AppResult<()> {
    let menu = app
        .menu()
        .ok_or_else(|| AppError::new("menuUnavailable", "Application menu is unavailable."))?;
    let view_menu = menu
        .get(VIEW_MENU_ID)
        .and_then(|item| item.as_submenu().cloned())
        .ok_or_else(|| AppError::new("viewMenuUnavailable", "View menu is unavailable."))?;
    let color_mode = view_menu
        .get(COLOR_MODE_ID)
        .and_then(|item| item.as_menuitem().cloned())
        .ok_or_else(|| AppError::new("colorModeItemUnavailable", "Color mode menu item is unavailable."))?;
    color_mode
        .set_text(if night_mode {
            labels.get("dayMode", "Day Mode")
        } else {
            labels.get("nightMode", "Night Mode")
        })
        .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))
}

pub(crate) fn set_encrypt_selection_enabled<R: Runtime>(
    app: &AppHandle<R>,
    enabled: bool,
) -> AppResult<()> {
    let menu = app
        .menu()
        .ok_or_else(|| AppError::new("menuUnavailable", "Application menu is unavailable."))?;
    let note_menu = menu
        .get(NOTE_MENU_ID)
        .and_then(|item| item.as_submenu().cloned())
        .ok_or_else(|| AppError::new("noteMenuUnavailable", "Note menu is unavailable."))?;
    let encrypt_selection = note_menu
        .get(ENCRYPT_SELECTION_ID)
        .and_then(|item| item.as_menuitem().cloned())
        .ok_or_else(|| AppError::new("encryptItemUnavailable", "Encrypt Selection menu item is unavailable."))?;
    encrypt_selection
        .set_enabled(enabled)
        .map_err(|error| AppError::new("menuUpdateFailed", error.to_string()))
}

#[cfg(test)]
mod tests {
    use std::collections::{BTreeSet, HashMap};

    #[test]
    fn frontend_menu_contract_covers_every_rust_label() {
        let contract: HashMap<String, String> =
            serde_json::from_str(include_str!("../../src/i18n/menu-contract.json")).unwrap();
        let source = include_str!("menu.rs");
        let rust_keys = source
            .split("labels.get(\"")
            .skip(1)
            .filter_map(|suffix| suffix.split('"').next())
            .collect::<BTreeSet<_>>();
        let contract_keys = contract.keys().map(String::as_str).collect::<BTreeSet<_>>();
        assert_eq!(rust_keys, contract_keys);
    }
}
