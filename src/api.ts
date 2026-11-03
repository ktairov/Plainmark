import { invoke } from "@tauri-apps/api/core";
import { listen, type UnlistenFn } from "@tauri-apps/api/event";
import { getCurrentWindow, type Theme } from "@tauri-apps/api/window";
import { writeText } from "@tauri-apps/plugin-clipboard-manager";
import { open } from "@tauri-apps/plugin-dialog";
import type { NoteContent, PathResult, TrashInfo, WorkspaceSnapshot } from "./model";

export type WorkspaceConfig = { root: string };
export type FilesChangedEvent = { paths: string[] };

export const menuActionIds = [
  "plainmark.open-folder", "plainmark.open-file", "plainmark.settings", "plainmark.quit",
  "plainmark.new-note", "plainmark.new-folder", "plainmark.save", "plainmark.close-tab",
  "plainmark.previous-tab", "plainmark.next-tab", "plainmark.refresh", "plainmark.view-preview",
  "plainmark.view-editor", "plainmark.view-split", "plainmark.toggle-sidebar",
  "plainmark.toggle-color-mode", "plainmark.encrypt-selection", "plainmark.copy-path",
  "plainmark.rename", "plainmark.move", "plainmark.delete",
  "plainmark.find", "plainmark.search-notes", "plainmark.toggle-pin", "plainmark.copy-relative-path",
  "plainmark.reveal-note", "plainmark.close-other-tabs", "plainmark.close-all-tabs",
] as const;

export type MenuActionId = typeof menuActionIds[number] | `plainmark.recent-folder:${string}`;

export type MenuState = { hasNote: boolean; dirty: boolean; busy: boolean; tabCount: number; mode: string; sidebarVisible: boolean; pinned: boolean };

export const api = {
  getConfig: () => invoke<WorkspaceConfig>("get_config"),
  setNotesRoot: (path: string) => invoke<WorkspaceConfig>("set_notes_root", { path }),
  scanWorkspace: () => invoke<WorkspaceSnapshot>("scan_workspace"),
  readNote: (path: string) => invoke<NoteContent>("read_note", { path }),
  saveNote: (path: string, markdown: string, expectedModifiedMs: number, force = false) =>
    invoke<NoteContent>("save_note", { path, markdown, expectedModifiedMs, force }),
  createNote: (parent: string, fileName: string, title: string | null = null) =>
    invoke<PathResult>("create_note", { parent, fileName, title }),
  createFolder: (parent: string, folderName: string) =>
    invoke<PathResult>("create_folder", { parent, folderName }),
  renamePath: (path: string, newName: string) => invoke<PathResult>("rename_path", { path, newName }),
  movePath: (path: string, destinationFolder: string) => invoke<PathResult>("move_path", { path, destinationFolder }),
  getTrashInfo: (path: string) => invoke<TrashInfo>("get_trash_info", { path }),
  trashPath: (path: string) => invoke<void>("trash_path", { path }),
  encryptText: (plaintext: string, password: string) => invoke<string>("encrypt_text", { plaintext, password }),
  decryptText: (payload: string, password: string) => invoke<string>("decrypt_text", { payload, password }),
  setEncryptSelectionEnabled: (enabled: boolean) => invoke<void>("set_encrypt_selection_enabled", { enabled }),
  setColorModeMenuLabel: (nightMode: boolean) => invoke<void>("set_color_mode_menu_label", { nightMode }),
  previewAppLanguage: (language: string, labels: Record<string, string>) => invoke<void>("preview_app_language", { language, labels }),
  setAppLanguage: (language: string, labels: Record<string, string>) => invoke<void>("set_app_language", { language, labels }),
  hideMainWindow: () => invoke<void>("hide_main_window"),
  revealMainWindow: () => invoke<void>("reveal_main_window"),
  windowTheme: () => getCurrentWindow().theme(),
  exitApp: () => invoke<void>("exit_app"),
  copyText: (text: string) => writeText(text),
  absoluteNotePath: (path: string) => invoke<string>("absolute_note_path", { path }),
  revealNote: (path: string) => invoke<void>("reveal_note", { path }),
  syncMenuState: (state: MenuState) => invoke<void>("sync_menu_state", { state }),
  syncRecentFolders: (paths: string[]) => invoke<void>("sync_recent_folders", { paths }),
  chooseFolder: (defaultPath?: string, title = "Open Markdown Folder") => open({ directory: true, multiple: false, defaultPath, title }),
  chooseMarkdownFile: (defaultPath?: string, title = "Open Markdown File") => open({ directory: false, multiple: false, defaultPath, title, filters: [{ name: "Markdown", extensions: ["md", "mdx"] }] }),
};

export const events = {
  onWindowThemeChanged: (handler: (theme: Theme) => void): Promise<UnlistenFn> =>
    getCurrentWindow().onThemeChanged(event => handler(event.payload)),
  onMenuAction: (handler: (action: MenuActionId) => void): Promise<UnlistenFn> =>
    listen<string>("app-menu", event => handler(event.payload as MenuActionId)),
  onWindowCloseRequested: (handler: () => void): Promise<UnlistenFn> => listen("window-close-requested", handler),
  onAppExitRequested: (handler: () => void): Promise<UnlistenFn> => listen("app-exit-requested", handler),
  onFilesChanged: (handler: (event: FilesChangedEvent) => void | Promise<void>): Promise<UnlistenFn> =>
    listen<FilesChangedEvent>("plainmark-files-changed", event => handler(event.payload)),
  onWatchError: (handler: (message: string) => void): Promise<UnlistenFn> =>
    listen<string>("plainmark-watch-error", event => handler(event.payload)),
};
