import { useCallback, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import { api } from "../api";
import type { FormState, MenuTarget, RefreshStatus } from "../appTypes";
import {
  activeFolderAfterDelete,
  baseName,
  folderTarget,
  parentFolder,
  removePinnedPaths,
  selectedFileLocation,
  type NoteContent,
  type PathResult,
  type TrashInfo,
} from "../model";
import type { TabSessionAction, TabSessionState } from "../tabSession";
import { t } from "../i18n";
import type { StatusTone } from "./useStatusMessage";

const refreshResetDelayMs = 1200;
const refreshErrorResetDelayMs = 1800;

type FileOperationsParams = {
  root: string;
  activeFolder: string;
  activeDocument: NoteContent | null;
  draft: string;
  form: FormState | null;
  trashInfo: TrashInfo | null;
  refreshStatus: RefreshStatus;
  setForm: Dispatch<SetStateAction<FormState | null>>;
  setTrashInfo: Dispatch<SetStateAction<TrashInfo | null>>;
  setRefreshStatus: Dispatch<SetStateAction<RefreshStatus>>;
  setMenu: Dispatch<SetStateAction<MenuTarget | null>>;
  activePathRef: MutableRefObject<string>;
  sessionRef: MutableRefObject<TabSessionState>;
  documentRef: MutableRefObject<NoteContent | null>;
  draftRef: MutableRefObject<string>;
  refreshTree: (preferred: string) => Promise<string>;
  loadNote: (path: string, disposition?: "replace" | "open" | "switch", mode?: "preview" | "edit" | "split") => Promise<void>;
  tabDispatch: (action: TabSessionAction) => void;
  treeChanged: (result: PathResult, oldPath?: string) => Promise<void>;
  rememberLocalTreeMutation: (result: PathResult) => void;
  updatePinnedPaths: (update: (paths: string[]) => string[]) => void;
  setActiveFolder: Dispatch<SetStateAction<string>>;
  switchRoot: (path: string, selectedPath?: string) => Promise<void>;
  showStatus: (text: string, tone?: StatusTone, timeoutMs?: number) => void;
  showError: (error: unknown, prefix?: string) => void;
};

/** User-initiated file operations: create, rename, move, delete, copy and refresh. */
export function useFileOperations(params: FileOperationsParams) {
  const {
    root,
    activeFolder,
    activeDocument,
    draft,
    form,
    trashInfo,
    refreshStatus,
    setForm,
    setTrashInfo,
    setRefreshStatus,
    setMenu,
    activePathRef,
    sessionRef,
    documentRef,
    draftRef,
    refreshTree,
    loadNote,
    tabDispatch,
    treeChanged,
    rememberLocalTreeMutation,
    updatePinnedPaths,
    setActiveFolder,
    switchRoot,
    showStatus,
    showError,
  } = params;

  const submitForm = useCallback(
    async (value: string) => {
      if (!form) return;
      try {
        const operation = form.operation;
        let result: Awaited<ReturnType<typeof api.createNote>>;
        if (operation === "note") result = await api.createNote(form.kind === "folder" ? form.path : "", value);
        else if (operation === "folder") result = await api.createFolder(form.kind === "folder" ? form.path : "", value);
        else if (operation === "rename") result = await api.renamePath(form.path, value);
        else result = await api.movePath(form.path, value);
        const oldPath = operation === "rename" || operation === "move" ? form.path : undefined;
        setForm(null);
        await treeChanged(result, oldPath);
        const messageKey = operation === "note" ? "files.created" : operation === "folder" ? "files.createdFolder" : operation === "rename" ? "files.renamed" : "files.moved";
        showStatus(t(messageKey, { path: result.path }), "success");
      } catch (error) {
        showError(error);
      }
    },
    [form, setForm, showError, showStatus, treeChanged]
  );

  const requestDelete = useCallback(
    async (path: string) => {
      try {
        setTrashInfo(await api.getTrashInfo(path));
      } catch (error) {
        showError(error);
      }
    },
    [setTrashInfo, showError]
  );

  const confirmDelete = useCallback(async () => {
    if (!trashInfo) return;
    try {
      const path = trashInfo.path;
      await api.trashPath(path);
      setTrashInfo(null);
      updatePinnedPaths((paths) => removePinnedPaths(paths, path));
      setActiveFolder((current) => activeFolderAfterDelete(current, path));
      const activeDeleted = activePathRef.current === path || activePathRef.current.startsWith(`${path}/`);
      tabDispatch({ type: "remove-tree", path });
      const remaining = sessionRef.current.tabs;
      const next = await refreshTree(sessionRef.current.activePath);
      if (activeDeleted) {
        const target = sessionRef.current.activePath || next;
        if (target) {
          const targetSession = sessionRef.current.tabs.find((tab) => tab.path === target);
          if (!targetSession?.document) await loadNote(target, remaining.length ? "switch" : "open", targetSession?.mode || "preview");
        } else {
          await loadNote("");
        }
      }
      showStatus(t("files.trashed", { path }), "success");
    } catch (error) {
      showError(error);
    }
  }, [activePathRef, loadNote, refreshTree, sessionRef, setActiveFolder, setTrashInfo, showError, showStatus, tabDispatch, trashInfo, updatePinnedPaths]);

  const saveCopy = useCallback(async () => {
    if (!activeDocument) return;
    const raw = window.prompt(t("files.copyPrompt"), `${baseName(activeDocument.path).replace(/\.mdx?$/i, "")}-copy.md`);
    if (!raw) return;
    try {
      const created = await api.createNote(parentFolder(activeDocument.path), raw);
      rememberLocalTreeMutation(created);
      const fresh = await api.readNote(created.path);
      const saved = await api.saveNote(created.path, draft, fresh.modified_ms);
      await refreshTree(saved.path);
      await loadNote(saved.path);
      showStatus(t("files.savedCopy", { path: saved.path }), "success");
    } catch (error) {
      showError(error);
    }
  }, [activeDocument, draft, loadNote, refreshTree, rememberLocalTreeMutation, showError, showStatus]);

  const copyPath = useCallback(
    async (path: string) => {
      try {
        const absolute = await api.absoluteNotePath(path);
        await api.copyText(absolute);
        showStatus(t("files.copiedPath", { path: absolute }), "success");
      } catch (error) {
        showError(error, t("files.copyFailed"));
      } finally {
        setMenu(null);
      }
    },
    [setMenu, showError, showStatus]
  );

  const chooseRoot = useCallback(
    async (path?: string) => {
      if (path) {
        try {
          await switchRoot(path);
        } catch (error) {
          showError(error, t("files.openFolderFailed"));
        }
        return;
      }
      try {
        const selected = await api.chooseFolder(root || undefined, t("dialog.openFolderTitle"));
        if (!selected || Array.isArray(selected) || selected === root) return;
        await switchRoot(selected);
      } catch (error) {
        showError(error, t("files.openFolderFailed"));
      }
    },
    [root, showError, switchRoot]
  );

  const chooseFile = useCallback(async () => {
    try {
      const selected = await api.chooseMarkdownFile(root || undefined, t("dialog.openFileTitle"));
      if (!selected || Array.isArray(selected)) return;
      const location = selectedFileLocation(selected);
      if (!location || !/\.mdx?$/i.test(location.fileName)) {
        showStatus(t("files.chooseMarkdown"), "error", 0);
        return;
      }
      await switchRoot(location.root, location.fileName);
    } catch (error) {
      showError(error, t("files.openFileFailed"));
    }
  }, [root, showError, showStatus, switchRoot]);

  const manualRefresh = useCallback(async () => {
    if (refreshStatus === "refreshing") return;
    setRefreshStatus("refreshing");
    const current = activePathRef.current;
    try {
      const next = await refreshTree(current);
      if (activePathRef.current === current) {
        const doc = documentRef.current;
        if (!doc || draftRef.current === doc.markdown) {
          await loadNote(next);
          showStatus(t("files.refreshed"), "success");
        } else {
          showStatus(t("files.refreshedDraftKept"));
        }
      }
      setRefreshStatus("success");
      window.setTimeout(() => setRefreshStatus("idle"), refreshResetDelayMs);
    } catch (error) {
      setRefreshStatus("error");
      showError(error);
      window.setTimeout(() => setRefreshStatus("idle"), refreshErrorResetDelayMs);
    }
  }, [activePathRef, documentRef, draftRef, loadNote, refreshStatus, refreshTree, setRefreshStatus, showError, showStatus]);

  const createInActiveFolder = useCallback(
    (operation: "note" | "folder") => setForm({ operation, ...folderTarget(activeFolder) }),
    [activeFolder, setForm]
  );

  return { submitForm, requestDelete, confirmDelete, saveCopy, copyPath, chooseRoot, chooseFile, manualRefresh, createInActiveFolder };
}
