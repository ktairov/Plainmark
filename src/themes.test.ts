import { describe, expect, it } from "vitest";
import { themeOptions, themes, themeVariables } from "./themes";

describe("themes", () => {
  it("exposes complete light and dark palettes for every option", () => {
    for (const option of themeOptions) {
      expect(Object.keys(themes[option.id].light)).toHaveLength(17);
      expect(Object.keys(themes[option.id].dark)).toHaveLength(17);
      expect(option.day).toEqual([
        themes[option.id].light.page,
        themes[option.id].light.control,
        themes[option.id].light.text,
        themes[option.id].light.accent,
      ]);
    }
  });

  it("maps typed palette fields to the CSS theme contract", () => {
    const variables = themeVariables("ayu", "dark") as Record<string, string>;
    expect(variables["--theme-page"]).toBe(themes.ayu.dark.page);
    expect(variables["--theme-text-secondary"]).toBe(themes.ayu.dark.textSecondary);
    expect(variables["--theme-accent-contrast"]).toBe(themes.ayu.dark.accentContrast);
  });
});
