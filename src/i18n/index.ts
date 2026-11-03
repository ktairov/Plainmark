import { hasMessageKey, isSupportedLanguage, localeRegistry, supportedLanguages, type Language, type MessageKey } from "./locales";
import type { Message, PluralForms } from "./types";

export type LanguageSetting = "system" | Language;
export type TranslateVars = Record<string, string | number>;
export type Translate = (key: MessageKey, vars?: TranslateVars) => string;
export type TranslatePlural = (key: MessageKey, count: number, vars?: TranslateVars) => string;

let activeLanguage: Language = "en";
const languageListeners = new Set<() => void>();

export function setActiveLanguage(language: Language) {
  if (activeLanguage === language) return;
  activeLanguage = language;
  languageListeners.forEach(listener => listener());
}

export function getActiveLanguage() {
  return activeLanguage;
}

export function subscribeActiveLanguage(listener: () => void) {
  languageListeners.add(listener);
  return () => {
    languageListeners.delete(listener);
  };
}

/** Resolves the language setting against the system locale list. */
export function resolveLanguage(setting: LanguageSetting | undefined, locales: readonly string[]): Language {
  if (setting && setting !== "system" && isSupportedLanguage(setting)) return setting;
  for (const locale of locales) {
    const base = locale.toLowerCase().split(/[-_]/)[0];
    if (isSupportedLanguage(base)) return base;
  }
  return "en";
}

function interpolate(template: string, vars?: TranslateVars) {
  if (!vars) return template;
  return template.replace(/\{(\w+)\}/g, (match, name: string) => (Object.prototype.hasOwnProperty.call(vars, name) ? String(vars[name]) : match));
}

function pluralTemplate(forms: PluralForms, count: number, language: Language) {
  const category = new Intl.PluralRules(language).select(count);
  return forms[category] ?? forms.other;
}

export function translate(language: Language, key: MessageKey, vars?: TranslateVars): string {
  const message: Message = localeRegistry[language]?.messages[key] ?? localeRegistry.en.messages[key];
  return interpolate(typeof message === "string" ? message : message.other, vars);
}

export function translatePlural(language: Language, key: MessageKey, count: number, vars?: TranslateVars): string {
  const message: Message = localeRegistry[language]?.messages[key] ?? localeRegistry.en.messages[key];
  const template = typeof message === "string" ? message : pluralTemplate(message, count, language);
  return interpolate(template, { count: new Intl.NumberFormat(language).format(count), ...vars });
}

/** Translates with the language currently used by the UI. */
export const t: Translate = (key, vars) => translate(activeLanguage, key, vars);

/** Translates a pluralized message with the language currently used by the UI. */
export const tp: TranslatePlural = (key, count, vars) => translatePlural(activeLanguage, key, count, vars);

export function formatCount(count: number) {
  return new Intl.NumberFormat(activeLanguage).format(count);
}

export { hasMessageKey, isSupportedLanguage, localeRegistry, supportedLanguages, type Language, type MessageKey };
