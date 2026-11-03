import { useCallback, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import type { MenuActionId } from "../api";
import type { FormState, Mode } from "../appTypes";
import type { NoteContent } from "../model";
import { toggledColorMode } from "../settings";
import { t } from "../i18n";
import type { StatusTone } from "./useStatusMessage";

type MenuActionsParams = {
  activePathRef: MutableRefObject<string>;
  documentRef: MutableRefObject<NoteContent | null>;
  draftRef: MutableRefObject<string>;
  colorMode: "light" | "dark";
  setColorModeOverride: Dispatch<SetStateAction<"light" | "dark" | null>>;
  setSidebarVisible: Dispatch<SetStateAction<boolean>>;
  setSettingsOpen: Dispatch<SetStateAction<boolean>>;
  setForm: Dispatch<SetStateAction<FormState | null>>;
  setMode: (mode: Mode) => void;
  chooseRoot: (path?: string) => Promise<void>;
  chooseFile: () => Promise<void>;
  createInActiveFolder: (operation: "note" | "folder") => void;
  cycleTab: (direction: -1 | 1) => void;
  manualRefresh: () => Promise<void>;
  save: () => Promise<boolean>;
  copyPath: (path: string) => Promise<void>;
  requestDelete: (path: string) => Promise<void>;
  requestCloseTab: (path: string) => void;
  requestCloseTabs: (othersOnly: boolean) => void;
  findInNote: () => void;
  searchNotes: () => void;
  togglePin: (path: string) => void;
  copyRelativePath: (path: string) => Promise<void>;
  revealNote: (path: string) => Promise<void>;
  requestQuit: () => void;
  requestEncryptSelection: () => void;
  showStatus: (text: string, tone?: StatusTone, timeoutMs?: number) => void;
};

/** Maps native menu events to app actions. */
export function useMenuActions(params: MenuActionsParams) {
  const {
    activePathRef,
    documentRef,
    draftRef,
    colorMode,
    setColorModeOverride,
    setSidebarVisible,
    setSettingsOpen,
    setForm,
    setMode,
    chooseRoot,
    chooseFile,
    createInActiveFolder,
    cycleTab,
    manualRefresh,
    save,
    copyPath,
    requestDelete,
    requestCloseTab,
    requestCloseTabs,
    findInNote,
    searchNotes,
    togglePin,
    copyRelativePath,
    revealNote,
    requestQuit,
    requestEncryptSelection,
    showStatus,
  } = params;

  return useCallback(
    (id: MenuActionId) => {
      const active = activePathRef.current;
      if (id.startsWith("plainmark.recent-folder:")) {
        void chooseRoot(id.slice("plainmark.recent-folder:".length));
        return;
      }
      const withActiveNote = (action: () => void) => {
        if (active) action();
        else showStatus(t("menu.openNoteFirst"));
      };
      switch (id) {
        case "plainmark.find":
          withActiveNote(findInNote);
          break;
        case "plainmark.search-notes":
          searchNotes();
          break;
        case "plainmark.toggle-pin":
          withActiveNote(() => togglePin(active));
          break;
        case "plainmark.copy-relative-path":
          withActiveNote(() => void copyRelativePath(active));
          break;
        case "plainmark.reveal-note":
          withActiveNote(() => void revealNote(active));
          break;
        case "plainmark.close-other-tabs":
          requestCloseTabs(true);
          break;
        case "plainmark.close-all-tabs":
          requestCloseTabs(false);
          break;
        case "plainmark.open-folder":
          void chooseRoot();
          break;
        case "plainmark.open-file":
          void chooseFile();
          break;
        case "plainmark.settings":
          setSettingsOpen(true);
          break;
        case "plainmark.quit":
          requestQuit();
          break;
        case "plainmark.new-note":
          createInActiveFolder("note");
          break;
        case "plainmark.new-folder":
          createInActiveFolder("folder");
          break;
        case "plainmark.save":
          if (documentRef.current && draftRef.current !== documentRef.current.markdown) void save();
          break;
        case "plainmark.close-tab":
          if (active) requestCloseTab(active);
          break;
        case "plainmark.previous-tab":
          cycleTab(-1);
          break;
        case "plainmark.next-tab":
          cycleTab(1);
          break;
        case "plainmark.refresh":
          void manualRefresh();
          break;
        case "plainmark.view-preview":
          setMode("preview");
          break;
        case "plainmark.view-editor":
          setMode("edit");
          break;
        case "plainmark.view-split":
          setMode("split");
          break;
        case "plainmark.toggle-sidebar":
          setSidebarVisible((value) => !value);
          break;
        case "plainmark.toggle-color-mode":
          setColorModeOverride(toggledColorMode(colorMode));
          break;
        case "plainmark.encrypt-selection":
          requestEncryptSelection();
          break;
        case "plainmark.copy-path":
          withActiveNote(() => void copyPath(active));
          break;
        case "plainmark.rename":
          withActiveNote(() => setForm({ operation: "rename", path: active, kind: "note" }));
          break;
        case "plainmark.move":
          withActiveNote(() => setForm({ operation: "move", path: active, kind: "note" }));
          break;
        case "plainmark.delete":
          withActiveNote(() => void requestDelete(active));
          break;
      }
    },
    [
      activePathRef,
      chooseFile,
      chooseRoot,
      colorMode,
      copyPath,
      createInActiveFolder,
      cycleTab,
      documentRef,
      draftRef,
      manualRefresh,
      requestCloseTab,
      requestCloseTabs,
      findInNote,
      searchNotes,
      togglePin,
      copyRelativePath,
      revealNote,
      requestDelete,
      requestEncryptSelection,
      requestQuit,
      save,
      setColorModeOverride,
      setForm,
      setMode,
      setSettingsOpen,
      setSidebarVisible,
      showStatus,
    ]
  );
}
