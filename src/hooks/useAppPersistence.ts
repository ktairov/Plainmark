import { useCallback } from "react";
import type { PinnedNotesByRoot, OpenNoteTab } from "../model";
import type { AppSettings } from "../settings";
import {
  loadExpandedFolders,
  loadPinnedNotes,
  loadRecentRoots,
  loadSettings,
  loadSidebarVisible,
  loadSidebarWidth,
  storageKeys,
} from "../storage";
import { usePersistentState } from "./usePersistentState";

export function useAppPersistence() {
  const [expanded, setExpanded] = usePersistentState(storageKeys.expanded, loadExpandedFolders, (value) => JSON.stringify([...value]));
  const [width, setWidth] = usePersistentState(storageKeys.width, loadSidebarWidth);
  const [sidebarVisible, setSidebarVisible] = usePersistentState(storageKeys.sidebarVisible, loadSidebarVisible);
  const [settings, setSettings] = usePersistentState<AppSettings>(storageKeys.settings, loadSettings);
  const [pinnedByRoot, setPinnedByRoot] = usePersistentState<PinnedNotesByRoot>(storageKeys.pinnedNotes, loadPinnedNotes);
  const [recentRoots, setRecentRoots] = usePersistentState<string[]>(storageKeys.recentRoots, loadRecentRoots);

  const rememberRoot = useCallback(
    (path: string) => setRecentRoots((current) => [path, ...current.filter((candidate) => candidate !== path)].slice(0, 5)),
    [setRecentRoots]
  );

  const persistTabs = useCallback((root: string, tabs: OpenNoteTab[], activePath: string) => {
    localStorage.setItem(storageKeys.tabs(root), JSON.stringify({ tabs: tabs.map(({ path, mode }) => ({ path, mode })), active: activePath }));
  }, []);

  return {
    expanded,
    setExpanded,
    width,
    setWidth,
    sidebarVisible,
    setSidebarVisible,
    settings,
    setSettings,
    pinnedByRoot,
    setPinnedByRoot,
    recentRoots,
    rememberRoot,
    persistTabs,
  };
}
