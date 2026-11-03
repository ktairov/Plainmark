export type Mode = "preview" | "edit" | "split";
export type SaveStatus = "idle" | "saving" | "saved" | "error" | "conflict";
export type RefreshStatus = "idle" | "refreshing" | "success" | "error";
export type MenuTarget = { path: string; kind: "root" | "folder" | "note"; x: number; y: number };
export type FormState = { operation: "note" | "folder" | "rename" | "move"; path: string; kind: "root" | "folder" | "note" };
export type Pending =
  | { kind: "close-tab"; path: string }
  | { kind: "close-tabs"; paths: string[] }
  | { kind: "close" }
  | { kind: "quit" };
export type CryptoDialog = { kind: "encrypt"; from: number; to: number; plaintext: string } | { kind: "decrypt"; payload: string };
export type DragItem = { path: string; label: string; kind: "folder" | "note" };
