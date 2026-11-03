import { hasMessageKey, t, type MessageKey } from "./i18n";

export type Heading = { level: number; text: string };
export type NoteEntry = { path: string; title: string; folder: string; headings: Heading[]; modified_ms: number; size: number };
export type FolderEntry = { path: string; name: string };
export type WorkspaceSnapshot = { notes: NoteEntry[]; folders: FolderEntry[] };
export type NoteContent = { path: string; title: string; markdown: string; headings: Heading[]; modified_ms: number };
export type PathResult = { path: string; kind: "note" | "folder" };
export type TrashInfo = { path: string; name: string; nested_files: number; is_directory: boolean };
export type TreeNode = { name: string; path: string; folders: Map<string, TreeNode>; notes: NoteEntry[] };
export type OpenNoteTab = { path: string; title: string; mode: "preview" | "edit" | "split" };
export type TabDisposition = "replace" | "open" | "switch";
export type StoredTabs = { tabs: Array<Pick<OpenNoteTab, "path" | "mode">>; active: string; persisted: boolean };
export type PinnedNotesByRoot = Record<string, string[]>;
export type SourceNode = { type?: string; tagName?: string; value?: string; properties?: Record<string, unknown>; position?: { start: { line: number }; end: { line: number } }; children?: SourceNode[] };

const previewBlockTags = new Set(["p", "h1", "h2", "h3", "h4", "h5", "h6", "li", "blockquote", "pre", "table", "tr", "hr"]);

export function markdownHeadingId(text: string) {
  return text.trim().toLocaleLowerCase().replace(/[^\p{L}\p{N}\s_-]/gu, "").replace(/\s+/g, "-");
}

export function markdownHrefFragment(href?: string) {
  const marker = href?.indexOf("#") ?? -1;
  if (marker < 0 || !href) return "";
  try { return decodeURIComponent(href.slice(marker + 1)); } catch { return ""; }
}

export function previewAnchorScrollPlan(currentScrollTop: number, targetTop: number, scrollerTop: number, scrollHeight: number, clientHeight: number) {
  const top = Math.max(0, currentScrollTop + targetTop - scrollerTop);
  return { top, extra: Math.max(0, top - Math.max(0, scrollHeight - clientHeight)) };
}

export function idleStatusText(ready: boolean, activePath: string, noteCount: number) {
  if (!ready) return t("app.status.loading");
  if (activePath) return t("app.status.opened", { path: activePath });
  return noteCount > 0 ? t("app.status.noNoteOpen") : t("app.status.noNotes");
}

export function addedNotePaths(previous: Pick<NoteEntry, "path">[], next: Pick<NoteEntry, "path">[]) {
  const known = new Set(previous.map(note => note.path));
  return next.map(note => note.path).filter(path => !known.has(path));
}

export function parsePinnedNotes(raw: string | null): PinnedNotesByRoot {
  if (!raw) return {};
  try {
    const value = JSON.parse(raw) as unknown;
    if (!value || typeof value !== "object" || Array.isArray(value)) return {};
    return Object.fromEntries(Object.entries(value).flatMap(([root, paths]) => {
      if (!root || !Array.isArray(paths)) return [];
      const unique = [...new Set(paths.filter((path): path is string => typeof path === "string" && Boolean(path)))];
      return unique.length ? [[root, unique]] : [];
    }));
  } catch { return {}; }
}

export function togglePinnedPath(paths: string[], path: string) {
  return paths.includes(path) ? paths.filter(candidate => candidate !== path) : [...paths, path];
}

export function remapPinnedPaths(paths: string[], from: string, to: string) {
  return [...new Set(paths.map(path => remapPath(path, from, to)))];
}

export function removePinnedPaths(paths: string[], removedPath: string) {
  return paths.filter(path => path !== removedPath && !path.startsWith(`${removedPath}/`));
}

export function updateOpenTabs(current: OpenNoteTab[], note: OpenNoteTab, previous: string, disposition: TabDisposition) {
  const existing = current.some(tab => tab.path === note.path);
  if (disposition === "open" || disposition === "switch") return existing ? current : [...current, note];
  if (!previous) return existing ? current : [...current, note];
  if (previous === note.path) return current.map(tab => tab.path === note.path ? { ...note, mode: tab.mode } : tab);
  if (existing) return current;
  return current.map(tab => tab.path === previous ? { ...note, mode: tab.mode } : tab);
}

export function tabsAfterClose(current: OpenNoteTab[], path: string) {
  const index = current.findIndex(tab => tab.path === path);
  if (index < 0) return { tabs: current, nextPath: "" };
  const tabs = current.filter(tab => tab.path !== path);
  return { tabs, nextPath: tabs[Math.min(index, tabs.length - 1)]?.path || "" };
}

export function reorderTabs(current: OpenNoteTab[], activePath: string, overPath: string) {
  const from = current.findIndex(tab => tab.path === activePath);
  const to = current.findIndex(tab => tab.path === overPath);
  if (from < 0 || to < 0 || from === to) return current;
  const tabs = [...current];
  const [moved] = tabs.splice(from, 1);
  tabs.splice(to, 0, moved);
  return tabs;
}

export function parseStoredTabs(raw: string | null): StoredTabs {
  if (!raw) return { tabs: [], active: "", persisted: false };
  try {
    const value = JSON.parse(raw) as { tabs?: unknown; paths?: unknown; active?: unknown };
    const modes = new Set(["preview", "edit", "split"]);
    const rawTabs = Array.isArray(value.tabs)
      ? value.tabs
      : Array.isArray(value.paths) ? value.paths.map(path => ({ path, mode: "preview" })) : [];
    const seen = new Set<string>();
    const tabs = rawTabs.flatMap(candidate => {
      if (!candidate || typeof candidate !== "object") return [];
      const { path, mode } = candidate as { path?: unknown; mode?: unknown };
      if (typeof path !== "string" || !path || seen.has(path)) return [];
      seen.add(path);
      return [{ path, mode: modes.has(String(mode)) ? mode as OpenNoteTab["mode"] : "preview" as const }];
    });
    const active = typeof value.active === "string" && tabs.some(tab => tab.path === value.active) ? value.active : tabs[0]?.path || "";
    return { tabs, active, persisted: true };
  } catch {
    return { tabs: [], active: "", persisted: false };
  }
}

export function shouldOpenFallbackNote(stored: StoredTabs, restoredCount: number) {
  return restoredCount === 0 && (!stored.persisted || stored.tabs.length > 0);
}

export function restoredOpenTabs(stored: StoredTabs, notes: Pick<NoteEntry, "path" | "title">[]) {
  const available = new Map(notes.map(note => [note.path, note.title]));
  return stored.tabs.flatMap(tab => {
    const title = available.get(tab.path);
    return title === undefined ? [] : [{ ...tab, title }];
  });
}

export function buildNoteTree(notes: NoteEntry[], folders: FolderEntry[]): TreeNode {
  const root: TreeNode = { name: "", path: "", folders: new Map(), notes: [] };
  const ensure = (path: string) => {
    let node = root;
    for (const part of path.split("/").filter(Boolean)) {
      const childPath = node.path ? `${node.path}/${part}` : part;
      let child = node.folders.get(part);
      if (!child) { child = { name: part, path: childPath, folders: new Map(), notes: [] }; node.folders.set(part, child); }
      node = child;
    }
    return node;
  };
  folders.forEach((folder) => ensure(folder.path));
  notes.forEach((note) => { const parts = note.path.split("/"); parts.pop(); ensure(parts.join("/")).notes.push(note); });
  return root;
}

export function parentFolder(path: string) {
  return path.split("/").slice(0, -1).join("/");
}

export function baseName(path: string) {
  return path.split("/").pop() || path;
}

export function folderAncestors(path: string) {
  const parts = path.split("/"); parts.pop();
  return parts.map((_, i) => parts.slice(0, i + 1).join("/"));
}

export function folderTarget(path: string) {
  return path ? { path, kind: "folder" as const } : { path: "", kind: "root" as const };
}

export function activeFolderAfterDelete(activeFolder: string, deletedPath: string) {
  return activeFolder === deletedPath || activeFolder.startsWith(`${deletedPath}/`) ? deletedPath.split("/").slice(0, -1).join("/") : activeFolder;
}

export function remapPath(path: string, from: string, to: string) {
  return path === from ? to : path.startsWith(`${from}/`) ? `${to}${path.slice(from.length)}` : path;
}

export function rootFolderName(path: string) {
  const name = path.replace(/[\\/]+$/, "").split(/[\\/]/).pop() || "Plainmark";
  return `${name.charAt(0).toLocaleUpperCase()}${name.slice(1)}`;
}

export function selectedFileLocation(path: string) {
  const separatorIndex = Math.max(path.lastIndexOf("/"), path.lastIndexOf("\\"));
  const fileName = path.slice(separatorIndex + 1);
  if (separatorIndex < 0 || !fileName) return null;
  const windowsRoot = separatorIndex === 2 && /^[A-Za-z]:[\\/]/.test(path);
  const root = separatorIndex === 0 || windowsRoot ? path.slice(0, separatorIndex + 1) : path.slice(0, separatorIndex);
  return { root, fileName };
}

export function markActivePreviewBlock(tree: SourceNode, line: number) {
  const candidates: { node: SourceNode; span: number; depth: number }[] = [];
  const walk = (node: SourceNode, depth: number) => {
    const existing = node.properties?.className;
    const classes = Array.isArray(existing) ? existing.map(String) : typeof existing === "string" ? existing.split(/\s+/) : [];
    if (node.properties && classes.includes("source-line-active")) node.properties.className = classes.filter(value => value !== "source-line-active");
    const position = node.position;
    if (node.type === "element" && node.tagName && previewBlockTags.has(node.tagName) && position && position.start.line <= line && position.end.line >= line) {
      candidates.push({ node, span: position.end.line - position.start.line, depth });
    }
    node.children?.forEach(child => walk(child, depth + 1));
  };
  walk(tree, 0);
  candidates.sort((a, b) => a.span - b.span || b.depth - a.depth);
  const active = candidates[0]?.node;
  if (!active) return false;
  active.properties ||= {};
  const existing = active.properties.className;
  const classes = Array.isArray(existing) ? existing.map(String) : typeof existing === "string" ? existing.split(/\s+/) : [];
  active.properties.className = [...classes, "source-line-active"];
  return true;
}

export function encryptedMarkdownBlock(payload: string, before = "", after = "") {
  const leading = before && before !== "\n" ? "\n\n" : "";
  const trailing = after && after !== "\n" ? "\n\n" : "";
  return `${leading}\`\`\`encrypted\n${payload.trim()}\n\`\`\`${trailing}`;
}

export function resolveLocalNotePath(currentPath: string, href?: string) {
  if (!href || href.startsWith("#") || href.startsWith("//") || /^[a-z][a-z\d+.-]*:/i.test(href)) return null;
  let linked: string;
  try { linked = decodeURIComponent(href.split(/[?#]/, 1)[0]).replaceAll("\\", "/"); } catch { return null; }
  if (!/\.mdx?$/i.test(linked)) return null;
  const parts = linked.startsWith("/") ? [] : currentPath.split("/").slice(0, -1);
  for (const part of linked.replace(/^\/+/, "").split("/")) {
    if (!part || part === ".") continue;
    if (part === "..") { if (!parts.length) return null; parts.pop(); } else parts.push(part);
  }
  return parts.join("/");
}

export type AppError = { code: string; message: string; details?: Record<string, string> };

const errorText = (value: unknown) => (value === undefined || value === null ? "" : String(value));

function errorDetails(value: unknown): Record<string, string> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
  const entries = Object.entries(value as Record<string, unknown>).map(([key, detail]) => [key, errorText(detail)] as const);
  return entries.length ? Object.fromEntries(entries) : undefined;
}

/** Backend commands reject with `{ code, message, details }`; older builds used `{ kind, message }`. */
export function parseAppError(error: unknown): AppError {
  const direct = error && typeof error === "object"
    ? (error as { code?: unknown; kind?: unknown; message?: unknown; details?: unknown })
    : null;
  if (direct && (direct.code !== undefined || direct.kind !== undefined || direct.message !== undefined)) {
    const code = errorText(direct.code ?? direct.kind);
    return { code, message: errorText(direct.message) || code, details: errorDetails(direct.details) };
  }
  try {
    const parsed = JSON.parse(String(error)) as { code?: unknown; kind?: unknown; message?: unknown; details?: unknown } | null;
    if (parsed && typeof parsed === "object") {
      const code = errorText(parsed.code ?? parsed.kind);
      return { code, message: errorText(parsed.message) || code, details: errorDetails(parsed.details) };
    }
  } catch { /* Not a JSON payload; fall through to the raw text. */ }
  return { code: "", message: String(error) };
}

export function errorKind(error: unknown): string {
  return parseAppError(error).code;
}

export function errorMessage(error: unknown): string {
  const parsed = parseAppError(error);
  const key = parsed.code ? (`error.${parsed.code}` as MessageKey) : null;
  if (key && hasMessageKey(key)) return t(key, parsed.details);
  return parsed.message;
}
