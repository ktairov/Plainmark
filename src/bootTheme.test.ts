import { describe, expect, it } from "vitest";
import { bootThemeScript, resolveBootTheme, type BootThemePalette } from "./bootTheme";
import { defaultAppSettings } from "./settings";
import { themes } from "./themes";

const pages: BootThemePalette = Object.fromEntries(
  Object.entries(themes).map(([id, theme]) => [id, { light: theme.light.page, dark: theme.dark.page }])
);
const defaults = { light: defaultAppSettings.dayTheme, dark: defaultAppSettings.nightTheme };

describe("boot theme", () => {
  it("paints the stored light theme", () => {
    expect(resolveBootTheme(JSON.stringify({ colorMode: "light", dayTheme: "catppuccin" }), true, pages, defaults)).toEqual({
      colorMode: "light",
      theme: "catppuccin",
      page: themes.catppuccin.light.page,
    });
  });

  it("follows the system appearance in system mode", () => {
    expect(resolveBootTheme(JSON.stringify({ colorMode: "system", nightTheme: "nord" }), true, pages, defaults)).toEqual({
      colorMode: "dark",
      theme: "nord",
      page: themes.nord.dark.page,
    });
    expect(resolveBootTheme(JSON.stringify({ colorMode: "system", nightTheme: "nord" }), false, pages, defaults).colorMode).toBe("light");
  });

  it("falls back to defaults for broken or unknown settings", () => {
    expect(resolveBootTheme(null, false, pages, defaults)).toEqual({
      colorMode: "light",
      theme: defaults.light,
      page: themes[defaults.light].light.page,
    });
    expect(resolveBootTheme("not json", true, pages, defaults)).toEqual({
      colorMode: "dark",
      theme: defaults.dark,
      page: themes[defaults.dark].dark.page,
    });
    expect(resolveBootTheme(JSON.stringify({ colorMode: "dark", nightTheme: "missing" }), false, pages, defaults).theme).toBe(defaults.dark);
  });

  it("sets the pre-paint background and color scheme from the generated script", () => {
    const run = (stored: string | null, prefersDark: boolean) => {
      const root = { dataset: {} as Record<string, string>, style: {} as Record<string, unknown> };
      const localStorage = { getItem: () => stored };
      const window = { matchMedia: () => ({ matches: prefersDark }) };
      const document = { documentElement: root };
      new Function("localStorage", "window", "document", bootThemeScript(pages, defaults))(localStorage, window, document);
      return root;
    };

    const dark = run(JSON.stringify({ colorMode: "dark", nightTheme: "mariana", dayTheme: "ayu" }), false);
    expect(dark.dataset).toEqual({ colorMode: "dark", theme: "mariana" });
    expect(dark.style.colorScheme).toBe("dark");
    expect(dark.style.backgroundColor).toBe(themes.mariana.dark.page);

    const light = run(JSON.stringify({ colorMode: "light", dayTheme: "ayu" }), true);
    expect(light.dataset).toEqual({ colorMode: "light", theme: "ayu" });
    expect(light.style.backgroundColor).toBe(themes.ayu.light.page);
  });
});
