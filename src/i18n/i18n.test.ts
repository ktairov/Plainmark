import { afterEach, describe, expect, it } from "vitest";
import { formatCount, localeRegistry, resolveLanguage, setActiveLanguage, supportedLanguages, t, tp, translate, translatePlural } from "./index";
import { en } from "./locales";
import { defaultAppSettings, parseAppSettings } from "../settings";
import { errorMessage } from "../model";
import { menuLabels } from "./menu";
import menuLabelContract from "./menu-contract.json";

afterEach(() => setActiveLanguage("en"));

describe("translations", () => {
  const placeholders = (value: string) => [...value.matchAll(/\{(\w+)\}/g)].map(match => match[1]).sort();

  it("keeps every locale in sync with the English key set", () => {
    for (const language of supportedLanguages) {
      expect(Object.keys(localeRegistry[language].messages).sort()).toEqual(Object.keys(en).sort());
      for (const [key, value] of Object.entries(localeRegistry[language].messages)) {
        const source = en[key as keyof typeof en];
        expect(typeof value, `${language}:${key}:message kind`).toBe(typeof source);
        for (const [category, template] of typeof value === "string" ? [["other", value]] : Object.entries(value)) {
          if (typeof template !== "string") continue;
          expect(template.length, `${language}:${key}`).toBeGreaterThan(0);
          const sourceTemplate = typeof source === "string" ? source : source[category as keyof typeof source] ?? source.other;
          expect(placeholders(template), `${language}:${key}:placeholders`).toEqual(placeholders(sourceTemplate));
        }
      }
    }
  });

  it("interpolates variables and leaves unknown placeholders intact", () => {
    expect(translate("en", "app.status.opened", { path: "notes/a.md" })).toBe("Opened notes/a.md");
    expect(translate("ru", "app.status.opened", { path: "notes/a.md" })).toBe("Открыто: notes/a.md");
    expect(translate("en", "header.modeShortcut", { label: "Preview", number: 1 })).toBe("Preview (⌥1)");
  });

  it("uses Russian plural categories", () => {
    expect(translatePlural("ru", "watch.newNotes", 1)).toContain("новая заметка");
    expect(translatePlural("ru", "watch.newNotes", 2)).toContain("новые заметки");
    expect(translatePlural("ru", "watch.newNotes", 5)).toContain("новых заметок");
    expect(translatePlural("en", "watch.newNotes", 2)).toBe("2 new notes detected");
    expect(translatePlural("en", "trash.folder", 1)).toBe("This folder contains 1 file.");
  });

  it("follows the active language for t() and formatCount()", () => {
    setActiveLanguage("ru");
    expect(t("common.cancel")).toBe("Отмена");
    expect(tp("trash.folder", 3)).toContain("3 файла");
    expect(formatCount(1234)).toMatch(/1\s234/);
    setActiveLanguage("en");
    expect(t("common.cancel")).toBe("Cancel");
    expect(formatCount(1234)).toBe("1,234");
  });

  it("exposes a locale registry with native names", () => {
    expect(supportedLanguages).toEqual(["en", "ru"]);
    expect(localeRegistry.en.nativeName).toBe("English");
    expect(localeRegistry.ru.nativeName).toBe("Русский");
  });

  it("resolves the language setting against system locales", () => {
    expect(resolveLanguage("system", ["ru-RU"])).toBe("ru");
    expect(resolveLanguage("system", ["ru-RU", "en-US"])).toBe("ru");
    expect(resolveLanguage("system", ["en-UA", "ru-UA"])).toBe("en");
    expect(resolveLanguage("system", ["de-DE", "ru-RU"])).toBe("ru");
    expect(resolveLanguage("system", ["de-DE"])).toBe("en");
    expect(resolveLanguage("en", ["ru-RU"])).toBe("en");
    expect(resolveLanguage("ru", ["en-US"])).toBe("ru");
    expect(resolveLanguage(undefined, [])).toBe("en");
  });

  it("localizes backend error codes with their details", () => {
    setActiveLanguage("ru");
    expect(errorMessage({ code: "itemExists", message: "An item with this name already exists." })).toBe("Элемент с таким именем уже существует.");
    expect(errorMessage({ code: "conflict", message: "changed", details: { expected: "1", found: "2" } })).toBe("Файл изменён на диске (ожидалось 1, получено 2).");
    expect(errorMessage({ code: "unknownCode", message: "raw fallback" })).toBe("raw fallback");
    expect(errorMessage("plain failure")).toBe("plain failure");
  });

  it("builds native menu labels for the current language", () => {
    expect(Object.values(menuLabelContract).every(key => Object.hasOwn(en, key))).toBe(true);
    const russian = menuLabels("ru");
    expect(Object.keys(russian)).toHaveLength(39);
    expect(russian.save).toBe("Сохранить");
    expect(russian.hideSidebar).toBe("Скрыть боковую панель");
    expect(russian.noRecentFolders).toBe("Нет недавних папок");
    expect(menuLabels("en").save).toBe("Save");
  });

  it("validates the stored language setting", () => {
    expect(defaultAppSettings.language).toBe("system");
    expect(parseAppSettings(null).language).toBe("system");
    expect(parseAppSettings('{"language":"ru"}').language).toBe("ru");
    expect(parseAppSettings('{"language":"en"}').language).toBe("en");
    expect(parseAppSettings('{"language":"fr"}').language).toBe("system");
  });
});
