import { isSupportedLanguage, type LanguageSetting } from "./i18n";
import { themeOptions, type ThemeId } from "./themes";

export type ColorMode = "system" | "light" | "dark";
export { themeOptions, type ThemeId } from "./themes";
export type FontId = "system" | "humanist" | "serif" | "mono";

export type AppSettings = {
  language: LanguageSetting;
  colorMode: ColorMode;
  dayTheme: ThemeId;
  nightTheme: ThemeId;
  uiFont: FontId;
  readingFont: FontId;
  treeFontSize: number;
  contentFontSize: number;
  previewWidth: number;
};

export const defaultAppSettings: AppSettings = {
  language: "system",
  colorMode: "system",
  dayTheme: "ayu",
  nightTheme: "mariana",
  uiFont: "system",
  readingFont: "system",
  treeFontSize: 13,
  contentFontSize: 16,
  previewWidth: 860,
};

export const fontOptions: Array<{ id: FontId; label: string; stack: string }> = [
  { id: "system", label: "System", stack: "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif" },
  { id: "humanist", label: "Humanist", stack: "'Avenir Next', Avenir, 'Segoe UI', sans-serif" },
  { id: "serif", label: "Serif", stack: "Charter, 'Iowan Old Style', Georgia, serif" },
  { id: "mono", label: "Monospace", stack: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" },
];

const modes = new Set<ColorMode>(["system", "light", "dark"]);
const themes = new Set<ThemeId>(themeOptions.map(option => option.id));
const fonts = new Set<FontId>(fontOptions.map(option => option.id));
const clamp = (value: unknown, minimum: number, maximum: number, fallback: number) => {
  const number = Number(value);
  return Number.isFinite(number) ? Math.min(maximum, Math.max(minimum, Math.round(number))) : fallback;
};

export function parseAppSettings(raw: string | null): AppSettings {
  if (!raw) return defaultAppSettings;
  try {
    const value = JSON.parse(raw) as Partial<AppSettings>;
    return {
      language: value.language === "system" || isSupportedLanguage(value.language) ? value.language : defaultAppSettings.language,
      colorMode: modes.has(value.colorMode as ColorMode) ? value.colorMode as ColorMode : defaultAppSettings.colorMode,
      dayTheme: themes.has(value.dayTheme as ThemeId) ? value.dayTheme as ThemeId : defaultAppSettings.dayTheme,
      nightTheme: themes.has(value.nightTheme as ThemeId) ? value.nightTheme as ThemeId : defaultAppSettings.nightTheme,
      uiFont: fonts.has(value.uiFont as FontId) ? value.uiFont as FontId : defaultAppSettings.uiFont,
      readingFont: fonts.has(value.readingFont as FontId) ? value.readingFont as FontId : defaultAppSettings.readingFont,
      treeFontSize: clamp(value.treeFontSize, 13, 21, defaultAppSettings.treeFontSize),
      contentFontSize: clamp(value.contentFontSize, 13, 26, defaultAppSettings.contentFontSize),
      previewWidth: clamp(value.previewWidth, 560, 1400, defaultAppSettings.previewWidth),
    };
  } catch {
    return defaultAppSettings;
  }
}

export function resolvedColorMode(mode: ColorMode, systemDark: boolean): "light" | "dark" {
  return mode === "system" ? systemDark ? "dark" : "light" : mode;
}

export function resolvedTheme(settings: AppSettings, colorMode: "light" | "dark"): ThemeId {
  return colorMode === "dark" ? settings.nightTheme : settings.dayTheme;
}

export function toggledColorMode(colorMode: "light" | "dark"): "light" | "dark" {
  return colorMode === "dark" ? "light" : "dark";
}

export function fontStack(id: FontId) {
  return fontOptions.find(option => option.id === id)?.stack || fontOptions[0].stack;
}
