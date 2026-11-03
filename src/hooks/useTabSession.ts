import { useCallback, useEffect, useLayoutEffect, useMemo, useReducer, useRef, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import { EditorView } from "@codemirror/view";
import { markdown as markdownLanguage } from "@codemirror/lang-markdown";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { tags } from "@lezer/highlight";
import { api } from "../api";
import { errorKind, errorMessage, folderAncestors, parseStoredTabs, restoredOpenTabs, shouldOpenFallbackNote, type NoteEntry, type TabDisposition } from "../model";
import { activeTabSession, dirtyTabSessionPaths, emptyTabSessionState, isTabSessionDirty, tabSessionReducer, type TabSessionAction, type TabSessionState } from "../tabSession";
import type { Mode, Pending, SaveStatus } from "../appTypes";
import { t } from "../i18n";
import { useStatusMessage } from "./useStatusMessage";

export type PreviewTarget = { path: string; fragment: string; request: number } | null;

export type UseTabSessionDeps = {
  refreshTree: (preferred: string) => Promise<string>;
  notesRef: MutableRefObject<NoteEntry[]>;
  setExpanded: Dispatch<SetStateAction<Set<string>>>;
};

export function useTabSession(deps: UseTabSessionDeps) {
  const { refreshTree, notesRef, setExpanded } = deps;
  const { message, show: showStatus, clear: clearStatus } = useStatusMessage();
  const [tabState, rawTabDispatch] = useReducer(tabSessionReducer, emptyTabSessionState);
  const tabStateRef = useRef<TabSessionState>(emptyTabSessionState);
  const tabDispatch = useCallback((action: TabSessionAction) => {
    tabStateRef.current = tabSessionReducer(tabStateRef.current, action);
    rawTabDispatch(action);
  }, []);
  const tabs = tabState.tabs;
  const activePath = tabState.activePath;
  const activeSession = activeTabSession(tabState);
  const document = activeSession?.document ?? null;
  const draft = activeSession?.draft ?? "";
  const saveStatus: SaveStatus = activeSession?.saveStatus ?? "idle";
  const saveError = activeSession?.saveError ?? "";
  const external = activeSession?.external ?? null;
  const mode: Mode = activeSession?.mode ?? "preview";
  const [pending, setPending] = useState<Pending | null>(null);
  const [previewTarget, setPreviewTarget] = useState<PreviewTarget>(null);
  const [tabsReady, setTabsReady] = useState(false);
  const [activeEditorLine, setActiveEditorLine] = useState(1);
  const [hasEditorSelection, setHasEditorSelection] = useState(false);
  const activeRef = useRef(activePath);
  activeRef.current = activePath;
  const tabsRef = useRef(tabs);
  tabsRef.current = tabs;
  const documentRef = useRef(document);
  documentRef.current = document;
  const draftRef = useRef(draft);
  draftRef.current = draft;
  const modeRef = useRef<Mode>(mode);
  modeRef.current = mode;
  const savingRef = useRef(false);
  const noteLoadRequestRef = useRef(0);
  const editorViewRef = useRef<EditorView | null>(null);
  const previewRequestRef = useRef(0);
  const documentViewRef = useRef<HTMLDivElement | null>(null);

  const editorExtensions = useMemo(
    () => [
      markdownLanguage(),
      syntaxHighlighting(
        HighlightStyle.define([
          { tag: tags.heading, color: "var(--color-heading)", fontWeight: "700" },
          { tag: [tags.link, tags.url], color: "var(--color-link)", textDecoration: "underline" },
          { tag: tags.quote, color: "var(--color-text-secondary)", fontStyle: "italic" },
          { tag: tags.monospace, color: "var(--color-link)" },
          { tag: tags.strong, color: "var(--color-text)", fontWeight: "700" },
          { tag: tags.emphasis, color: "var(--color-text-secondary)", fontStyle: "italic" },
          { tag: [tags.meta, tags.contentSeparator], color: "var(--color-muted)" },
        ])
      ),
      EditorView.lineWrapping,
      EditorView.updateListener.of((update) => {
        if (update.selectionSet || update.docChanged) {
          const selected = !update.state.selection.main.empty;
          update.view.dom.classList.toggle("has-selection", selected);
          setActiveEditorLine(update.state.doc.lineAt(update.state.selection.main.head).number);
          setHasEditorSelection(selected);
        }
      }),
    ],
    []
  );

  const showError = useCallback(
    (error: unknown, prefix = "") => showStatus(`${prefix}${errorMessage(error)}`, "error", 0),
    [showStatus]
  );

  const setDraft = useCallback(
    (value: string) => {
      const path = tabStateRef.current.activePath;
      if (path) tabDispatch({ type: "draft", path, draft: value });
    },
    [tabDispatch]
  );

  const rememberActiveScroll = useCallback(() => {
    const path = activeRef.current;
    if (!path) return;
    const currentMode = modeRef.current;
    const view = documentViewRef.current;
    let top = 0;
    if (currentMode === "edit") {
      const pane = view?.querySelector<HTMLElement>(".editor-pane");
      top = Math.max(editorViewRef.current?.scrollDOM.scrollTop || 0, pane?.scrollTop || 0);
    } else if (currentMode === "split") {
      top = view?.scrollTop || 0;
    } else {
      top = view?.querySelector<HTMLElement>(".markdown-body")?.scrollTop || 0;
    }
    tabDispatch({ type: "scroll", path, mode: currentMode, top });
  }, [tabDispatch]);

  const restoreEditorScroll = useCallback((editor: EditorView, path: string, top: number) => {
    let attempts = 0;
    const apply = () => {
      if (activeRef.current !== path || modeRef.current !== "edit" || editorViewRef.current !== editor) return;
      const pane = documentViewRef.current?.querySelector<HTMLElement>(".editor-pane");
      editor.scrollDOM.scrollTop = top;
      if (pane) pane.scrollTop = top;
      editor.requestMeasure({
        read: () => top,
        write: (value) => {
          if (activeRef.current === path && modeRef.current === "edit" && editorViewRef.current === editor) {
            editor.scrollDOM.scrollTop = value;
            if (pane) pane.scrollTop = value;
          }
        },
      });
      if (++attempts < 3) requestAnimationFrame(apply);
    };
    requestAnimationFrame(apply);
  }, []);

  const handleEditorCreated = useCallback(
    (view: EditorView, path: string) => {
      const selected = !view.state.selection.main.empty;
      editorViewRef.current = view;
      view.dom.classList.toggle("has-selection", selected);
      setActiveEditorLine(view.state.doc.lineAt(view.state.selection.main.head).number);
      setHasEditorSelection(selected);
      if (modeRef.current === "edit") {
        restoreEditorScroll(view, path, tabStateRef.current.tabs.find((tab) => tab.path === path)?.scroll.edit || 0);
      }
    },
    [restoreEditorScroll]
  );

  const setMode = useCallback(
    (nextMode: Mode) => {
      if (nextMode === modeRef.current) return;
      rememberActiveScroll();
      const path = activeRef.current;
      if (path) tabDispatch({ type: "mode", path, mode: nextMode });
    },
    [rememberActiveScroll, tabDispatch]
  );

  const isTabDirty = useCallback(
    (path: string) => isTabSessionDirty(tabStateRef.current.tabs.find((tab) => tab.path === path)),
    []
  );

  const dirtyTabPaths = useCallback(() => dirtyTabSessionPaths(tabStateRef.current), []);

  const cancelPendingLoad = useCallback(() => {
    ++noteLoadRequestRef.current;
  }, []);

  const save = useCallback(
    async (force = false) => {
      const doc = documentRef.current;
      if (!doc || savingRef.current) return false;
      const savingPath = doc.path;
      const submittedDraft = draftRef.current;
      savingRef.current = true;
      tabDispatch({ type: "save-start", path: savingPath });
      try {
        const saved = await api.saveNote(savingPath, submittedDraft, doc.modified_ms, force);
        tabDispatch({ type: "save-success", path: savingPath, note: saved, submittedDraft });
        await refreshTree(savingPath);
        showStatus(t("tab.saved", { path: savingPath }), "success", 3000);
        return true;
      } catch (error) {
        const errorText = errorMessage(error);
        const status = errorKind(error) === "conflict" ? "conflict" : "error";
        tabDispatch({ type: "save-error", path: savingPath, status, message: errorText });
        if (status === "error") showError(error, activeRef.current === savingPath ? "" : t("tab.saveFailed", { path: savingPath }));
        return false;
      } finally {
        savingRef.current = false;
      }
    },
    [refreshTree, showError, showStatus, tabDispatch]
  );

  const saveDirtyTabs = useCallback(async (paths = dirtyTabPaths()) => {
    for (const path of paths) {
      if (path === activeRef.current) {
        if (!(await save())) return false;
        continue;
      }
      const session = tabStateRef.current.tabs.find((tab) => tab.path === path);
      if (!session?.document || !isTabSessionDirty(session)) continue;
      const submittedDraft = session.draft;
      tabDispatch({ type: "save-start", path });
      try {
        const saved = await api.saveNote(path, submittedDraft, session.document.modified_ms);
        tabDispatch({ type: "save-success", path, note: saved, submittedDraft });
      } catch (error) {
        tabDispatch({
          type: "save-error",
          path,
          status: errorKind(error) === "conflict" ? "conflict" : "error",
          message: errorMessage(error),
        });
        showError(error, `Could not save ${path}: `);
        return false;
      }
    }
    await refreshTree(activeRef.current);
    return true;
  }, [dirtyTabPaths, refreshTree, save, showError]);

  const loadNote = useCallback(
    async (path: string, disposition: TabDisposition = "replace", requestedMode: Mode = "preview", useCachedDraft = true) => {
      const request = ++noteLoadRequestRef.current;
      setActiveEditorLine(1);
      setHasEditorSelection(false);
      rememberActiveScroll();
      if (!path) {
        tabDispatch({ type: "activate-empty" });
        clearStatus();
        return;
      }
      const previous = activeRef.current;
      const previousSession = tabStateRef.current.tabs.find((tab) => tab.path === previous);
      const effectiveDisposition = disposition === "replace" && isTabSessionDirty(previousSession) ? "open" : disposition;
      const cached = useCachedDraft ? tabStateRef.current.tabs.find((tab) => tab.path === path) : undefined;
      const note = cached?.document || (await api.readNote(path));
      if (request !== noteLoadRequestRef.current) return;
      tabDispatch({
        type: "load",
        note,
        disposition: effectiveDisposition,
        mode: requestedMode,
        preserveDraft: Boolean(useCachedDraft && cached?.document),
      });
      setExpanded((current) => {
        const next = new Set(current);
        folderAncestors(path).forEach((folder) => next.add(folder));
        return next;
      });
      clearStatus();
    },
    [api, clearStatus, rememberActiveScroll, setExpanded, tabDispatch]
  );

  const restoreTabs = useCallback(
    async (rootPath: string, fallback: string) => {
      const stored = parseStoredTabs(localStorage.getItem(`plainmark-open-tabs:${rootPath}`));
      const restored = restoredOpenTabs(stored, notesRef.current);
      tabDispatch({ type: "restore", tabs: restored, activePath: stored.active });
      const target = restored.some((tab) => tab.path === stored.active) ? stored.active : restored[0]?.path || "";
      if (target) {
        const targetMode = restored.find((tab) => tab.path === target)?.mode || "preview";
        await loadNote(target, "switch", targetMode);
      } else if (fallback && shouldOpenFallbackNote(stored, restored.length)) {
        await loadNote(fallback, "open");
      }
      setTabsReady(true);
    },
    [loadNote, notesRef, tabDispatch]
  );

  const navigate = useCallback(
    (path: string, disposition: TabDisposition, fragment = "") => {
      if (fragment) setPreviewTarget({ path, fragment, request: ++previewRequestRef.current });
      else setPreviewTarget(null);
      if (path === activeRef.current) return;
      loadNote(path, disposition).catch((error) => showError(error));
    },
    [loadNote, setPreviewTarget, showError]
  );

  const requestNavigate = useCallback(
    (path: string, newTab = false, fragment = "") => navigate(path, newTab ? "open" : "replace", fragment),
    [navigate]
  );

  const closeTabNow = useCallback(
    async (path: string) => {
      const wasActive = activeRef.current === path;
      tabDispatch({ type: "close", path });
      if (!wasActive) return;
      ++noteLoadRequestRef.current;
      const nextPath = tabStateRef.current.activePath;
      const next = tabStateRef.current.tabs.find((tab) => tab.path === nextPath);
      if (next?.document) return;
      await loadNote(nextPath, "switch", next?.mode || "preview");
    },
    [loadNote, tabDispatch]
  );

  const requestCloseTab = useCallback(
    (path: string) => {
      if (isTabDirty(path)) {
        void (async () => {
          if (path !== activeRef.current) await loadNote(path, "switch");
          setPending({ kind: "close-tab", path });
        })().catch((error) => showError(error));
        return;
      }
      void closeTabNow(path).catch((error) => showError(error));
    },
    [closeTabNow, isTabDirty, loadNote, showError]
  );

  const closeTabsNow = useCallback(async (paths: string[]) => {
    ++noteLoadRequestRef.current;
    rememberActiveScroll();
    tabDispatch({ type: "close-many", paths });
    const nextPath = tabStateRef.current.activePath;
    const next = tabStateRef.current.tabs.find(tab => tab.path === nextPath);
    if (!next?.document) await loadNote(nextPath, "switch", next?.mode || "preview");
  }, [loadNote, rememberActiveScroll, tabDispatch]);

  const requestCloseTabs = useCallback((othersOnly: boolean) => {
    const paths = tabStateRef.current.tabs
      .filter(tab => !othersOnly || tab.path !== tabStateRef.current.activePath)
      .map(tab => tab.path);
    if (paths.some(isTabDirty)) setPending({ kind: "close-tabs", paths });
    else void closeTabsNow(paths).catch(error => showError(error));
  }, [closeTabsNow, isTabDirty, showError]);

  const cycleTab = useCallback(
    (direction: -1 | 1) => {
      const current = tabsRef.current;
      if (current.length < 2) return;
      const index = current.findIndex((tab) => tab.path === activeRef.current);
      const target = current[(index + direction + current.length) % current.length];
      if (target) navigate(target.path, "switch");
    },
    [navigate]
  );

  const selectTab = useCallback(
    (index: number) => {
      const target = tabsRef.current[index];
      if (target) navigate(target.path, "switch");
    },
    [navigate]
  );

  const requestQuit = useCallback(() => {
    rememberActiveScroll();
    if (dirtyTabPaths().length) setPending({ kind: "quit" });
    else void api.exitApp();
  }, [dirtyTabPaths, rememberActiveScroll]);

  const finishPending = useCallback(
    async (action: "save" | "discard") => {
      const current = pending;
      if (!current) return;
      const targetPaths = current.kind === "close-tabs" ? current.paths : null;
      if (action === "save") {
        const saved = current.kind === "close-tab" ? await save() : await saveDirtyTabs(targetPaths?.filter(isTabDirty));
        if (!saved) return;
        if (targetPaths?.some(isTabDirty)) return;
      } else if (current.kind !== "close-tab") {
        for (const session of tabStateRef.current.tabs) {
          if (targetPaths && !targetPaths.includes(session.path)) continue;
          if (session.document && isTabSessionDirty(session)) {
            tabDispatch({ type: "replace-from-disk", path: session.path, note: session.document });
          }
        }
      }
      setPending(null);
      if (current.kind === "close-tab") await closeTabNow(current.path);
      if (current.kind === "close-tabs") await closeTabsNow(current.paths);
      if (current.kind === "close") await api.hideMainWindow();
      if (current.kind === "quit") await api.exitApp();
    },
    [closeTabNow, closeTabsNow, isTabDirty, pending, save, saveDirtyTabs, tabDispatch]
  );

  useEffect(() => {
    if (saveStatus !== "saved" || !activePath) return;
    const timer = window.setTimeout(() => tabDispatch({ type: "save-idle", path: activePath }), 3000);
    return () => window.clearTimeout(timer);
  }, [activePath, saveStatus, tabDispatch]);

  useLayoutEffect(() => {
    if (!activePath || !document || (previewTarget?.path === activePath && previewTarget.fragment)) return;
    const path = activePath;
    const currentMode = mode;
    const top = tabStateRef.current.tabs.find((tab) => tab.path === path)?.scroll[currentMode] || 0;
    let secondFrame = 0;
    const firstFrame = requestAnimationFrame(() => {
      secondFrame = requestAnimationFrame(() => {
        if (activeRef.current !== path || modeRef.current !== currentMode) return;
        const view = documentViewRef.current;
        if (currentMode === "edit") {
          if (editorViewRef.current) restoreEditorScroll(editorViewRef.current, path, top);
        } else if (currentMode === "split") {
          if (view) view.scrollTop = top;
        } else {
          const preview = view?.querySelector<HTMLElement>(".markdown-body");
          if (preview) preview.scrollTop = top;
        }
      });
    });
    return () => {
      cancelAnimationFrame(firstFrame);
      cancelAnimationFrame(secondFrame);
    };
  }, [activePath, document?.path, mode, previewTarget, restoreEditorScroll]);

  return {
    tabs,
    activePath,
    activeSession,
    document,
    draft,
    saveStatus,
    saveError,
    external,
    mode,
    pending,
    previewTarget,
    tabsReady,
    setTabsReady,
    statusMessage: message,
    showStatus,
    tabDispatch,
    editorExtensions,
    editorViewRef,
    documentViewRef,
    documentRef,
    draftRef,
    activePathRef: activeRef,
    sessionRef: tabStateRef,
    activeEditorLine,
    hasEditorSelection,
    handleEditorCreated,
    save,
    saveDirtyTabs,
    loadNote,
    restoreTabs,
    setDraft,
    setMode,
    rememberActiveScroll,
    restoreEditorScroll,
    navigate,
    requestNavigate,
    closeTabNow,
    requestCloseTab,
    requestCloseTabs,
    cycleTab,
    selectTab,
    dirtyTabPaths,
    isTabDirty,
    cancelPendingLoad,
    setPending,
    finishPending,
    requestQuit,
  };
}
