package desktop

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"plainmark/internal/locales"
	"plainmark/internal/settings"
)

func (a *App) language() string {
	if a.preferences.Language != "system" {
		return a.preferences.Language
	}
	return a.resolvedLanguage
}
func (a *App) t(text string, args ...string) string { return locales.Text(a.language(), text, args...) }
func (a *App) displayStatus() string {
	for _, p := range []struct{ prefix, template string }{{"Opened ", "Opened {path}"}, {"Saved copy: ", "Saved copy: {path}"}, {"Saved ", "Saved {path}"}, {"Updated ", "Updated {path}"}, {"Reloaded ", "Reloaded {path}"}, {"Moved to Trash: ", "Moved to Trash: {path}"}} {
		if strings.HasPrefix(a.status, p.prefix) {
			return a.t(p.template, "path", strings.TrimPrefix(a.status, p.prefix))
		}
	}
	return a.t(a.status)
}
func (a *App) applyStyle(c *ui.Context) {
	a.resolvedLanguage = c.Locale()
	colorMode := a.preferences.ColorMode
	if a.settingsOpen && a.settingsPreviewMode != "" {
		colorMode = a.settingsPreviewMode
	}
	if a.win != nil {
		if a.nativeLocale != a.preferences.Language {
			tag := a.preferences.Language
			if tag == "system" {
				tag = ""
			}
			mygo.App.SetLocale(tag)
			a.nativeLocale = a.preferences.Language
		}
		if a.nativeColor != colorMode {
			source := mygo.ThemeSystem
			if colorMode == "dark" {
				source = mygo.ThemeDark
			}
			if colorMode == "light" {
				source = mygo.ThemeLight
			}
			mygo.Theme.SetSource(source)
			a.nativeColor = colorMode
		}
	}
	preview := a.preferences
	preview.ColorMode = colorMode
	dark := preview.Dark(c.Theme().Dark)
	a.appearanceDark = dark
	t := ui.LightTheme()
	id := a.preferences.DayTheme
	if dark {
		t = ui.DarkTheme()
		id = a.preferences.NightTheme
	}
	definition := settings.Themes[id]
	p := definition.Light
	if dark {
		p = definition.Dark
	}
	t.Background = ui.Hex(p.Page)
	t.Surface = ui.Hex(p.Control)
	t.SurfaceHover = ui.Hex(p.Selected)
	t.SurfacePressed = ui.Hex(p.Selection)
	t.Text = ui.Hex(p.Text)
	t.TextMuted = ui.Hex(p.TextSecondary)
	t.Border = ui.Hex(p.Border)
	t.Accent = ui.Hex(p.Accent)
	t.AccentText = ui.Hex(p.AccentContrast)
	t.AccentHover = t.Accent.Mix(t.Text, .1)
	t.AccentPressed = t.Accent.Mix(t.Text, .2)
	t.Selection = ui.Hex(p.Selection)
	t.Focus = t.Accent.Alpha(.6)
	t.Danger = ui.Hex(p.Danger)
	t.Warning = ui.Hex(p.Warning)
	t.Success = ui.Hex(p.Success)
	t.Font = settings.Font(a.preferences.UIFont)
	if c.Preferences().HighContrast {
		t.Border = t.Border.Mix(t.Text, .4)
		t.TextMuted = t.TextMuted.Mix(t.Text, .3)
	}
	c.SetTheme(t)
}
func (a *App) openSettings() {
	if a.busy || a.settingsOpen {
		return
	}
	a.settingsSection = "appearance"
	a.settingsPreviewMode = ""
	a.beforePreferences = a.preferences
	a.settingsOpen = true
}
func (a *App) toggleColor() {
	dark := a.preferences.ColorMode == "dark"
	if a.preferences.ColorMode == "system" && a.win != nil {
		dark = mygo.Theme.IsDark()
	}
	if dark {
		a.preferences.ColorMode = "light"
	} else {
		a.preferences.ColorMode = "dark"
	}
}

func (a *App) positionText(line, column int) string {
	return a.t("Ln {line}, Col {column}", "line", strconv.Itoa(line), "column", strconv.Itoa(column))
}
