export type TextMatch = { start: number; end: number };

/** Finds literal, case-insensitive matches while preserving source offsets. */
export function previewTextMatches(text: string, query: string, limit = 2000): TextMatch[] {
  if (!query || limit <= 0) return [];
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  let matcher: RegExp;
  try { matcher = new RegExp(escaped, "giu"); } catch { return []; }
  const matches: TextMatch[] = [];
  for (const match of text.matchAll(matcher)) {
    matches.push({ start: match.index, end: match.index + match[0].length });
    if (matches.length >= limit) break;
  }
  return matches;
}
