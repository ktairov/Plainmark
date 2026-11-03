import { parsePinnedNotes, type PinnedNotesByRoot } from "./model";
import { parseAppSettings, type AppSettings } from "./settings";

export const storageKeys = {
  expanded: "plainmark-tree-expanded-folders",
  width: "plainmark-sidebar-width",
  sidebarVisible: "plainmark-sidebar-visible",
  settings: "plainmark-app-settings",
  pinnedNotes: "plainmark-pinned-notes",
  recentRoots: "plainmark-recent-roots",
  tabs: (root: string) => `plainmark-open-tabs:${root}`,
};

const legacyKeys = new Set([
  "notes-tree-expanded-folders",
  "notes-sidebar-width",
  "notes-sidebar-visible",
  "notes-app-settings",
]);

export const defaultSidebarWidth = 340;

let legacyStorageCleaned = false;

/** Removes state written by pre-Plainmark builds. Runs once per session. */
export function cleanupLegacyStorage() {
  if (legacyStorageCleaned) return;
  legacyStorageCleaned = true;
  try {
    for (let index = localStorage.length - 1; index >= 0; index -= 1) {
      const key = localStorage.key(index);
      if (key && (legacyKeys.has(key) || key.startsWith("notes-open-tabs:"))) localStorage.removeItem(key);
    }
  } catch {
    /* Storage cleanup must not prevent Plainmark from launching. */
  }
}

export function loadExpandedFolders() {
  try {
    const value = JSON.parse(localStorage.getItem(storageKeys.expanded) || "[]");
    return new Set<string>(Array.isArray(value) ? value : []);
  } catch {
    return new Set<string>();
  }
}

export function loadSidebarWidth() {
  const width = Number(localStorage.getItem(storageKeys.width));
  return Number.isFinite(width) && width >= 240 ? width : defaultSidebarWidth;
}

export function loadSidebarVisible() {
  return localStorage.getItem(storageKeys.sidebarVisible) !== "false";
}

export function loadSettings() {
  return parseAppSettings(localStorage.getItem(storageKeys.settings));
}

export function loadPinnedNotes() {
  return parsePinnedNotes(localStorage.getItem(storageKeys.pinnedNotes));
}

export function loadRecentRoots() {
  try {
    const value = JSON.parse(localStorage.getItem(storageKeys.recentRoots) || "[]");
    return Array.isArray(value) ? value.filter((path): path is string => typeof path === "string" && Boolean(path)).slice(0, 5) : [];
  } catch {
    return [];
  }
}

export type StoredAppState = {
  expanded: Set<string>;
  width: number;
  sidebarVisible: boolean;
  settings: AppSettings;
  pinnedNotes: PinnedNotesByRoot;
  recentRoots: string[];
};

export function loadAppState(): StoredAppState {
  return {
    expanded: loadExpandedFolders(),
    width: loadSidebarWidth(),
    sidebarVisible: loadSidebarVisible(),
    settings: loadSettings(),
    pinnedNotes: loadPinnedNotes(),
    recentRoots: loadRecentRoots(),
  };
}
