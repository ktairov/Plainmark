import { useCallback, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import { api } from "../api";
import { rootFolderName, type NoteEntry } from "../model";
import type { TabSessionAction } from "../tabSession";
import { t } from "../i18n";
import type { StatusTone } from "./useStatusMessage";

type RootSwitcherParams = {
  notesRef: MutableRefObject<NoteEntry[]>;
  refreshTree: (preferred: string) => Promise<string>;
  cancelPendingRefresh: () => void;
  cancelPendingLoad: () => void;
  clearWorkspace: () => void;
  setRoot: Dispatch<SetStateAction<string>>;
  setTabsReady: Dispatch<SetStateAction<boolean>>;
  setSidebarVisible: Dispatch<SetStateAction<boolean>>;
  setActiveFolder: Dispatch<SetStateAction<string>>;
  setExpanded: Dispatch<SetStateAction<Set<string>>>;
  setQuery: Dispatch<SetStateAction<string>>;
  rememberRoot: (path: string) => void;
  tabDispatch: (action: TabSessionAction) => void;
  loadNote: (path: string, disposition?: "replace" | "open" | "switch", mode?: "preview" | "edit" | "split") => Promise<void>;
  restoreTabs: (rootPath: string, fallback: string) => Promise<void>;
  showStatus: (text: string, tone?: StatusTone, timeoutMs?: number) => void;
};

export function useRootSwitcher(params: RootSwitcherParams) {
  const {
    notesRef,
    refreshTree,
    cancelPendingRefresh,
    cancelPendingLoad,
    clearWorkspace,
    setRoot,
    setTabsReady,
    setSidebarVisible,
    setActiveFolder,
    setExpanded,
    setQuery,
    rememberRoot,
    tabDispatch,
    loadNote,
    restoreTabs,
    showStatus,
  } = params;

  return useCallback(
    async (path: string, selectedPath = "") => {
      cancelPendingLoad();
      cancelPendingRefresh();
      showStatus(selectedPath ? t("root.openingFile") : t("root.openingFolder"), "info", 0);
      setTabsReady(false);
      const config = await api.setNotesRoot(path);
      tabDispatch({ type: "reset" });
      setRoot(config.root);
      setActiveFolder("");
      clearWorkspace();
      setExpanded(new Set());
      setQuery("");
      const first = await refreshTree("");
      if (selectedPath) {
        if (!notesRef.current.some((note) => note.path === selectedPath)) {
          setTabsReady(true);
          throw new Error(t("root.selectedFileUnavailable"));
        }
        await loadNote(selectedPath, "open");
        setTabsReady(true);
        setSidebarVisible(false);
        showStatus(t("app.status.opened", { path: selectedPath }), "success");
      } else {
        await restoreTabs(config.root, first);
        showStatus(t("root.openedFolder", { name: rootFolderName(config.root) }), "success");
        rememberRoot(path);
      }
    },
    [
      cancelPendingLoad,
      cancelPendingRefresh,
      clearWorkspace,
      loadNote,
      notesRef,
      refreshTree,
      rememberRoot,
      restoreTabs,
      setActiveFolder,
      setExpanded,
      setQuery,
      setRoot,
      setSidebarVisible,
      setTabsReady,
      showStatus,
      tabDispatch,
    ]
  );
}
