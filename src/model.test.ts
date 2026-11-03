import { describe, expect, it } from "vitest";
import { activeFolderAfterDelete, addedNotePaths, buildNoteTree, encryptedMarkdownBlock, errorKind, errorMessage, parseAppError, folderTarget, idleStatusText, markdownHeadingId, markdownHrefFragment, markActivePreviewBlock, parsePinnedNotes, parseStoredTabs, previewAnchorScrollPlan, remapPath, remapPinnedPaths, removePinnedPaths, reorderTabs, resolveLocalNotePath, restoredOpenTabs, rootFolderName, selectedFileLocation, shouldOpenFallbackNote, tabsAfterClose, togglePinnedPath, updateOpenTabs, type SourceNode } from "./model";
import { defaultAppSettings, parseAppSettings, resolvedColorMode, resolvedTheme, toggledColorMode } from "./settings";

describe("tree and path helpers", () => {
  it("keeps empty folders in the tree", () => {
    const tree = buildNoteTree([], [{ path: "a/empty", name: "empty" }]);
    expect(tree.folders.get("a")?.folders.has("empty")).toBe(true);
  });
  it("remaps a node and its descendants", () => {
    expect(remapPath("a/b/note.md", "a", "moved/a")).toBe("moved/a/b/note.md");
    expect(remapPath("another.md", "a", "b")).toBe("another.md");
  });
  it("uses the active folder as the creation target", () => {
    expect(folderTarget("thoughts/ideas")).toEqual({ path: "thoughts/ideas", kind: "folder" });
    expect(folderTarget("")).toEqual({ path: "", kind: "root" });
    expect(activeFolderAfterDelete("thoughts/ideas/drafts", "thoughts/ideas")).toBe("thoughts");
    expect(activeFolderAfterDelete("other", "thoughts/ideas")).toBe("other");
  });
  it("resolves safe local links", () => {
    expect(resolveLocalNotePath("a/b/current.md", "../other.md#x")).toBe("a/other.md");
    expect(resolveLocalNotePath("a.md", "../../outside.md")).toBeNull();
    expect(resolveLocalNotePath("a.md", "https://example.com/x.md")).toBeNull();
  });
  it("creates stable Unicode heading anchors", () => {
    expect(markdownHeadingId("Таблицы и списки")).toBe("таблицы-и-списки");
    expect(markdownHeadingId("What's new? — v2")).toBe("whats-new-v2");
    expect(markdownHrefFragment("other.md#обновление-%D1%81%D0%B8%D1%81%D1%82%D0%B5%D0%BC%D1%8B")).toBe("обновление-системы");
    expect(previewAnchorScrollPlan(200, 500, 100, 900, 600)).toEqual({ top: 600, extra: 300 });
  });
  it("extracts structured backend errors", () => {
    expect(errorMessage({ kind: "io", message: "permission denied" })).toBe("permission denied");
    expect(errorMessage('{"kind":"invalid_path","message":"outside"}')).toBe("outside");
    expect(errorMessage({ kind: "io" })).toBe("io");
    expect(errorKind({ code: "conflict", message: "changed on disk" })).toBe("conflict");
    expect(errorKind({ kind: "conflict", message: "changed on disk" })).toBe("conflict");
    expect(errorKind('{"kind":"conflict","message":"changed on disk"}')).toBe("conflict");
    expect(parseAppError("plain failure")).toEqual({ code: "", message: "plain failure" });
    expect(parseAppError({ code: "itemExists", message: "exists", details: { name: "a.md" } })).toEqual({ code: "itemExists", message: "exists", details: { name: "a.md" } });
    expect(errorMessage({ code: "itemExists", message: "An item with this name already exists." })).toBe("An item with this name already exists.");
  });
  it("creates a fenced encrypted block without joining surrounding prose", () => {
    expect(encryptedMarkdownBlock("enc:v1:data", "x", "y")).toBe("\n\n```encrypted\nenc:v1:data\n```\n\n");
    expect(encryptedMarkdownBlock("enc:v1:data")).toBe("```encrypted\nenc:v1:data\n```");
  });
  it("shows only the selected root folder name", () => {
    expect(rootFolderName("")).toBe("Plainmark");
    expect(rootFolderName("/Users/user/Projects/Notes")).toBe("Notes");
    expect(rootFolderName("/Users/user/Projects/Notes/thoughts/")).toBe("Thoughts");
    expect(rootFolderName("C:\\Notes\\work")).toBe("Work");
    expect(rootFolderName("/Users/user/заметки")).toBe("Заметки");
  });
  it("uses the selected file's parent folder as the notes root", () => {
    expect(selectedFileLocation("/Users/user/Notes/article.md")).toEqual({ root: "/Users/user/Notes", fileName: "article.md" });
    expect(selectedFileLocation("/article.md")).toEqual({ root: "/", fileName: "article.md" });
    expect(selectedFileLocation("C:\\Notes\\article.mdx")).toEqual({ root: "C:\\Notes", fileName: "article.mdx" });
    expect(selectedFileLocation("article.md")).toBeNull();
  });
  it("shows useful idle status instead of the watcher file count", () => {
    expect(idleStatusText(false, "", 0)).toBe("Loading notes…");
    expect(idleStatusText(true, "thoughts/note.md", 230)).toBe("Opened thoughts/note.md");
    expect(idleStatusText(true, "", 230)).toBe("No note open");
    expect(idleStatusText(true, "", 0)).toBe("No Markdown notes found");
  });
  it("finds only paths newly added to the note tree", () => {
    expect(addedNotePaths([{ path: "one.md" }], [{ path: "one.md" }, { path: "two.md" }])).toEqual(["two.md"]);
    expect(addedNotePaths([{ path: "one.md" }], [{ path: "one.md" }])).toEqual([]);
  });
  it("loads and updates workspace-specific pinned notes", () => {
    expect(parsePinnedNotes('{"/notes":["one.md","one.md","folder/two.md"],"broken":[1,null]}')).toEqual({ "/notes": ["one.md", "folder/two.md"] });
    expect(parsePinnedNotes("broken")).toEqual({});
    expect(togglePinnedPath(["one.md"], "two.md")).toEqual(["one.md", "two.md"]);
    expect(togglePinnedPath(["one.md", "two.md"], "one.md")).toEqual(["two.md"]);
  });
  it("keeps pins in sync with tree moves and deletes", () => {
    expect(remapPinnedPaths(["folder/one.md", "other.md"], "folder", "archive/folder")).toEqual(["archive/folder/one.md", "other.md"]);
    expect(removePinnedPaths(["folder/one.md", "folder/deep/two.md", "other.md"], "folder")).toEqual(["other.md"]);
  });
  it("loads and validates appearance settings", () => {
    expect(parseAppSettings(null)).toEqual(defaultAppSettings);
    expect(parseAppSettings(JSON.stringify({ dayTheme: "gruvbox", nightTheme: "mariana", colorMode: "dark", treeFontSize: 99, contentFontSize: 8, previewWidth: 9999 }))).toMatchObject({ dayTheme: "gruvbox", nightTheme: "mariana", colorMode: "dark", treeFontSize: 21, contentFontSize: 13, previewWidth: 1400 });
    expect(parseAppSettings('{"theme":"violet"}')).toMatchObject({ dayTheme: "ayu", nightTheme: "mariana" });
    expect(parseAppSettings('{"dayTheme":"missing","nightTheme":"missing","uiFont":"missing"}')).toMatchObject({ dayTheme: "ayu", nightTheme: "mariana", uiFont: "system" });
    expect(resolvedColorMode("system", true)).toBe("dark");
    expect(resolvedColorMode("light", true)).toBe("light");
    expect(resolvedTheme({ ...defaultAppSettings, dayTheme: "gruvbox", nightTheme: "mariana" }, "light")).toBe("gruvbox");
    expect(resolvedTheme({ ...defaultAppSettings, dayTheme: "gruvbox", nightTheme: "mariana" }, "dark")).toBe("mariana");
    expect(toggledColorMode("light")).toBe("dark");
    expect(toggledColorMode("dark")).toBe("light");
  });
  it("opens, switches, and replaces note tabs without duplicates", () => {
    const first = [{ path: "one.md", title: "One", mode: "edit" as const }];
    const opened = updateOpenTabs(first, { path: "two.md", title: "Two", mode: "preview" }, "one.md", "open");
    expect(opened.map(tab => tab.path)).toEqual(["one.md", "two.md"]);
    expect(updateOpenTabs(opened, { path: "two.md", title: "Two", mode: "split" }, "one.md", "open")).toBe(opened);
    expect(updateOpenTabs(opened, { path: "two.md", title: "Two", mode: "split" }, "one.md", "replace")).toBe(opened);
    const replaced = updateOpenTabs(opened, { path: "three.md", title: "Three", mode: "preview" }, "one.md", "replace");
    expect(replaced.map(tab => tab.path)).toEqual(["three.md", "two.md"]);
    expect(replaced[0].mode).toBe("edit");
    expect(updateOpenTabs(first, { path: "one.md", title: "Renamed", mode: "preview" }, "one.md", "replace")[0]).toMatchObject({ title: "Renamed", mode: "edit" });
  });
  it("selects the neighboring tab after closing one", () => {
    const tabs = [{ path: "one.md", title: "One", mode: "preview" as const }, { path: "two.md", title: "Two", mode: "edit" as const }, { path: "three.md", title: "Three", mode: "split" as const }];
    expect(tabsAfterClose(tabs, "two.md")).toEqual({ tabs: [tabs[0], tabs[2]], nextPath: "three.md" });
    expect(tabsAfterClose(tabs, "three.md").nextPath).toBe("two.md");
  });
  it("reorders tabs by their paths", () => {
    const tabs = [{ path: "one.md", title: "One", mode: "preview" as const }, { path: "two.md", title: "Two", mode: "edit" as const }, { path: "three.md", title: "Three", mode: "split" as const }];
    expect(reorderTabs(tabs, "one.md", "three.md").map(tab => tab.path)).toEqual(["two.md", "three.md", "one.md"]);
    expect(reorderTabs(tabs, "three.md", "one.md").map(tab => tab.path)).toEqual(["three.md", "one.md", "two.md"]);
    expect(reorderTabs(tabs, "missing.md", "one.md")).toBe(tabs);
  });
  it("restores per-tab modes and migrates the old path-only session", () => {
    expect(parseStoredTabs('{"tabs":[{"path":"one.md","mode":"edit"},{"path":"two.md","mode":"split"},{"path":"one.md","mode":"preview"}],"active":"two.md"}')).toEqual({ tabs: [{ path: "one.md", mode: "edit" }, { path: "two.md", mode: "split" }], active: "two.md", persisted: true });
    expect(parseStoredTabs('{"paths":["one.md"],"active":"missing.md"}')).toEqual({ tabs: [{ path: "one.md", mode: "preview" }], active: "one.md", persisted: true });
    expect(parseStoredTabs('{"tabs":[],"active":""}')).toEqual({ tabs: [], active: "", persisted: true });
    expect(parseStoredTabs(null)).toEqual({ tabs: [], active: "", persisted: false });
    expect(parseStoredTabs("broken")).toEqual({ tabs: [], active: "", persisted: false });
  });
  it("preserves an explicitly closed tab session", () => {
    expect(shouldOpenFallbackNote(parseStoredTabs('{"tabs":[],"active":""}'), 0)).toBe(false);
    expect(shouldOpenFallbackNote(parseStoredTabs(null), 0)).toBe(true);
    expect(shouldOpenFallbackNote(parseStoredTabs('{"tabs":[{"path":"deleted.md","mode":"edit"}],"active":"deleted.md"}'), 0)).toBe(true);
    expect(shouldOpenFallbackNote(parseStoredTabs('{"tabs":[{"path":"one.md","mode":"split"}],"active":"one.md"}'), 1)).toBe(false);
  });
  it("restores tab metadata in one pass without activating every document", () => {
    const stored = parseStoredTabs('{"tabs":[{"path":"one.md","mode":"edit"},{"path":"missing.md","mode":"split"},{"path":"two.md","mode":"preview"}],"active":"two.md"}');
    expect(restoredOpenTabs(stored, [{ path: "two.md", title: "Two" }, { path: "one.md", title: "One" }])).toEqual([
      { path: "one.md", title: "One", mode: "edit" },
      { path: "two.md", title: "Two", mode: "preview" },
    ]);
  });
  it("marks the most specific preview block for the editor line", () => {
    const paragraph: SourceNode = { type: "element", tagName: "p", properties: {}, position: { start: { line: 4 }, end: { line: 6 } } };
    const quote: SourceNode = { type: "element", tagName: "blockquote", properties: {}, position: { start: { line: 3 }, end: { line: 7 } }, children: [paragraph] };
    const tree: SourceNode = { type: "root", children: [quote] };
    expect(markActivePreviewBlock(tree, 5)).toBe(true);
    expect(paragraph.properties?.className).toContain("source-line-active");
    expect(quote.properties?.className ?? []).not.toContain("source-line-active");
    expect(markActivePreviewBlock(tree, 20)).toBe(false);
    expect(paragraph.properties?.className ?? []).not.toContain("source-line-active");
  });
});
