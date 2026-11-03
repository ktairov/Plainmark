import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { PointerSensor, useSensor, useSensors, type DragEndEvent, type DragStartEvent } from "@dnd-kit/core";
import CodeMirror from "@uiw/react-codemirror";
import { openSearchPanel } from "@codemirror/search";
import { api, events, type MenuActionId } from "./api";
import { MarkdownPreview, PathDialog, SettingsDialog } from "./components";
import { AppSidebar } from "./components/AppSidebar";
import { ConflictModal, CryptoModal, PendingModal, TrashModal } from "./components/AppModals";
import { ContextMenu } from "./components/ContextMenu";
import { ReaderHeader } from "./components/ReaderHeader";
import { TabsBar } from "./components/TabsBar";
import { useAppPersistence } from "./hooks/useAppPersistence";
import { useCryptoDialog } from "./hooks/useCryptoDialog";
import { useDelayedTooltip } from "./hooks/useDelayedTooltip";
import { useFileOperations } from "./hooks/useFileOperations";
import { useMenuActions } from "./hooks/useMenuActions";
import { useRootSwitcher } from "./hooks/useRootSwitcher";
import { useTabSession } from "./hooks/useTabSession";
import { useTreeOperations } from "./hooks/useTreeOperations";
import { useWorkspace } from "./hooks/useWorkspace";
import { useWorkspaceEvents } from "./hooks/useWorkspaceEvents";
import type { DragItem, FormState, MenuTarget, RefreshStatus } from "./appTypes";
import {
  buildNoteTree,
  errorMessage,
  idleStatusText,
  rootFolderName,
  togglePinnedPath,
  type TrashInfo,
} from "./model";
import { fontStack, resolvedColorMode, resolvedTheme, toggledColorMode, type AppSettings } from "./settings";
import { themePage, themeVariables } from "./themes";
import { resolveLanguage, t } from "./i18n";
import { useAppLanguage } from "./i18n/react";
import { menuLabels } from "./i18n/menu";
import { observeSystemTheme } from "./systemTheme";

const sidebarMinWidth = 240;
const sidebarMaxWidth = 640;
const sidebarReservedWidth = 360;

export function App() {
  const { root, setRoot, notes, folders, notesRef, foldersRef, refreshTree, cancelPendingRefresh, clearWorkspace } = useWorkspace();
  const {
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
  } = useAppPersistence();

  const tabSession = useTabSession({ refreshTree, notesRef, setExpanded });
  const {
    tabs,
    activePath,
    document: activeDocument,
    draft,
    saveStatus,
    saveError,
    external,
    mode,
    pending,
    previewTarget,
    tabsReady,
    setTabsReady,
    statusMessage,
    showStatus,
    editorExtensions,
    editorViewRef,
    documentViewRef,
    documentRef,
    draftRef,
    activePathRef,
    sessionRef,
    activeEditorLine,
    hasEditorSelection,
    handleEditorCreated,
    save,
    loadNote,
    restoreTabs,
    setDraft,
    setMode,
    rememberActiveScroll,
    navigate,
    requestNavigate,
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
    tabDispatch,
  } = tabSession;

  const [activeFolder, setActiveFolder] = useState("");
  const [query, setQuery] = useState("");
  const [menu, setMenu] = useState<MenuTarget | null>(null);
  const [form, setForm] = useState<FormState | null>(null);
  const [trashInfo, setTrashInfo] = useState<TrashInfo | null>(null);
  const [settingsPreview, setSettingsPreview] = useState<AppSettings | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [systemDark, setSystemDark] = useState(() => window.matchMedia("(prefers-color-scheme: dark)").matches);
  const [colorModeOverride, setColorModeOverride] = useState<"light" | "dark" | null>(null);
  const [refreshStatus, setRefreshStatus] = useState<RefreshStatus>("idle");
  const [activeDrag, setActiveDrag] = useState<DragItem | null>(null);
  const [menuEpoch, setMenuEpoch] = useState(0);
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const menuActionRef = useRef<(id: MenuActionId) => void>(() => {});
  const languageSyncRef = useRef<Promise<void>>(Promise.resolve());
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));
  const { tooltip, hide: hideTooltip, onPointerOver: handleTooltipPointerOver, onPointerOut: handleTooltipPointerOut } = useDelayedTooltip();

  const visibleSettings = settingsPreview || settings;
  const targetLanguage = resolveLanguage(visibleSettings.language, [navigator.language, ...navigator.languages]);
  const language = useAppLanguage(targetLanguage);
  const configuredColorMode = resolvedColorMode(visibleSettings.colorMode, systemDark);
  const colorMode = settingsPreview ? configuredColorMode : colorModeOverride || configuredColorMode;
  const activeTheme = resolvedTheme(visibleSettings, colorMode);
  const appStyle = {
    ...themeVariables(activeTheme, colorMode),
    "--sidebar-width": `${width}px`,
    "--tree-font-size": `${visibleSettings.treeFontSize}px`,
    "--content-font-size": `${visibleSettings.contentFontSize}px`,
    "--preview-width": `${visibleSettings.previewWidth}px`,
    "--font-ui": fontStack(visibleSettings.uiFont),
    "--font-reading": fontStack(visibleSettings.readingFont),
  } as React.CSSProperties;

  const showError = useCallback(
    (error: unknown, prefix = "") => showStatus(`${prefix}${errorMessage(error)}`, "error", 0),
    [showStatus]
  );

  const updatePinnedPaths = useCallback(
    (update: (paths: string[]) => string[]) => setPinnedByRoot((current) => ({ ...current, [root]: update(current[root] || []) })),
    [root, setPinnedByRoot]
  );

  const { treeChanged, moveByDrop, rememberLocalTreeMutation, isLocalTreeAddition } = useTreeOperations({
    activePathRef,
    refreshTree,
    loadNote,
    tabDispatch,
    setExpanded,
    setActiveFolder,
    updatePinnedPaths,
    showStatus,
    showError,
  });

  const switchRoot = useRootSwitcher({
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
  });

  const {
    cryptoDialog,
    cryptoPassword,
    setCryptoPassword,
    cryptoConfirm,
    setCryptoConfirm,
    cryptoError,
    cryptoBusy,
    unlocked,
    closeCrypto,
    requestEncryptSelection,
    requestDecrypt,
    lockEncrypted,
    submitCrypto,
  } = useCryptoDialog({ editorViewRef, mode, activeDocument, setDraft, showStatus });

  useWorkspaceEvents({
    refreshTree,
    loadNote,
    isLocalTreeAddition,
    notesRef,
    foldersRef,
    activePathRef,
    documentRef,
    draftRef,
    tabDispatch,
    setActiveFolder,
    showStatus,
    showError,
  });

  const { submitForm, requestDelete, confirmDelete, saveCopy, copyPath, chooseRoot, chooseFile, manualRefresh, createInActiveFolder } = useFileOperations({
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
  });

  const togglePinned = useCallback(
    (path: string) => {
      const pinned = (pinnedByRoot[root] || []).includes(path);
      updatePinnedPaths((paths) => togglePinnedPath(paths, path));
      setMenu(null);
      showStatus(t(pinned ? "app.toast.unpinned" : "app.toast.pinned", { path }), "success");
    },
    [pinnedByRoot, root, showStatus, updatePinnedPaths]
  );

  const findInNote = useCallback(() => {
    if (mode === "preview") window.dispatchEvent(new Event("plainmark-find"));
    else if (editorViewRef.current) openSearchPanel(editorViewRef.current);
  }, [mode, editorViewRef]);

  const searchNotes = useCallback(() => {
    setSidebarVisible(true);
    requestAnimationFrame(() => searchInputRef.current?.select());
  }, [setSidebarVisible]);

  const copyRelativePath = useCallback(async (path: string) => {
    try {
      await api.copyText(path);
      showStatus(t("app.toast.copiedRelativePath", { path }), "success");
    } catch (error) { showError(error); }
  }, [showStatus, showError]);

  const revealNote = useCallback(async (path: string) => {
    try { await api.revealNote(path); }
    catch (error) { showError(error); }
  }, [showError]);

  const handleMenuAction = useMenuActions({
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
    togglePin: togglePinned,
    copyRelativePath,
    revealNote,
    requestQuit,
    requestEncryptSelection,
    showStatus,
  });

  useEffect(() => {
    const root = document.documentElement;
    root.dataset.colorMode = colorMode;
    root.dataset.theme = activeTheme;
    root.style.colorScheme = colorMode;
    root.style.backgroundColor = themePage(activeTheme, colorMode);
  }, [activeTheme, colorMode]);

  useEffect(() => {
    // The window starts hidden so its first visible frame already has the
    // resolved theme; show it once the shell has been committed.
    api.revealMainWindow().catch(() => {});
  }, []);

  useEffect(() => {
    (async () => {
      try {
        const config = await api.getConfig();
        setRoot(config.root);
        const first = await refreshTree("");
        await restoreTabs(config.root, first);
      } catch (error) {
        showError(error);
      }
    })();
  }, [refreshTree, restoreTabs, setRoot, showError]);

  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    return observeSystemTheme(
      media,
      {
        current: api.windowTheme,
        subscribe: events.onWindowThemeChanged,
      },
      dark => {
        setSystemDark(dark);
        if (settings.colorMode === "system") setColorModeOverride(null);
      },
    );
  }, [settings.colorMode]);

  useEffect(() => {
    if (root && tabsReady) persistTabs(root, tabs, activePath);
  }, [root, tabs, activePath, tabsReady, persistTabs]);

  useEffect(() => {
    const enabled = mode !== "preview" && hasEditorSelection && Boolean(activeDocument) && !pending && !form && !trashInfo && !cryptoDialog && !settingsOpen && saveStatus !== "saving";
    api.setEncryptSelectionEnabled(enabled).catch((error) => showError(error, t("app.error.encryptMenu")));
  }, [mode, hasEditorSelection, activeDocument?.path, pending, form, trashInfo, cryptoDialog, settingsOpen, saveStatus, menuEpoch, showError]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && !event.altKey && event.code === "Comma") {
        event.preventDefault();
        setSettingsOpen(true);
        return;
      }
      const digit = /^Digit([1-9])$/.exec(event.code);
      if ((event.metaKey || event.ctrlKey) && !event.altKey && digit) {
        event.preventDefault();
        selectTab(Number(digit[1]) - 1);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [selectTab]);

  useEffect(() => {
    const unlisten = events.onWindowCloseRequested(() => {
      rememberActiveScroll();
      if (dirtyTabPaths().length) setPending({ kind: "close" });
      else void api.hideMainWindow();
    });
    return () => {
      unlisten.then((fn) => fn());
    };
  }, [dirtyTabPaths, rememberActiveScroll, setPending]);

  useEffect(() => {
    const unlisten = events.onAppExitRequested(requestQuit);
    return () => {
      unlisten.then((fn) => fn());
    };
  }, [requestQuit]);

  useEffect(() => {
    const before = (event: BeforeUnloadEvent) => {
      rememberActiveScroll();
      if (dirtyTabPaths().length) {
        event.preventDefault();
        event.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", before);
    return () => window.removeEventListener("beforeunload", before);
  }, [dirtyTabPaths, rememberActiveScroll]);

  const pinnedPaths = pinnedByRoot[root] || [];
  const pinnedSet = useMemo(() => new Set(pinnedPaths), [pinnedPaths]);
  const filtered = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return notes;
    return notes.filter((note) =>
      [note.path, note.title, note.folder, ...note.headings.map((heading) => heading.text)].join(" ").toLowerCase().includes(normalized)
    );
  }, [notes, query]);
  const pinnedNotes = useMemo(() => filtered.filter((note) => pinnedSet.has(note.path)).sort((left, right) => left.title.localeCompare(right.title)), [filtered, pinnedSet]);
  const regularNotes = useMemo(() => filtered.filter((note) => !pinnedSet.has(note.path)), [filtered, pinnedSet]);
  const visibleFolders = useMemo(
    () => (query.trim() ? folders.filter((folder) => regularNotes.some((note) => note.path.startsWith(`${folder.path}/`))) : folders),
    [folders, regularNotes, query]
  );
  const tree = useMemo(() => buildNoteTree(regularNotes, visibleFolders), [regularNotes, visibleFolders]);

  const showContextMenu = useCallback((event: React.MouseEvent, path: string, kind: "folder" | "note") => {
    event.preventDefault();
    if (kind === "folder") setActiveFolder(path);
    setMenu({ path, kind, x: event.clientX, y: event.clientY });
  }, []);

  const toggleFolder = useCallback(
    (path: string) => {
      if (query.trim()) return;
      setExpanded((current) => {
        const next = new Set(current);
        if (next.has(path)) next.delete(path);
        else next.add(path);
        return next;
      });
    },
    [query, setExpanded]
  );

  const clampWidth = useCallback(
    (value: number) => Math.min(Math.max(value, sidebarMinWidth), Math.max(sidebarMinWidth, Math.min(sidebarMaxWidth, window.innerWidth - sidebarReservedWidth))),
    []
  );

  const handleDragStart = useCallback((event: DragStartEvent) => setActiveDrag(event.active.data.current as DragItem), []);
  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const item = event.active.data.current as DragItem | undefined;
      const destination = event.over?.data.current?.path;
      setActiveDrag(null);
      if (item && typeof destination === "string" && item.path !== destination) void moveByDrop(item.path, destination);
    },
    [moveByDrop]
  );

  useEffect(() => {
    menuActionRef.current = handleMenuAction;
  }, [handleMenuAction]);

  useEffect(() => {
    const unlisten = events.onMenuAction((action) => menuActionRef.current(action));
    return () => {
      unlisten.then((fn) => fn());
    };
  }, []);

  const statusText = statusMessage?.text || idleStatusText(Boolean(root && tabsReady), activePath, notes.length);
  const statusTone = statusMessage?.tone || "info";
  const pendingDirtyPaths = pending ? dirtyTabPaths().filter(path =>
    pending.kind === "close-tabs" ? pending.paths.includes(path) : pending.kind === "close-tab" ? path === pending.path : true
  ) : [];

  const menuBusy = Boolean(pending || form || trashInfo || cryptoDialog || settingsOpen || saveStatus === "saving" || !tabsReady);
  const activeDirty = Boolean(activeDocument && draft !== activeDocument.markdown);
  const activePinned = pinnedSet.has(activePath);

  useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);

  useEffect(() => {
    let cancelled = false;
    const update = () => settingsPreview
      ? api.previewAppLanguage(targetLanguage, menuLabels(targetLanguage))
      : api.setAppLanguage(targetLanguage, menuLabels(targetLanguage));
    const request = languageSyncRef.current.catch(() => undefined).then(update);
    languageSyncRef.current = request;
    request
      .then(() => { if (!cancelled) setMenuEpoch(value => value + 1); })
      .catch(error => showError(error, t("app.error.menuState")));
    return () => { cancelled = true; };
  }, [targetLanguage, Boolean(settingsPreview), showError]);

  useEffect(() => {
    api.syncMenuState({ hasNote: Boolean(activeDocument), dirty: activeDirty, busy: menuBusy, tabCount: tabs.length, mode, sidebarVisible, pinned: activePinned })
      .catch(error => showError(error, t("app.error.menuState")));
  }, [activeDocument?.path, activeDirty, menuBusy, tabs.length, mode, sidebarVisible, activePinned, menuEpoch, showError]);

  useEffect(() => {
    api.syncRecentFolders(recentRoots).catch(error => showError(error, t("app.error.recentFolders")));
  }, [recentRoots, menuEpoch, showError]);

  useEffect(() => {
    api.setColorModeMenuLabel(colorMode === "dark").catch((error) => showError(error, t("app.error.colorModeMenu")));
  }, [colorMode, menuEpoch, showError]);

  return (
    <main
      className={`app-shell ${sidebarVisible ? "" : "sidebar-hidden"}`}
      data-color-mode={colorMode}
      data-theme={activeTheme}
      style={appStyle}
      onClick={() => menu && setMenu(null)}
      onPointerOver={handleTooltipPointerOver}
      onPointerOut={handleTooltipPointerOut}
      onPointerDownCapture={hideTooltip}
      onWheelCapture={hideTooltip}
      onPointerLeave={hideTooltip}
    >
      <ReaderHeader
        sidebarVisible={sidebarVisible}
        mode={mode}
        colorMode={colorMode}
        saveStatus={saveStatus}
        saveError={saveError}
        rootLabel={rootFolderName(root)}
        chooseRoot={chooseRoot}
        create={createInActiveFolder}
        toggleSidebar={() => setSidebarVisible((value) => !value)}
        setMode={setMode}
        toggleColorMode={() => setColorModeOverride(toggledColorMode(colorMode))}
        recentRoots={recentRoots}
      >
        <TabsBar
          tabs={tabs}
          activePath={activePath}
          isDirty={isTabDirty}
          navigate={(path) => navigate(path, "switch")}
          close={requestCloseTab}
          reorder={(from, to) => tabDispatch({ type: "reorder", path: from, overPath: to })}
        />
      </ReaderHeader>

      {sidebarVisible && (
        <AppSidebar
          rootLabel={rootFolderName(root)}
          query={query}
          tree={tree}
          pinnedNotes={pinnedNotes}
          activePath={activePath}
          activeFolder={activeFolder}
          expanded={expanded}
          activeDrag={activeDrag}
          searchInputRef={searchInputRef}
          sensors={sensors}
          setQuery={setQuery}
          selectFolder={setActiveFolder}
          selectNote={requestNavigate}
          toggleFolder={toggleFolder}
          showMenu={showContextMenu}
          startDrag={handleDragStart}
          endDrag={handleDragEnd}
          cancelDrag={() => setActiveDrag(null)}
          showRootMenu={(event) => {
            if (event.target === event.currentTarget) {
              event.preventDefault();
              setActiveFolder("");
              setMenu({ path: "", kind: "root", x: event.clientX, y: event.clientY });
            }
          }}
        />
      )}

      {sidebarVisible && (
        <div
          className="sidebar-resizer"
          role="separator"
          tabIndex={0}
          onPointerDown={(event) => event.currentTarget.setPointerCapture(event.pointerId)}
          onPointerMove={(event) => event.currentTarget.hasPointerCapture(event.pointerId) && setWidth(clampWidth(event.clientX))}
          onPointerUp={(event) => event.currentTarget.releasePointerCapture(event.pointerId)}
          onKeyDown={(event) => {
            if (event.key === "ArrowLeft") setWidth((value) => clampWidth(value - 20));
            if (event.key === "ArrowRight") setWidth((value) => clampWidth(value + 20));
          }}
        />
      )}

      <section className="reader">
        <div ref={documentViewRef} className={`document-view mode-${mode}`}>
          {activeDocument ? (
            <>
              {mode !== "preview" && (
                <div className="editor-pane">
                  <CodeMirror
                    key={activeDocument.path}
                    value={draft}
                    height={mode === "split" ? "auto" : "100%"}
                    minHeight="300px"
                    theme={colorMode}
                    extensions={editorExtensions}
                    onCreateEditor={(view) => handleEditorCreated(view, activeDocument.path)}
                    onChange={setDraft}
                    basicSetup={{ lineNumbers: true, highlightActiveLine: true, highlightSelectionMatches: false, history: true, foldGutter: true }}
                  />
                </div>
              )}
              {mode !== "edit" && (
                <MarkdownPreview
                  note={activeDocument}
                  markdown={draft}
                  notes={notes}
                  unlocked={unlocked}
                  activeLine={mode === "split" ? activeEditorLine : 0}
                  suppressActiveHighlight={mode === "split" && hasEditorSelection}
                  searchEnabled={mode === "preview"}
                  target={previewTarget}
                  navigate={requestNavigate}
                  report={(message) => showStatus(message, "error", 0)}
                  unlock={requestDecrypt}
                  lock={lockEncrypted}
                />
              )}
            </>
          ) : (
            <div className="empty-state">{notes.length ? t("app.empty.noNote") : t("app.empty.noNotes")}</div>
          )}
        </div>
      </section>

      <footer className={`app-status ${statusTone}`} role="status" aria-live="polite" title={statusText}>{statusText}</footer>

      {menu && (
        <ContextMenu
          menu={menu}
          pinned={pinnedSet.has(menu.path)}
          onNewNote={() => {
            setForm({ operation: "note", path: menu.path, kind: menu.kind });
            setMenu(null);
          }}
          onNewFolder={() => {
            setForm({ operation: "folder", path: menu.path, kind: menu.kind });
            setMenu(null);
          }}
          onCopyPath={() => void copyPath(menu.path)}
          onOpenInNewTab={() => {
            requestNavigate(menu.path, true);
            setMenu(null);
          }}
          onTogglePin={() => togglePinned(menu.path)}
          onRename={() => {
            setForm({ operation: "rename", path: menu.path, kind: menu.kind });
            setMenu(null);
          }}
          onMove={() => {
            setForm({ operation: "move", path: menu.path, kind: menu.kind });
            setMenu(null);
          }}
          onDelete={() => {
            void requestDelete(menu.path);
            setMenu(null);
          }}
        />
      )}

      {cryptoDialog && (
        <CryptoModal
          dialog={cryptoDialog}
          password={cryptoPassword}
          confirm={cryptoConfirm}
          error={cryptoError}
          busy={cryptoBusy}
          onPassword={setCryptoPassword}
          onConfirm={setCryptoConfirm}
          onSubmit={submitCrypto}
          onClose={closeCrypto}
        />
      )}

      {form && <PathDialog state={form} folders={folders} rootLabel={rootFolderName(root)} onSubmit={submitForm} onClose={() => setForm(null)} />}

      {pending && (
        <PendingModal
          pending={pending}
          dirtyPaths={pendingDirtyPaths}
          onStay={() => setPending(null)}
          onDiscard={() => void finishPending("discard")}
          onSave={() => void finishPending("save")}
        />
      )}

      {(external || saveStatus === "conflict") && (
        <ConflictModal
          onLoadFromDisk={() => {
            if (external) tabDispatch({ type: "replace-from-disk", path: activePath, note: external });
            else void loadNote(activePath, "switch", "preview", false).catch(showError);
          }}
          onSaveCopy={() => void saveCopy()}
          onOverwrite={() => void save(true)}
        />
      )}

      {trashInfo && <TrashModal info={trashInfo} onCancel={() => setTrashInfo(null)} onConfirm={() => void confirmDelete()} />}

      {settingsOpen && (
        <SettingsDialog
          settings={settings}
          preview={setSettingsPreview}
          apply={(next) => {
            setSettings(next);
            setSettingsPreview(null);
            setColorModeOverride(null);
            setSettingsOpen(false);
          }}
          close={() => {
            setSettingsPreview(null);
            setSettingsOpen(false);
          }}
        />
      )}

      {tooltip && <div className="app-tooltip" role="tooltip" style={{ left: tooltip.x, top: tooltip.y }}>{tooltip.text}</div>}
    </main>
  );
}
