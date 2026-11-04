// Export the same Lucide vectors used by the preserved React UI.
// Run from any directory after npm ci in the repository root.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const source = path.resolve(root, "../node_modules/lucide-react");
const target = path.join(root, "internal/icons");
const names = ["panel-left-close", "panel-left-open", "folder-open", "folder", "folder-plus", "file-plus-2", "file-text", "eye", "pencil-line", "columns-2", "sun", "moon", "search", "save", "ellipsis", "x", "pin", "settings", "lock-keyhole", "type", "palette", "languages", "check", "chevron-down", "plus"];
fs.mkdirSync(path.join(target, "svg"), { recursive: true });
const escape = text => String(text).replaceAll("&", "&amp;").replaceAll('"', "&quot;").replaceAll("<", "&lt;");
for (const name of names) {
  const { __iconNode: nodes } = await import(pathToFileURL(path.join(source, "dist/esm/icons", name + ".js")));
  const body = nodes.map(([tag, attributes]) => `<${tag} ${Object.entries(attributes).filter(([key]) => key !== "key").map(([key, value]) => `${key}="${escape(value)}"`).join(" ")}/>`).join("");
  fs.writeFileSync(path.join(target, "svg", name + ".svg"), `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${body}</svg>\n`);
}
fs.copyFileSync(path.join(source, "LICENSE"), path.join(target, "LICENSE"));
