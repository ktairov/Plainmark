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
	if a.win != nil {
		if a.nativeLocale != a.preferences.Language {
			tag := a.preferences.Language
			if tag == "system" {
				tag = ""
			}
			mygo.App.SetLocale(tag)
			a.nativeLocale = a.preferences.Language
		}
		if a.nativeColor != a.preferences.ColorMode {
			source := mygo.ThemeSystem
			if a.preferences.ColorMode == "dark" {
				source = mygo.ThemeDark
			}
			if a.preferences.ColorMode == "light" {
				source = mygo.ThemeLight
			}
			mygo.Theme.SetSource(source)
			a.nativeColor = a.preferences.ColorMode
		}
	}
	dark := a.preferences.Dark(c.Theme().Dark)
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
	if a.busy {
		return
	}
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

type choice struct{ id, label string }

func (a *App) selectSetting(c *ui.Context, label string, value *string, options []choice) {
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, a.t(label)).Width(200)
		var labels []string
		selected := ""
		for _, o := range options {
			display := a.t(o.label)
			labels = append(labels, display)
			if o.id == *value {
				selected = display
			}
		}
		c.Key(label)
		ui.Select(c, &selected, labels).Label(a.t(label)).Grow(1)
		for _, o := range options {
			if a.t(o.label) == selected {
				*value = o.id
				break
			}
		}
	})
}
func (a *App) sizeSetting(c *ui.Context, label string, value *int, lo, hi int) {
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, a.t(label)).Width(200)
		n := float64(*value)
		c.Key(label)
		ui.NumberInput(c, &n, float64(lo), float64(hi), 1).Label(a.t(label)).Grow(1)
		*value = int(n)
	})
}
func (a *App) preferencesModal(c *ui.Context) {
	if !a.settingsOpen {
		return
	}
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Width(600).Padding(24).Radius(12).Background(c.Theme().Background)
		ui.Column(c).Width(540).Gap(14).Children(func() {
			ui.Text(c, a.t("Settings")).FontSize(20).Bold()
			ui.Scroll(c).MaxHeight(500).FillWidth().Children(func() {
				ui.Column(c).Gap(12).FillWidth().Children(func() {
					a.selectSetting(c, "Language", &a.preferences.Language, []choice{{"system", "System"}, {"en", "English"}, {"ru", "Русский"}})
					a.selectSetting(c, "Color mode", &a.preferences.ColorMode, []choice{{"system", "System"}, {"light", "Light"}, {"dark", "Dark"}})
					var themes []choice
					for _, id := range settings.ThemeOrder {
						themes = append(themes, choice{id, settings.Themes[id].Label})
					}
					a.selectSetting(c, "Day theme", &a.preferences.DayTheme, themes)
					a.selectSetting(c, "Night theme", &a.preferences.NightTheme, themes)
					fonts := []choice{{"system", "System"}, {"humanist", "Humanist"}, {"serif", "Serif"}, {"mono", "Monospace"}}
					a.selectSetting(c, "Interface font", &a.preferences.UIFont, fonts)
					a.selectSetting(c, "Reading font", &a.preferences.ReadingFont, fonts)
					a.sizeSetting(c, "Interface text size", &a.preferences.TreeFontSize, 13, 21)
					a.sizeSetting(c, "Document text size", &a.preferences.ContentFontSize, 13, 26)
					a.sizeSetting(c, "Preview text width", &a.preferences.PreviewWidth, 560, 1400)
				})
			})
			a.preferences.Normalize()
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, a.t("Reset defaults")).Clicked() {
					a.preferences = settings.Defaults()
				}
				ui.Box(c).Grow(1)
				if ui.Button(c, a.t("Cancel")).Clicked() {
					a.preferences = a.beforePreferences
					a.settingsOpen = false
				}
				if ui.PrimaryButton(c, a.t("Apply")).Key("apply-settings").Clicked() {
					a.settingsOpen = false
					a.status = "Settings applied"
				}
			})
		})
	})
	if !open {
		a.preferences = a.beforePreferences
		a.settingsOpen = false
	}
}

func (a *App) positionText(line, column int) string {
	return a.t("Ln {line}, Col {column}", "line", strconv.Itoa(line), "column", strconv.Itoa(column))
}
