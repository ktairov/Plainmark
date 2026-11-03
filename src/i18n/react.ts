import { useLayoutEffect, useSyncExternalStore } from "react";
import { getActiveLanguage, setActiveLanguage, subscribeActiveLanguage, type Language } from "./index";

/** Keeps the imperative translation helpers aligned without mutating them during render. */
export function useAppLanguage(language: Language) {
  const activeLanguage = useSyncExternalStore(subscribeActiveLanguage, getActiveLanguage, getActiveLanguage);
  useLayoutEffect(() => setActiveLanguage(language), [language]);
  return activeLanguage;
}
