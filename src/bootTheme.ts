export type BootThemePage = { light: string; dark: string };
export type BootThemePalette = Record<string, BootThemePage>;
export type BootThemeDefaults = { light: string; dark: string };
export type BootTheme = { colorMode: "light" | "dark"; theme: string; page: string };

/**
 * Resolves the theme that should be painted before the app bundle loads.
 *
 * The function is inlined into index.html via `toString()`, so it must stay
 * self-contained (no imports, no module-scope constants).
 */
export function resolveBootTheme(
  storageValue: string | null,
  systemPrefersDark: boolean,
  pages: BootThemePalette,
  defaults: BootThemeDefaults
): BootTheme {
  let settings: Record<string, unknown> = {};
  try {
    const parsed: unknown = storageValue ? JSON.parse(storageValue) : {};
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) settings = parsed as Record<string, unknown>;
  } catch {
    settings = {};
  }
  const mode = settings.colorMode === "light" || settings.colorMode === "dark" ? settings.colorMode : "system";
  const dark = mode === "dark" || (mode === "system" && systemPrefersDark);
  const key: "light" | "dark" = dark ? "dark" : "light";
  const storedTheme = dark ? settings.nightTheme : settings.dayTheme;
  const theme = typeof storedTheme === "string" && Object.prototype.hasOwnProperty.call(pages, storedTheme) ? storedTheme : defaults[key];
  return { colorMode: key, theme, page: pages[theme][key] };
}

/** Inline script injected into index.html by the Vite theme bootstrap plugin. */
export function bootThemeScript(pages: BootThemePalette, defaults: BootThemeDefaults) {
  const config = JSON.stringify({ pages, defaults });
  return `(function(){try{
var config=${config};
var resolveBootTheme=${resolveBootTheme.toString()};
var boot=resolveBootTheme(localStorage.getItem("plainmark-app-settings"),window.matchMedia("(prefers-color-scheme: dark)").matches,config.pages,config.defaults);
var root=document.documentElement;
root.dataset.colorMode=boot.colorMode;
root.dataset.theme=boot.theme;
root.style.colorScheme=boot.colorMode;
root.style.backgroundColor=boot.page;
}catch(error){}})();`;
}
