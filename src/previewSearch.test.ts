import { describe, expect, it } from "vitest";
import { previewTextMatches } from "./previewSearch";

describe("preview text search", () => {
  it("finds every literal match without case sensitivity", () => {
    expect(previewTextMatches("Notes notes NOTES", "notes")).toEqual([
      { start: 0, end: 5 },
      { start: 6, end: 11 },
      { start: 12, end: 17 },
    ]);
  });

  it("treats regular-expression characters as ordinary text", () => {
    expect(previewTextMatches("Use x^2, not x+2. x^2 wins.", "x^2")).toEqual([
      { start: 4, end: 7 },
      { start: 18, end: 21 },
    ]);
  });

  it("returns no matches for an empty query", () => {
    expect(previewTextMatches("document", "")).toEqual([]);
  });

  it("caps matches to protect the preview from pathological documents", () => {
    expect(previewTextMatches("a a a a", "a", 2)).toHaveLength(2);
  });
});
