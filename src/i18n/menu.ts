import { translate, type Language } from "./index";
import type { MessageKey as Key } from "./locales";
import menuLabelContract from "./menu-contract.json";

const menuLabelKeys = menuLabelContract as Record<string, Key>;

/** Labels for the native menu, keyed by the ids menu.rs understands. */
export function menuLabels(language: Language): Record<string, string> {
  return Object.fromEntries(Object.entries(menuLabelKeys).map(([id, key]) => [id, translate(language, key)]));
}
