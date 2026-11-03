import type { MarkdownNode } from "./types";

function isHtmlComment(node: MarkdownNode) {
  return node.type === "html" && /^\s*<!--[\s\S]*-->\s*$/.test(node.value || "");
}

/**
 * Removes non-document metadata from the rendered tree only. The editor draft and the
 * Markdown file stay byte-for-byte unchanged.
 */
export function hidePreviewMetadata(tree: MarkdownNode) {
  const walk = (node: MarkdownNode) => {
    if (!node.children) return;
    node.children = node.children.filter(child => child.type !== "yaml" && !isHtmlComment(child));
    node.children.forEach(walk);
  };
  walk(tree);
  return tree;
}

export function remarkHidePreviewMetadata() {
  return (tree: MarkdownNode) => { hidePreviewMetadata(tree); };
}
