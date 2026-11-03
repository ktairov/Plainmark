import { useCallback, useRef, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import { api } from "../api";
import { folderAncestors, parentFolder, remapPath, remapPinnedPaths, type PathResult } from "../model";
import type { TabSessionAction } from "../tabSession";
import { t } from "../i18n";
import type { StatusTone } from "./useStatusMessage";

type TreeOperationsParams = {
  activePathRef: MutableRefObject<string>;
  refreshTree: (preferred: string) => Promise<string>;
  loadNote: (path: string, disposition?: "replace" | "open" | "switch", mode?: "preview" | "edit" | "split") => Promise<void>;
  tabDispatch: (action: TabSessionAction) => void;
  setExpanded: Dispatch<SetStateAction<Set<string>>>;
  setActiveFolder: Dispatch<SetStateAction<string>>;
  updatePinnedPaths: (update: (paths: string[]) => string[]) => void;
  showStatus: (text: string, tone?: StatusTone, timeoutMs?: number) => void;
  showError: (error: unknown, prefix?: string) => void;
};

type LocalTreeMutation = PathResult & { expiresAt: number };

/** Tree mutations triggered by the app itself, briefly ignored by the watcher events. */
export function useTreeOperations(params: TreeOperationsParams) {
  const { activePathRef, refreshTree, loadNote, tabDispatch, setExpanded, setActiveFolder, updatePinnedPaths, showStatus, showError } = params;
  const localTreeMutationsRef = useRef<LocalTreeMutation[]>([]);

  const rememberLocalTreeMutation = useCallback((result: PathResult) => {
    const now = Date.now();
    localTreeMutationsRef.current = [...localTreeMutationsRef.current.filter((item) => item.expiresAt > now), { ...result, expiresAt: now + 4000 }];
  }, []);

  const isLocalTreeAddition = useCallback((path: string) => {
    const now = Date.now();
    localTreeMutationsRef.current = localTreeMutationsRef.current.filter((item) => item.expiresAt > now);
    return localTreeMutationsRef.current.some((item) => path === item.path || (item.kind === "folder" && path.startsWith(`${item.path}/`)));
  }, []);

  const treeChanged = useCallback(
    async (result: PathResult, oldPath?: string) => {
      rememberLocalTreeMutation(result);
      let selected = result.kind === "note" ? result.path : activePathRef.current;
      if (oldPath && activePathRef.current && (activePathRef.current === oldPath || activePathRef.current.startsWith(`${oldPath}/`))) {
        selected = remapPath(activePathRef.current, oldPath, result.path);
      }
      if (oldPath) tabDispatch({ type: "remap", from: oldPath, to: result.path });
      if (oldPath) setActiveFolder((current) => remapPath(current, oldPath, result.path));
      if (oldPath) setExpanded((current) => new Set([...current].map((path) => remapPath(path, oldPath, result.path))));
      if (oldPath) updatePinnedPaths((paths) => remapPinnedPaths(paths, oldPath, result.path));
      setExpanded((current) => {
        const next = new Set(current);
        folderAncestors(result.path).forEach((path) => next.add(path));
        if (result.kind === "folder") next.add(result.path);
        return next;
      });
      if (result.kind === "folder" && !oldPath) setActiveFolder(result.path);
      await refreshTree(selected);
      if (result.kind === "note" && !oldPath) await loadNote(result.path);
    },
    [activePathRef, loadNote, refreshTree, rememberLocalTreeMutation, setActiveFolder, setExpanded, tabDispatch, updatePinnedPaths]
  );

  const moveByDrop = useCallback(
    async (path: string, destination: string) => {
      if (parentFolder(path) === destination) {
        showStatus(t("files.alreadyInFolder", { path }));
        return;
      }
      try {
        const result = await api.movePath(path, destination);
        await treeChanged(result, path);
        showStatus(t("files.moved", { path: result.path }), "success");
      } catch (error) {
        showError(error);
        await refreshTree(activePathRef.current);
      }
    },
    [activePathRef, refreshTree, showError, showStatus, treeChanged]
  );

  return { treeChanged, moveByDrop, rememberLocalTreeMutation, isLocalTreeAddition };
}
