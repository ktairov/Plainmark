import { describe, expect, it } from "vitest";
import type { NoteContent, OpenNoteTab } from "./model";
import { activeTabSession, dirtyTabSessionPaths, emptyTabSessionState, tabSessionReducer } from "./tabSession";

const note = (path: string, markdown = `# ${path}`): NoteContent => ({
  path,
  title: path,
  markdown,
  headings: [],
  modified_ms: 1,
});

const restored = (...paths: string[]) => tabSessionReducer(emptyTabSessionState, {
  type: "restore",
  tabs: paths.map(path => ({ path, title: path, mode: "preview" }) satisfies OpenNoteTab),
  activePath: paths[0] ?? "",
});

describe("tabSessionReducer", () => {
  it("keeps the document, draft, mode, and scroll position in one session", () => {
    let state = restored("one.md");
    state = tabSessionReducer(state, { type: "load", note: note("one.md"), disposition: "switch", mode: "preview" });
    state = tabSessionReducer(state, { type: "draft", path: "one.md", draft: "changed" });
    state = tabSessionReducer(state, { type: "mode", path: "one.md", mode: "split" });
    state = tabSessionReducer(state, { type: "scroll", path: "one.md", mode: "split", top: 144 });

    expect(activeTabSession(state)).toMatchObject({ draft: "changed", mode: "split", scroll: { split: 144 } });
    expect(dirtyTabSessionPaths(state)).toEqual(["one.md"]);
  });

  it("preserves an unsaved draft when an existing tab is activated", () => {
    let state = restored("one.md", "two.md");
    state = tabSessionReducer(state, { type: "load", note: note("one.md"), disposition: "switch", mode: "preview" });
    state = tabSessionReducer(state, { type: "draft", path: "one.md", draft: "unsaved" });
    state = tabSessionReducer(state, { type: "load", note: note("two.md"), disposition: "switch", mode: "edit" });
    state = tabSessionReducer(state, { type: "load", note: note("one.md", "disk"), disposition: "switch", mode: "preview", preserveDraft: true });

    expect(activeTabSession(state)?.draft).toBe("unsaved");
    expect(activeTabSession(state)?.document?.markdown).toBe("# one.md");
  });

  it("records save results without discarding edits made while saving", () => {
    let state = restored("one.md");
    state = tabSessionReducer(state, { type: "load", note: note("one.md"), disposition: "switch", mode: "preview" });
    state = tabSessionReducer(state, { type: "draft", path: "one.md", draft: "submitted" });
    state = tabSessionReducer(state, { type: "save-start", path: "one.md" });
    state = tabSessionReducer(state, { type: "draft", path: "one.md", draft: "newer" });
    state = tabSessionReducer(state, { type: "save-success", path: "one.md", note: note("one.md", "submitted"), submittedDraft: "submitted" });

    expect(activeTabSession(state)).toMatchObject({ draft: "newer", saveStatus: "idle" });
    expect(dirtyTabSessionPaths(state)).toEqual(["one.md"]);
  });

  it("closes the active tab and activates its nearest neighbour", () => {
    const state = tabSessionReducer(restored("one.md", "two.md", "three.md"), { type: "close", path: "one.md" });
    expect(state.tabs.map(tab => tab.path)).toEqual(["two.md", "three.md"]);
    expect(state.activePath).toBe("two.md");
  });

  it("closes other tabs without losing the active draft, mode, or scroll", () => {
    let state = restored("one.md", "two.md", "three.md");
    state = tabSessionReducer(state, { type: "load", note: note("two.md"), disposition: "switch", mode: "edit" });
    state = tabSessionReducer(state, { type: "mode", path: "two.md", mode: "edit" });
    state = tabSessionReducer(state, { type: "draft", path: "two.md", draft: "keep this draft" });
    state = tabSessionReducer(state, { type: "scroll", path: "two.md", mode: "edit", top: 250 });
    state = tabSessionReducer(state, { type: "close-many", paths: ["one.md", "three.md"] });
    expect(state.tabs.map(tab => tab.path)).toEqual(["two.md"]);
    expect(state.activePath).toBe("two.md");
    expect(activeTabSession(state)).toMatchObject({ draft: "keep this draft", mode: "edit", scroll: { edit: 250 } });
    expect(dirtyTabSessionPaths(state)).toEqual(["two.md"]);
  });

  it("closes all tabs into an empty session and can open a note afterwards", () => {
    let state = tabSessionReducer(restored("one.md", "two.md"), { type: "close-many", paths: ["one.md", "two.md"] });
    expect(state).toEqual(emptyTabSessionState);
    state = tabSessionReducer(state, { type: "load", note: note("three.md"), disposition: "replace", mode: "preview" });
    expect(state.activePath).toBe("three.md");
    expect(state.tabs).toHaveLength(1);
  });

  it("remaps folder descendants without losing session state", () => {
    let state = restored("old/one.md");
    state = tabSessionReducer(state, { type: "load", note: note("old/one.md"), disposition: "switch", mode: "preview" });
    state = tabSessionReducer(state, { type: "draft", path: "old/one.md", draft: "kept" });
    state = tabSessionReducer(state, { type: "remap", from: "old", to: "new" });

    expect(state.activePath).toBe("new/one.md");
    expect(activeTabSession(state)).toMatchObject({ path: "new/one.md", draft: "kept" });
    expect(activeTabSession(state)?.document?.path).toBe("new/one.md");
  });

  it("removes all sessions below a deleted folder", () => {
    const state = tabSessionReducer(restored("keep.md", "gone/one.md", "gone/two.md"), { type: "remove-tree", path: "gone" });
    expect(state.tabs.map(tab => tab.path)).toEqual(["keep.md"]);
  });
});
