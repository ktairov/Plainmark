import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useDraggable, useDroppable } from "@dnd-kit/core";
import ReactMarkdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkFrontmatter from "remark-frontmatter";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import { openUrl } from "@tauri-apps/plugin-opener";
import { ChevronDown, ChevronRight, ChevronUp, FileText, Folder, FolderOpen, Languages, LockKeyhole, Palette, Pin, Search, Type, UnlockKeyhole, X } from "lucide-react";
import type { FormState, DragItem } from "./appTypes";
import { baseName, markdownHeadingId, markdownHrefFragment, markActivePreviewBlock, parentFolder, previewAnchorScrollPlan, resolveLocalNotePath, type FolderEntry, type NoteContent, type NoteEntry, type OpenNoteTab, type SourceNode, type TreeNode } from "./model";
import { defaultAppSettings, fontOptions, themeOptions, type AppSettings, type ColorMode, type FontId, type ThemeId } from "./settings";
import { typographicDashes } from "./typography";
import { normalizePreviewMath } from "./preview/math";
import { remarkHidePreviewMetadata } from "./preview/metadata";
import { remarkTableBreaks } from "./preview/tables";
import { previewTextMatches } from "./previewSearch";
import { localeRegistry, supportedLanguages, t, type LanguageSetting, type MessageKey } from "./i18n";

type MarkdownNode = { type?: string; value?: string; children?: MarkdownNode[] };
type TreeProps = { node: TreeNode; depth?: number; active: string; activeFolder: string; expanded: Set<string>; forced: boolean; select: (path: string, newTab?: boolean) => void; selectFolder: (path: string) => void; toggle: (path: string) => void; menu: (event: React.MouseEvent, path: string, kind: "folder" | "note") => void };

const treeFontSizes = Array.from({ length: 9 }, (_, index) => index + 13);
const contentFontSizes = Array.from({ length: 14 }, (_, index) => index + 13);

function remarkTypographicDashes() {
  return (tree: MarkdownNode) => {
    const walk = (node: MarkdownNode) => {
      if (node.type === "text" && node.value) node.value = typographicDashes(node.value);
      node.children?.forEach(walk);
    };
    walk(tree);
  };
}

function rehypeActivePreviewLine(activeLine: number) {
  return () => (tree: SourceNode) => { markActivePreviewBlock(tree, activeLine); };
}

function rehypeHeadingIds() {
  return (tree: SourceNode) => {
    const seen = new Map<string, number>();
    const text = (node: SourceNode): string => node.value || node.children?.map(text).join("") || "";
    const walk = (node: SourceNode) => {
      if (node.type === "element" && /^h[1-6]$/.test(node.tagName || "")) {
        const base = markdownHeadingId(text(node)) || "section";
        const duplicate = seen.get(base) || 0;
        seen.set(base, duplicate + 1);
        node.properties = { ...node.properties, id: duplicate ? `${base}-${duplicate}` : base };
      }
      node.children?.forEach(walk);
    };
    walk(tree);
  };
}

export function Modal({ title, children, onClose }: { title: string; children: React.ReactNode; onClose?: () => void }) {
  return <div className="modal-backdrop" role="presentation" onMouseDown={event => { if (event.target === event.currentTarget) onClose?.(); }}><section className="modal" role="dialog" aria-modal="true" aria-label={title}><h3>{title}</h3>{children}</section></div>;
}

export function SettingsDialog({ settings, preview, apply, close }: { settings: AppSettings; preview: (settings: AppSettings) => void; apply: (settings: AppSettings) => void; close: () => void }) {
  const [section, setSection] = useState<"appearance" | "theme" | "language">("appearance");
  const [draft, setDraft] = useState(settings);
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") close(); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [close]);
  const change = <Key extends keyof AppSettings>(key: Key, value: AppSettings[Key]) => {
    const next = { ...draft, [key]: value };
    setDraft(next);
    preview(next);
  };
  const changeTheme = (key: "dayTheme" | "nightTheme", value: ThemeId, colorMode: "light" | "dark") => {
    const next = { ...draft, [key]: value };
    setDraft(next);
    preview({ ...next, colorMode });
  };
  const reset = () => { setDraft(defaultAppSettings); preview(defaultAppSettings); };
  return (
    <div className="modal-backdrop settings-backdrop" role="presentation" onMouseDown={event => { if (event.target === event.currentTarget) close(); }}>
      <section className="settings-dialog" role="dialog" aria-modal="true" aria-label={t("settings.aria.dialog")}>
        <header className="settings-header"><h2>{t("settings.title")}</h2></header>
        <div className="settings-layout">
          <aside className="settings-sidebar">
            <nav className="settings-nav" aria-label={t("settings.aria.sections")}>
              <button className={section === "appearance" ? "selected" : ""} onClick={() => setSection("appearance")}><Type size={17} />{t("settings.section.appearance")}</button>
              <button className={section === "theme" ? "selected" : ""} onClick={() => setSection("theme")}><Palette size={17} />{t("settings.section.theme")}</button>
              <button className={section === "language" ? "selected" : ""} onClick={() => setSection("language")}><Languages size={17} />{t("settings.section.language")}</button>
            </nav>
          </aside>
          <div className="settings-panel">
            {section === "appearance" ? <>
              <div className="settings-title"><h3>{t("settings.section.appearance")}</h3><p>{t("settings.appearance.description")}</p></div>
              <div className="setting-group">
                <span className="setting-label">{t("settings.colorMode")}</span>
                <div className="settings-segmented" role="group" aria-label={t("settings.aria.colorMode")}>
                  {([["system", "settings.colorMode.system"], ["light", "settings.colorMode.light"], ["dark", "settings.colorMode.dark"]] as Array<[ColorMode, MessageKey]>).map(([value, label]) => (
                    <button key={value} className={draft.colorMode === value ? "selected" : ""} aria-pressed={draft.colorMode === value} onClick={() => change("colorMode", value)}>{t(label)}</button>
                  ))}
                </div>
              </div>
              <div className="setting-row">
                <span><strong>{t("settings.uiFont")}</strong><small>{t("settings.uiFontHint")}</small></span>
                <div className="setting-controls">
                  <select aria-label={t("settings.aria.interfaceFont")} value={draft.uiFont} onChange={event => change("uiFont", event.target.value as FontId)}>
                    {fontOptions.map(option => <option key={option.id} value={option.id}>{t(`font.${option.id}` as MessageKey)}</option>)}
                  </select>
                  <select className="font-size-select" aria-label={t("settings.aria.interfaceSize")} value={draft.treeFontSize} onChange={event => change("treeFontSize", Number(event.target.value))}>
                    {treeFontSizes.map(size => <option key={size} value={size}>{size} px</option>)}
                  </select>
                </div>
              </div>
              <div className="setting-row">
                <span><strong>{t("settings.readingFont")}</strong><small>{t("settings.readingFontHint")}</small></span>
                <div className="setting-controls">
                  <select aria-label={t("settings.aria.readingFont")} value={draft.readingFont} onChange={event => change("readingFont", event.target.value as FontId)}>
                    {fontOptions.map(option => <option key={option.id} value={option.id}>{t(`font.${option.id}` as MessageKey)}</option>)}
                  </select>
                  <select className="font-size-select" aria-label={t("settings.aria.documentSize")} value={draft.contentFontSize} onChange={event => change("contentFontSize", Number(event.target.value))}>
                    {contentFontSizes.map(size => <option key={size} value={size}>{size} px</option>)}
                  </select>
                </div>
              </div>
              <label className="setting-slider"><span><strong>{t("settings.previewWidth")}</strong><output>{draft.previewWidth}px</output></span><input type="range" min="560" max="1400" step="20" value={draft.previewWidth} onChange={event => change("previewWidth", Number(event.target.value))} /></label>
            </> : section === "theme" ? <>
              <div className="settings-title"><h3>{t("settings.section.theme")}</h3><p>{t("settings.theme.description")}</p></div>
              <section className="theme-section" aria-labelledby="day-theme-heading">
                <div className="theme-section-heading"><h4 id="day-theme-heading">{t("settings.theme.day")}</h4><p>{t("settings.theme.dayHint")}</p></div>
                <div className="theme-grid">
                  {themeOptions.map(option => (
                    <button key={option.id} className={`theme-card ${draft.dayTheme === option.id ? "selected" : ""}`} aria-pressed={draft.dayTheme === option.id} onClick={() => changeTheme("dayTheme", option.id, "light")}>
                      <span className="theme-swatches">{option.day.map(color => <i key={color} style={{ background: color }} />)}</span>
                      <strong>{option.label}</strong>
                      {draft.dayTheme === option.id && <span className="theme-check">✓</span>}
                    </button>
                  ))}
                </div>
              </section>
              <section className="theme-section" aria-labelledby="night-theme-heading">
                <div className="theme-section-heading"><h4 id="night-theme-heading">{t("settings.theme.night")}</h4><p>{t("settings.theme.nightHint")}</p></div>
                <div className="theme-grid">
                  {themeOptions.map(option => (
                    <button key={option.id} className={`theme-card ${draft.nightTheme === option.id ? "selected" : ""}`} aria-pressed={draft.nightTheme === option.id} onClick={() => changeTheme("nightTheme", option.id, "dark")}>
                      <span className="theme-swatches">{option.night.map(color => <i key={color} style={{ background: color }} />)}</span>
                      <strong>{option.label}</strong>
                      {draft.nightTheme === option.id && <span className="theme-check">✓</span>}
                    </button>
                  ))}
                </div>
              </section>
            </> : <>
              <div className="settings-title"><h3>{t("settings.section.language")}</h3><p>{t("settings.language.description")}</p></div>
              <div className="setting-group">
                <span className="setting-label">{t("settings.section.language")}</span>
                <select aria-label={t("settings.section.language")} value={draft.language} onChange={event => change("language", event.target.value as LanguageSetting)}>
                  <option value="system">{t("settings.language.system")}</option>
                  {supportedLanguages.map(language => (
                    <option key={language} value={language}>{localeRegistry[language].nativeName}</option>
                  ))}
                </select>
                <p className="setting-hint">{t("settings.language.systemHint")}</p>
              </div>
            </>}
          </div>
        </div>
        <footer className="settings-footer">
          <button type="button" onClick={reset}>{t("common.resetDefaults")}</button>
          <div className="settings-footer-actions">
            <button className="primary" type="button" onClick={() => apply(draft)}>{t("common.apply")}</button>
            <button type="button" onClick={close}>{t("common.cancel")}</button>
          </div>
        </footer>
      </section>
    </div>
  );
}

type PreviewTarget = { path: string; fragment: string; request: number } | null;
type PreviewTextSegment = { node: Text; start: number; end: number };
type HighlightRegistry = { set: (name: string, highlight: unknown) => void; delete: (name: string) => void };

const previewSearchHighlight = "plainmark-preview-search";
const previewSearchCurrentHighlight = "plainmark-preview-search-current";

function previewHighlightApi() {
  const css = CSS as typeof CSS & { highlights?: HighlightRegistry };
  const HighlightConstructor = (globalThis as typeof globalThis & { Highlight?: new (...ranges: Range[]) => unknown }).Highlight;
  return css.highlights && HighlightConstructor ? { registry: css.highlights, HighlightConstructor } : null;
}

function clearPreviewHighlights() {
  const api = previewHighlightApi();
  api?.registry.delete(previewSearchHighlight);
  api?.registry.delete(previewSearchCurrentHighlight);
}

function previewSearchRanges(content: HTMLElement, query: string) {
  if (!query) return [];
  const segments: PreviewTextSegment[] = [];
  let text = "";
  let previousBlock: Element | null = null;
  const walker = document.createTreeWalker(content, NodeFilter.SHOW_TEXT);
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    const textNode = node as Text;
    const parent = textNode.parentElement;
    if (!textNode.data || !parent || parent.closest(".katex,[aria-hidden='true'],script,style")) continue;
    const block = parent.closest("p,h1,h2,h3,h4,h5,h6,li,blockquote,pre,td,th,div");
    if (text && block !== previousBlock) text += "\n";
    const start = text.length;
    text += textNode.data;
    segments.push({ node: textNode, start, end: text.length });
    previousBlock = block;
  }
  const ranges: Range[] = [];
  for (const { start, end } of previewTextMatches(text, query)) {
    const first = segments.find(segment => start >= segment.start && start < segment.end);
    const last = segments.find(segment => end > segment.start && end <= segment.end);
    if (!first || !last) continue;
    const range = document.createRange();
    range.setStart(first.node, start - first.start);
    range.setEnd(last.node, end - last.start);
    ranges.push(range);
  }
  return ranges;
}

function previewScroller(preview: HTMLElement) {
  return preview.closest<HTMLElement>(".mode-split") || preview;
}

function scrollPreviewToFragment(preview: HTMLElement, fragment: string, report: (status: string) => void) {
  const target = preview.querySelector<HTMLElement>(`#${CSS.escape(fragment)}`);
  if (!target) { report(t("preview.headingNotFound", { fragment })); return; }
  preview.style.setProperty("--anchor-scroll-extra", "0px");
  requestAnimationFrame(() => {
    const scroller = previewScroller(preview);
    const plan = previewAnchorScrollPlan(scroller.scrollTop, target.getBoundingClientRect().top, scroller.getBoundingClientRect().top, scroller.scrollHeight, scroller.clientHeight);
    if (plan.extra) preview.style.setProperty("--anchor-scroll-extra", `${Math.ceil(plan.extra)}px`);
    requestAnimationFrame(() => scroller.scrollTo({ top: plan.top, behavior: "smooth" }));
  });
}

export function MarkdownPreview({ note, markdown, notes, unlocked, activeLine, suppressActiveHighlight, searchEnabled, target, navigate, report, unlock, lock }: { note: NoteContent; markdown: string; notes: NoteEntry[]; unlocked: Record<string, string>; activeLine: number; suppressActiveHighlight: boolean; searchEnabled: boolean; target: PreviewTarget; navigate: (path: string, newTab?: boolean, fragment?: string) => void; report: (status: string) => void; unlock: (payload: string) => void; lock: (payload: string) => void }) {
  const previewRef = useRef<HTMLElement|null>(null);
  const previewSearchInputRef = useRef<HTMLInputElement|null>(null);
  const previewSearchRangesRef = useRef<Range[]>([]);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchIndex, setSearchIndex] = useState(-1);
  const [searchCount, setSearchCount] = useState(0);
  const previewMarkdown = useMemo(() => normalizePreviewMath(markdown), [markdown]);
  const activeLinePlugin = useMemo(() => rehypeActivePreviewLine(activeLine), [activeLine]);
  const closeSearch = useCallback(() => { setSearchOpen(false); clearPreviewHighlights(); }, []);
  const moveSearch = useCallback((direction: -1 | 1) => {
    if (!searchCount) return;
    setSearchIndex(current => (current + direction + searchCount) % searchCount);
  }, [searchCount]);
  useEffect(() => {
    if (!searchEnabled) { closeSearch(); return; }
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && !event.altKey && !event.shiftKey && !event.defaultPrevented && event.key.toLowerCase() === "f") {
        event.preventDefault();
        setSearchOpen(true);
        requestAnimationFrame(() => previewSearchInputRef.current?.select());
      } else if (event.key === "Escape" && searchOpen) {
        event.preventDefault();
        closeSearch();
      }
    };
    const onFind = () => {
      setSearchOpen(true);
      requestAnimationFrame(() => previewSearchInputRef.current?.select());
    };
    window.addEventListener("plainmark-find", onFind);
    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("plainmark-find", onFind);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [closeSearch, searchEnabled, searchOpen]);
  useLayoutEffect(() => {
    clearPreviewHighlights();
    const content = previewRef.current?.querySelector<HTMLElement>(".markdown-content");
    const ranges = searchOpen && content ? previewSearchRanges(content, searchQuery) : [];
    previewSearchRangesRef.current = ranges;
    setSearchCount(ranges.length);
    setSearchIndex(current => ranges.length ? Math.min(Math.max(current, 0), ranges.length - 1) : -1);
    const api = previewHighlightApi();
    if (api && ranges.length) api.registry.set(previewSearchHighlight, new api.HighlightConstructor(...ranges));
    return clearPreviewHighlights;
  }, [note.path, previewMarkdown, searchOpen, searchQuery, unlocked]);
  useLayoutEffect(() => {
    const range = previewSearchRangesRef.current[searchIndex];
    const preview = previewRef.current;
    const api = previewHighlightApi();
    api?.registry.delete(previewSearchCurrentHighlight);
    if (!range || !preview) return;
    if (api) api.registry.set(previewSearchCurrentHighlight, new api.HighlightConstructor(range));
    const scroller = previewScroller(preview);
    const matchRect = range.getBoundingClientRect();
    const scrollerRect = scroller.getBoundingClientRect();
    if (matchRect.top < scrollerRect.top || matchRect.bottom > scrollerRect.bottom) {
      scroller.scrollBy({ top: matchRect.top - scrollerRect.top - scroller.clientHeight / 2, behavior: "smooth" });
    }
  }, [note.path, previewMarkdown, searchIndex, searchCount, searchQuery, unlocked]);
  useLayoutEffect(() => {
    const preview = previewRef.current;
    if (!preview) return;
    if (target && target.path !== note.path) return;
    preview.style.setProperty("--anchor-scroll-extra", "0px");
    if (target?.path === note.path && target.fragment) scrollPreviewToFragment(preview, target.fragment, report);
    else previewScroller(preview).scrollTo({ top: 0 });
  }, [note.path, target?.request]);
  useLayoutEffect(() => {
    const preview = previewRef.current;
    const content = preview?.querySelector<HTMLElement>(".markdown-content");
    const split = preview?.closest<HTMLElement>(".mode-split");
    if (!preview || !content) return;
    let appliedOffset = 0;
    const applyOffset = (offset: number) => {
      if (offset === appliedOffset) return;
      appliedOffset = offset;
      content.style.setProperty("--preview-alignment-offset", `${offset}px`);
      preview.style.setProperty("--preview-alignment-space", `${Math.max(0, offset)}px`);
    };
    content.style.setProperty("--preview-alignment-offset", "0px");
    preview.style.setProperty("--preview-alignment-space", "0px");
    const align = () => {
      if (!split || !activeLine) { applyOffset(0); return; }
      const editorLine = split.querySelector<HTMLElement>(".cm-content .cm-activeLine");
      const previewBlock = preview.querySelector<HTMLElement>(".source-line-active");
      if (!editorLine || !previewBlock) { applyOffset(0); return; }
      const editorRect = editorLine.getBoundingClientRect();
      const splitRect = split.getBoundingClientRect();
      if (editorRect.bottom < splitRect.top || editorRect.top > splitRect.bottom) { applyOffset(0); return; }
      const unalignedPreviewTop = previewBlock.getBoundingClientRect().top - appliedOffset;
      applyOffset(Math.round(editorRect.top - unalignedPreviewTop));
    };
    align();
    if (!split) return;
    let frame = 0;
    const scheduleAlignment = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(align);
    };
    split.addEventListener("scroll", scheduleAlignment, { passive: true });
    return () => {
      cancelAnimationFrame(frame);
      split.removeEventListener("scroll", scheduleAlignment);
    };
  }, [activeLine, markdown, note.path]);
  const scrollToFragment = (href: string) => {
    const preview = previewRef.current;
    const fragment = markdownHrefFragment(href);
    if (preview && fragment) scrollPreviewToFragment(preview, fragment, report);
  };
  return <article ref={previewRef} className={`markdown-body ${suppressActiveHighlight?"suppress-active-highlight":""}`}>{searchOpen&&<div className="preview-search-layer"><div className="preview-search" role="search"><Search size={15}/><input ref={previewSearchInputRef} aria-label={t("preview.find")} value={searchQuery} onChange={event=>{setSearchQuery(event.target.value);setSearchIndex(0)}} onKeyDown={event=>{if(event.key==="Enter"){event.preventDefault();moveSearch(event.shiftKey?-1:1)}}} placeholder={t("preview.find")}/><output aria-live="polite">{searchQuery?`${searchIndex<0?0:searchIndex+1} / ${searchCount}`:""}</output><button type="button" aria-label={t("preview.previous")} disabled={!searchCount} onClick={()=>moveSearch(-1)}><ChevronUp size={15}/></button><button type="button" aria-label={t("preview.next")} disabled={!searchCount} onClick={()=>moveSearch(1)}><ChevronDown size={15}/></button><button type="button" aria-label={t("preview.close")} onClick={closeSearch}><X size={15}/></button></div></div>}<div className="markdown-content"><ReactMarkdown remarkPlugins={[remarkGfm, remarkFrontmatter, remarkMath, remarkTableBreaks, remarkTypographicDashes, remarkHidePreviewMetadata]} rehypePlugins={[[rehypeKatex,{trust:false,strict:"ignore"}], rehypeHeadingIds, activeLinePlugin]} components={{
    table: ({ node: _node, ...props }) => <div className="markdown-table-scroll"><table {...props}/></div>,
    a: ({ href, title, children }) => {
      const local = resolveLocalNotePath(note.path, href);
      return <a href={href} title={title} onClick={event => {
        if (href?.startsWith("#")) {
          event.preventDefault();
          scrollToFragment(href);
        } else if (local) {
          event.preventDefault();
          if (notes.some(candidate => candidate.path === local)) navigate(local, event.metaKey || event.ctrlKey, markdownHrefFragment(href));
          else report(t("preview.linkNotFound", { path: local }));
        } else if (href && /^https?:\/\//i.test(href)) {
          event.preventDefault();
          openUrl(href).catch(error => report(String(error)));
        }
      }}>{children}</a>;
    },
    code: ({ className, children }) => {
      const payload = String(children).trim();
      if (!className?.split(" ").includes("language-encrypted")) return <code className={className}>{children}</code>;
      const plaintext = unlocked[payload];
      return plaintext === undefined
        ? <button type="button" className="encrypted-block" onClick={() => unlock(payload)} aria-label={t("preview.unlock")}><LockKeyhole size={17}/><span>****************</span></button>
        : <span className="encrypted-revealed"><span>{plaintext}</span><button type="button" onClick={() => lock(payload)}><UnlockKeyhole size={15}/>{t("preview.lock")}</button></span>;
    },
  }}>{previewMarkdown}</ReactMarkdown></div></article>;
}

function FolderRow({ folder, depth, open, active, select, toggle, menu }: { folder: TreeNode; depth: number; open: boolean; active: boolean; select: (path: string) => void; toggle: (path: string) => void; menu: TreeProps["menu"] }) {
  const draggable = useDraggable({ id: `drag:${folder.path}`, data: { path: folder.path, label: folder.name, kind: "folder" satisfies DragItem["kind"] } });
  const droppable = useDroppable({ id: `drop:${folder.path}`, data: { path: folder.path } });
  const setRef = useCallback((node: HTMLElement | null) => { draggable.setNodeRef(node); droppable.setNodeRef(node); }, [draggable.setNodeRef, droppable.setNodeRef]);
  return <button ref={setRef} className={`tree-folder-row ${active ? "active" : ""} ${droppable.isOver ? "drop-target" : ""} ${draggable.isDragging ? "dragging" : ""}`} style={{ paddingLeft: 8 + depth * 16 }} onClick={() => { select(folder.path); toggle(folder.path); }} onContextMenu={event => menu(event, folder.path, "folder")} {...draggable.listeners} {...draggable.attributes} aria-expanded={open} aria-pressed={active}>{open ? <ChevronDown size={14}/> : <ChevronRight size={14}/>} {open ? <FolderOpen size={16}/> : <Folder size={16}/>}<span>{folder.name}</span></button>;
}

function NoteRow({ note, depth, active, select, menu }: { note: NoteEntry; depth: number; active: string; select: TreeProps["select"]; menu: TreeProps["menu"] }) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({ id: `drag:${note.path}`, data: { path: note.path, label: note.title, kind: "note" satisfies DragItem["kind"] } });
  return <button ref={setNodeRef} className={`${note.path === active ? "note-row active" : "note-row"} ${isDragging ? "dragging" : ""}`} style={{ paddingLeft: 26 + depth * 16 }} onClick={event => select(note.path, event.metaKey || event.ctrlKey)} onContextMenu={event => menu(event, note.path, "note")} title={note.path} {...listeners} {...attributes}><FileText size={15}/><span>{note.title}</span></button>;
}

export function RootDropZone({ label, active, open, select }: { label: string; active: boolean; open: boolean; select: () => void }) {
  const { setNodeRef, isOver } = useDroppable({ id: "drop:__notes_root__", data: { path: "" } });
  return <button ref={setNodeRef} className={`root-drop-zone ${active ? "active" : ""} ${isOver ? "drop-target" : ""}`} type="button" aria-expanded={open} aria-pressed={active} onClick={select}><FolderOpen size={15}/><span>{label}</span>{open?<ChevronDown className="root-drop-zone-chevron" size={14}/>:<ChevronRight className="root-drop-zone-chevron" size={14}/>}</button>;
}

export function DraggableTab({ tab, active, dirty, navigate, close }: { tab: OpenNoteTab; active: boolean; dirty: boolean; navigate: (path: string) => void; close: (path: string) => void }) {
  const draggable = useDraggable({ id: `tab-drag:${tab.path}`, data: { path: tab.path } });
  const droppable = useDroppable({ id: `tab-drop:${tab.path}`, data: { path: tab.path }, disabled: draggable.isDragging });
  const setRef = useCallback((node: HTMLElement | null) => { draggable.setNodeRef(node); droppable.setNodeRef(node); }, [draggable.setNodeRef, droppable.setNodeRef]);
  return <div ref={setRef} className={`document-tab ${active ? "active" : ""} ${draggable.isDragging ? "dragging" : ""} ${droppable.isOver && !draggable.isDragging ? "drop-target" : ""}`}>
    <button {...draggable.listeners} {...draggable.attributes} className="document-tab-label" role="tab" aria-selected={active} title={tab.path} onClick={() => navigate(tab.path)}><span>{tab.title}</span>{dirty&&<i className="document-tab-dirty" role="img" aria-label={t("tab.unsaved")}/>}</button>
    <button className="document-tab-close tooltip-button" data-tooltip={t("tab.close")} aria-label={t("tab.closeLabel", { title: tab.title })} onClick={() => close(tab.path)}><X size={13}/></button>
  </div>;
}

export function NoteTree({ node, depth = 0, active, activeFolder, expanded, forced, select, selectFolder, toggle, menu }: TreeProps) {
  const folders = [...node.folders.values()].sort((left, right) => left.name.localeCompare(right.name));
  const notes = [...node.notes].sort((left, right) => left.title.localeCompare(right.title));
  return <div className="tree-level" role={depth === 0 ? "tree" : "group"}>
    {folders.map(folder => { const open = forced || expanded.has(folder.path); return <div className="tree-folder" key={folder.path}><FolderRow folder={folder} depth={depth} open={open} active={folder.path===activeFolder} select={selectFolder} toggle={toggle} menu={menu}/>{open && <NoteTree node={folder} depth={depth + 1} active={active} activeFolder={activeFolder} expanded={expanded} forced={forced} select={select} selectFolder={selectFolder} toggle={toggle} menu={menu}/>}</div>; })}
    {notes.map(note => <NoteRow key={note.path} note={note} depth={depth} active={active} select={select} menu={menu}/>) }
  </div>;
}

export function PinnedNotes({ notes, active, select, menu }: { notes: NoteEntry[]; active: string; select: TreeProps["select"]; menu: TreeProps["menu"] }) {
  const [open, setOpen] = useState(true);
  if (!notes.length) return null;
  return <section className="pinned-notes" aria-label={t("sidebar.pinned")}>
    <button className="pinned-notes-toggle" type="button" aria-expanded={open} onClick={()=>setOpen(value=>!value)}><Pin size={15}/><span>{t("sidebar.pinned")}</span>{open?<ChevronDown className="pinned-notes-chevron" size={14}/>:<ChevronRight className="pinned-notes-chevron" size={14}/>}</button>
    {open&&<div className="pinned-notes-list">{notes.map(note=><NoteRow key={note.path} note={note} depth={0} active={active} select={select} menu={menu}/>)}</div>}
  </section>;
}

export function PathDialog({ state, folders, rootLabel, onSubmit, onClose }: { state: FormState; folders: FolderEntry[]; rootLabel: string; onSubmit: (value: string) => void; onClose: () => void }) {
  const initial = state.operation === "rename" ? baseName(state.path) : state.operation === "move" ? parentFolder(state.path) : "";
  const [value, setValue] = useState(initial);
  const title = { note: t("dialog.newNote"), folder: t("dialog.newFolder"), rename: t("dialog.rename"), move: t("dialog.moveToFolder") }[state.operation];
  return <Modal title={title} onClose={onClose}><form onSubmit={event => { event.preventDefault(); onSubmit(value); }}><label className="dialog-field"><span>{state.operation === "move" ? t("dialog.destination") : t("dialog.name")}</span>{state.operation === "move" ? <select value={value} onChange={event => setValue(event.target.value)}><option value="">{rootLabel}</option>{folders.filter(folder => folder.path !== state.path && !folder.path.startsWith(`${state.path}/`)).map(folder => <option key={folder.path} value={folder.path}>{folder.path}</option>)}</select> : <input autoFocus value={value} onChange={event => setValue(event.target.value)}/>}</label><div className="modal-actions"><button type="button" onClick={onClose}>{t("common.cancel")}</button><button className="primary" type="submit" disabled={state.operation !== "move" && !value.trim()}>{t("common.confirm")}</button></div></form></Modal>;
}
