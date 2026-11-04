// Regenerate shared palettes and translations from the preserved Tauri sources.
// Node is needed only for this maintenance command, never by the Go app.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { en } from "../../src/i18n/locales/en.ts";
import { ru } from "../../src/i18n/locales/ru.ts";
import { themes } from "../../src/themes.ts";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const write = (p, value) => fs.writeFileSync(path.join(root, p), JSON.stringify(value, null, 2) + "\n");
write("internal/settings/themes.json", themes);
const messages = { en: {}, ru: {} }, aliases = {};
for (const [key, text] of Object.entries(en)) {
  if (typeof text !== "string") continue;
  messages.en[key] = text;
  messages.ru[key] = ru[key];
  aliases[text] = key;
}
const aliasKeys = {
  "Open folder": "menu.openFolder", "Open file": "menu.openFile", "Open folder…": "menu.openFolder", "Open file…": "menu.openFile", "New note": "menu.newNote", "New folder": "menu.newFolder", "New note…": "menu.newNote", "New folder…": "menu.newFolder", "Edit": "menu.editor", "Split": "menu.split",
  "Recent folders": "menu.recentFolders", "Close tab": "tab.close", "Close other tabs": "menu.closeOtherTabs", "Close all tabs": "menu.closeAllTabs", "Previous tab": "menu.previousTab", "Next tab": "menu.nextTab",
  "Copy relative path": "menu.copyRelativePath", "Copy full path": "menu.copyPath", "Reveal in Finder": "menu.reveal", "Search notes": "menu.searchNotes", "Reload": "common.loadFromDisk",
  "Name": "dialog.name", "Destination folder": "dialog.destination", "Settings": "settings.title", "View mode": "settings.aria.sections",
};
Object.assign(aliases, aliasKeys);
const extra = {
  "View mode": "Режим просмотра", "Markdown editor": "Редактор Markdown", "Open": "Открыть",
  "Find": "Найти", "Wrap": "Перенос строк", "More": "Ещё", "Pinned": "Закреплено", "Toggle sidebar": "Показать/скрыть боковую панель",
  "Open a note from the sidebar": "Откройте заметку в боковой панели", "Open a Markdown folder or file to begin": "Откройте папку или файл Markdown",
  "Save changed notes before continuing?": "Сохранить изменённые заметки перед продолжением?", "Save all": "Сохранить всё", "Move to Trash?": "Переместить в корзину?", "{name} — {count} nested files": "{name} — вложенных файлов: {count}",
  "File changed on disk. Your draft is kept.": "Файл изменён на диске. Черновик сохранён в памяти.", "File was removed. Your draft is kept.": "Файл удалён с диска. Черновик сохранён в памяти.",
  "Working…": "Обработка…", "Password cannot be empty": "Пароль не может быть пустым", "Passwords do not match": "Пароли не совпадают", "Selection changed. Select it again.": "Выделение изменилось. Выделите текст заново.",
  "The password cannot be recovered if you forget it.": "Если вы забудете пароль, восстановить его будет невозможно.", "Switch to Edit or Split and select text first": "Перейдите в редактор или разделённый вид и выделите текст", "Select text to encrypt": "Выделите текст для шифрования",
  "Only Markdown files can be opened": "Можно открывать только файлы Markdown", "Workspace refreshed": "Папка обновлена", "Updated {path}": "Обновлено: {path}", "Reloaded {path}": "Загружено с диска: {path}", "Saved copy: {path}": "Копия сохранена: {path}", "Moved to Trash: {path}": "Перемещено в корзину: {path}",
  "System": "Системный", "Light": "Светлая", "Dark": "Тёмная", "Language": "Язык", "Color mode": "Цветовой режим", "Day theme": "Дневная тема", "Night theme": "Ночная тема", "Interface font": "Шрифт интерфейса", "Reading font": "Шрифт чтения", "Interface text size": "Размер текста интерфейса", "Document text size": "Размер текста документа", "Preview text width": "Ширина текста при чтении",
  "Humanist": "Гуманистический", "Serif": "С засечками", "Monospace": "Моноширинный", "Appearance": "Оформление", "Theme": "Тема", "Reset defaults": "Сбросить настройки", "Ln {line}, Col {column}": "Стр {line}, Стлб {column}",
  "Toggle color mode": "Переключить цветовой режим", "Settings applied": "Настройки применены", "Tab {title}": "Вкладка {title}", "Close {title}": "Закрыть {title}", "Name is required": "Введите имя",
};
for (const [text, translated] of Object.entries(extra)) {
  const key = "ui." + text.toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_|_$/g, "");
  messages.en[key] = text; messages.ru[key] = translated; aliases[text] = key;
}
// Include the toolkit source messages for fallback, and translate the controls
// used by this application through MyGo's supported namespace override.
const moduleRoot = process.env.MYGO_SOURCE || path.join(process.env.HOME, "go/pkg/mod/github.com/egoist/mygo@v0.7.4");
const nativeEN = JSON.parse(fs.readFileSync(path.join(moduleRoot, "internal/l10n/messages/en.json"), "utf8"));
const nativeRU = JSON.parse(fs.readFileSync(path.join(root, "internal/locales/mygo-ru.json"), "utf8"));
for (const [key, text] of Object.entries(nativeEN)) {
  messages.en["mygo." + key] = text;
  messages.ru["mygo." + key] = nativeRU[key] || text;
}
write("internal/locales/messages/en.json", messages.en);
write("internal/locales/messages/ru.json", messages.ru);
write("internal/locales/aliases.json", aliases);
