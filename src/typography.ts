/** Applies display-only typography to Markdown text nodes. Source Markdown is never changed. */
export function typographicDashes(value: string) {
  return value.replace(/---/g, "—").replace(/--/g, "–");
}
