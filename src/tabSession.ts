import type { Mode, SaveStatus } from "./appTypes";
import { remapPath, type NoteContent, type OpenNoteTab, type TabDisposition } from "./model";

export type TabScrollPositions = Partial<Record<Mode, number>>;

export type TabSession = OpenNoteTab & {
  document: NoteContent | null;
  draft: string;
  saveStatus: SaveStatus;
  saveError: string;
  external: NoteContent | null;
  scroll: TabScrollPositions;
};

export type TabSessionState = {
  tabs: TabSession[];
  activePath: string;
};

export type TabSessionAction =
  | { type: "restore"; tabs: OpenNoteTab[]; activePath: string }
  | { type: "load"; note: NoteContent; disposition: TabDisposition; mode: Mode; preserveDraft?: boolean }
  | { type: "activate-empty" }
  | { type: "draft"; path: string; draft: string }
  | { type: "mode"; path: string; mode: Mode }
  | { type: "scroll"; path: string; mode: Mode; top: number }
  | { type: "save-start"; path: string }
  | { type: "save-success"; path: string; note: NoteContent; submittedDraft: string }
  | { type: "save-error"; path: string; status: "error" | "conflict"; message: string }
  | { type: "save-idle"; path: string }
  | { type: "external-conflict"; path: string; note: NoteContent | null }
  | { type: "replace-from-disk"; path: string; note: NoteContent }
  | { type: "close"; path: string }
  | { type: "close-many"; paths: string[] }
  | { type: "remove-tree"; path: string }
  | { type: "reorder"; path: string; overPath: string }
  | { type: "remap"; from: string; to: string }
  | { type: "reset" };

export const emptyTabSessionState: TabSessionState = { tabs: [], activePath: "" };

export function createTabSession(tab: OpenNoteTab, document: NoteContent | null = null): TabSession {
  return {
    ...tab,
    document,
    draft: document?.markdown ?? "",
    saveStatus: "idle",
    saveError: "",
    external: null,
    scroll: {},
  };
}

export function activeTabSession(state: TabSessionState) {
  return state.tabs.find(tab => tab.path === state.activePath) ?? null;
}

export function isTabSessionDirty(tab: TabSession | null | undefined) {
  return Boolean(tab?.document && tab.draft !== tab.document.markdown);
}

export function dirtyTabSessionPaths(state: TabSessionState) {
  return state.tabs.filter(isTabSessionDirty).map(tab => tab.path);
}

function updateTab(state: TabSessionState, path: string, update: (tab: TabSession) => TabSession): TabSessionState {
  if (!state.tabs.some(tab => tab.path === path)) return state;
  return { ...state, tabs: state.tabs.map(tab => tab.path === path ? update(tab) : tab) };
}

export function tabSessionReducer(state: TabSessionState, action: TabSessionAction): TabSessionState {
  switch (action.type) {
    case "close-many": {
      const removed = new Set(action.paths);
      const tabs = state.tabs.filter(tab => !removed.has(tab.path));
      const activePath = tabs.some(tab => tab.path === state.activePath) ? state.activePath : tabs[0]?.path ?? "";
      return { tabs, activePath };
    }
    case "restore": {
      const tabs = action.tabs.map(tab => createTabSession(tab));
      const activePath = tabs.some(tab => tab.path === action.activePath) ? action.activePath : tabs[0]?.path ?? "";
      return { tabs, activePath };
    }
    case "load": {
      const existingIndex = state.tabs.findIndex(tab => tab.path === action.note.path);
      const existing = state.tabs[existingIndex];
      const loaded = existing
        ? {
            ...existing,
            title: action.note.title,
            mode: existing.mode,
            document: action.preserveDraft && existing.document ? existing.document : action.note,
            draft: action.preserveDraft && existing.document ? existing.draft : action.note.markdown,
            saveStatus: action.preserveDraft ? existing.saveStatus : "idle" as const,
            saveError: action.preserveDraft ? existing.saveError : "",
            external: action.preserveDraft ? existing.external : null,
          }
        : createTabSession({ path: action.note.path, title: action.note.title, mode: action.mode }, action.note);
      let tabs: TabSession[];
      if (existingIndex >= 0) {
        tabs = state.tabs.map((tab, index) => index === existingIndex ? loaded : tab);
      } else if (action.disposition === "replace" && state.activePath) {
        tabs = state.tabs.map(tab => tab.path === state.activePath ? loaded : tab);
      } else {
        tabs = [...state.tabs, loaded];
      }
      return { tabs, activePath: action.note.path };
    }
    case "activate-empty":
      return { ...state, activePath: "" };
    case "draft":
      return updateTab(state, action.path, tab => ({ ...tab, draft: action.draft, saveStatus: tab.saveStatus === "saved" ? "idle" : tab.saveStatus }));
    case "mode":
      return updateTab(state, action.path, tab => ({ ...tab, mode: action.mode }));
    case "scroll":
      return updateTab(state, action.path, tab => ({ ...tab, scroll: { ...tab.scroll, [action.mode]: Math.max(0, action.top) } }));
    case "save-start":
      return updateTab(state, action.path, tab => ({ ...tab, saveStatus: "saving", saveError: "" }));
    case "save-success":
      return updateTab(state, action.path, tab => {
        const hasNewerDraft = tab.draft !== action.submittedDraft;
        return {
          ...tab,
          title: action.note.title,
          document: action.note,
          draft: hasNewerDraft ? tab.draft : action.note.markdown,
          saveStatus: hasNewerDraft ? "idle" : "saved",
          saveError: "",
          external: null,
        };
      });
    case "save-error":
      return updateTab(state, action.path, tab => ({ ...tab, saveStatus: action.status, saveError: action.message }));
    case "save-idle":
      return updateTab(state, action.path, tab => ({ ...tab, saveStatus: "idle", saveError: "" }));
    case "external-conflict":
      return updateTab(state, action.path, tab => ({ ...tab, external: action.note, saveStatus: "conflict" }));
    case "replace-from-disk":
      return updateTab(state, action.path, tab => ({ ...tab, title: action.note.title, document: action.note, draft: action.note.markdown, external: null, saveStatus: "idle", saveError: "" }));
    case "close": {
      const index = state.tabs.findIndex(tab => tab.path === action.path);
      if (index < 0) return state;
      const tabs = state.tabs.filter(tab => tab.path !== action.path);
      const activePath = state.activePath === action.path ? tabs[Math.min(index, tabs.length - 1)]?.path ?? "" : state.activePath;
      return { tabs, activePath };
    }
    case "remove-tree": {
      const removed = (path: string) => path === action.path || path.startsWith(`${action.path}/`);
      const tabs = state.tabs.filter(tab => !removed(tab.path));
      return { tabs, activePath: removed(state.activePath) ? tabs[0]?.path ?? "" : state.activePath };
    }
    case "reorder": {
      const from = state.tabs.findIndex(tab => tab.path === action.path);
      const to = state.tabs.findIndex(tab => tab.path === action.overPath);
      if (from < 0 || to < 0 || from === to) return state;
      const tabs = [...state.tabs];
      const [moved] = tabs.splice(from, 1);
      tabs.splice(to, 0, moved);
      return { ...state, tabs };
    }
    case "remap":
      return {
        activePath: remapPath(state.activePath, action.from, action.to),
        tabs: state.tabs.map(tab => {
          const path = remapPath(tab.path, action.from, action.to);
          return { ...tab, path, document: tab.document ? { ...tab.document, path } : null, external: tab.external ? { ...tab.external, path } : null };
        }),
      };
    case "reset":
      return emptyTabSessionState;
  }
}
