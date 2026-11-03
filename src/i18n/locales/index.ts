import type { Message } from "../types";
import { en, type MessageKey } from "./en";
import { ru } from "./ru";

export const localeRegistry = {
  en: { nativeName: "English", messages: en as Record<MessageKey, Message> },
  ru: { nativeName: "Русский", messages: ru },
} as const;

export type Language = keyof typeof localeRegistry;

export const supportedLanguages = Object.keys(localeRegistry) as Language[];

const messageKeys = new Set<string>(Object.keys(en));

export function hasMessageKey(key: string): key is MessageKey {
  return messageKeys.has(key);
}

export function isSupportedLanguage(value: unknown): value is Language {
  return typeof value === "string" && Object.prototype.hasOwnProperty.call(localeRegistry, value);
}

export { en, ru, type MessageKey };
