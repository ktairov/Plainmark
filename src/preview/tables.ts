import type { MarkdownNode } from "./types";

/** Converts only safe, known HTML layout tags inside GFM table cells. */
export function remarkTableBreaks() {
  return (tree: MarkdownNode) => {
    const walk = (node: MarkdownNode) => {
      if (node.type === "table") for (const row of node.children || []) for (const cell of row.children || []) {
        let hasListItem = false;
        cell.children = (cell.children || []).flatMap(child => {
          if (child.type !== "html") return [child];
          if (/^<br\s*\/?\s*>$/i.test(child.value || "")) return [{ type: "break" }];
          if (!/^(?:\s*<\/?(?:ul|li)\s*>\s*)+$/i.test(child.value || "")) return [child];
          const nodes: MarkdownNode[] = [];
          for (const tag of (child.value || "").matchAll(/<\/?(?:ul|li)\s*>/gi)) {
            if (/^<li/i.test(tag[0])) {
              if (hasListItem) nodes.push({ type: "break" });
              nodes.push({ type: "text", value: "• " });
              hasListItem = true;
            }
          }
          return nodes;
        });
      }
      node.children?.forEach(walk);
    };
    walk(tree);
  };
}
