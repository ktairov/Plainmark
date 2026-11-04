package desktop

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"plainmark/internal/icons"
	"plainmark/internal/locales"
	"plainmark/internal/settings"
)

func (a *App) settingsText(key string) string {
	return locales.Catalog.Format(a.language(), "settings."+key, nil)
}

func (a *App) closeSettings(apply bool) {
	if !apply {
		a.preferences = a.beforePreferences
	} else {
		a.status = "Settings applied"
		a.windowColorOverride = ""
	}
	a.settingsPreviewMode = ""
	a.settingsOpen = false
}

func (a *App) preferencesModal(c *ui.Context) {
	if !a.settingsOpen {
		return
	}
	w, h := c.Size()
	w, h = min(float32(820), w-64), min(float32(650), h-64)
	s := a.visual(c)
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.Hex("#1e2523aa"))
		panel.Label(a.t("Settings")).Width(w).Height(h).Radius(12).Border(1, s.softBorder).Background(s.page).Shadow(0, 22, 80, 0, ui.RGBA(0, 0, 0, .4)).Clip()
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Height(52).Shrink(0).Padding(0, 20).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(s.softBorder).Children(func() {
				ui.Text(c, a.t("Settings")).FontSize(17).Bold()
			})
			ui.Row(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
				ui.Column(c).Width(190).Shrink(0).FillHeight().Padding(14).Gap(5).Background(s.page).BorderWidth(0, 1, 0, 0).BorderColor(s.softBorder).Children(func() {
					for _, item := range []struct{ id, icon string }{{"appearance", "type"}, {"theme", "palette"}, {"language", "languages"}} {
						label := a.settingsText("section." + item.id)
						button := ui.ButtonBase(c).Key("settings-section-"+item.id).Label(label).FillWidth().Height(38).Padding(0, 10).Gap(8).Justify(ui.Start).Radius(7).Selected(a.settingsSection == item.id)
						if a.settingsSection == item.id {
							button.Background(s.accentSoft).TextColor(s.accent)
						} else if button.Hovered() {
							button.Background(s.hover)
						}
						button.Children(func() { ui.Icon(c, icons.Get(item.icon)).Size(17, 17); ui.Text(c, label).FontSize(13) })
						if button.Clicked() {
							a.settingsSection = item.id
						}
					}
				})
				c.Key("settings-content-" + a.settingsSection)
				ui.Scroll(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
					ui.Column(c).FillWidth().Padding(24, 28, 32, 28).Gap(18).Children(func() {
						ui.Column(c).FillWidth().Gap(6).Padding(0, 0, 16, 0).BorderWidth(0, 0, 1, 0).BorderColor(s.softBorder).Children(func() {
							ui.Text(c, a.settingsText("section."+a.settingsSection)).FontSize(18.5).Bold()
							ui.Text(c, a.settingsText(a.settingsSection+".description")).FontSize(13).LineHeight(1.45).TextColor(s.muted)
						})
						switch a.settingsSection {
						case "theme":
							a.themeSettings(c, false)
							a.themeSettings(c, true)
						case "language":
							a.languageSettings(c)
						default:
							a.appearanceSettings(c)
						}
					})
				})
			})
			a.preferences.Normalize()
			ui.Row(c).FillWidth().Height(64).Shrink(0).Padding(0, 20).Gap(8).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(s.softBorder).Children(func() {
				if a.actionButton(c, "Reset defaults", "reset-settings", "").Clicked() {
					a.preferences = settings.Defaults()
					a.settingsPreviewMode = ""
				}
				ui.Box(c).Grow(1)
				if a.actionButton(c, "Apply", "apply-settings", "primary").Clicked() {
					a.closeSettings(true)
				}
				if a.actionButton(c, "Cancel", "cancel-settings", "").Clicked() {
					a.closeSettings(false)
				}
			})
		})
	})
	if !open {
		a.closeSettings(false)
	}
}

func (a *App) appearanceSettings(c *ui.Context) {
	s := a.visual(c)
	ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Padding(16, 0).BorderWidth(0, 0, 1, 0).BorderColor(s.softBorder).Children(func() {
		ui.Text(c, a.t("Color mode")).FontSize(14).Bold()
		ui.Row(c).Padding(2).Gap(0).Radius(7).Border(1, s.border).Background(s.page).Role(ui.RoleRadioGroup).Label(a.t("Color mode")).Children(func() {
			for _, option := range []struct{ id, label string }{{"system", a.settingsText("colorMode.system")}, {"light", a.settingsText("colorMode.light")}, {"dark", a.settingsText("colorMode.dark")}} {
				b := ui.ButtonBase(c).Key("settings-color-"+option.id).Role(ui.RoleRadio).Label(option.label).Height(30).Padding(6, 10).Radius(5).FontSize(14).Selected(a.preferences.ColorMode == option.id)
				b.Background(ui.Transparent).TextColor(s.muted)
				if a.preferences.ColorMode == option.id {
					b.Background(s.control).TextColor(s.accent).Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, .13))
				} else if b.Hovered() {
					b.Background(s.hover)
				}
				b.Children(func() { ui.Text(c, option.label).SingleLine() })
				if b.Clicked() {
					a.preferences.ColorMode = option.id
					a.settingsPreviewMode = ""
				}
			}
		})
	})
	a.fontSettings(c, false)
	a.fontSettings(c, true)
	ui.Column(c).FillWidth().Gap(12).Padding(16, 0).BorderWidth(0, 0, 1, 0).BorderColor(s.softBorder).Children(func() {
		ui.Row(c).FillWidth().Children(func() {
			ui.Text(c, a.t("Preview text width")).Bold().FontSize(14)
			ui.Box(c).Grow(1)
			ui.Text(c, strconv.Itoa(a.preferences.PreviewWidth)+" px").TextColor(s.muted).FontSize(12)
		})
		width := float64(a.preferences.PreviewWidth)
		c.Key("settings-preview-width")
		ui.Slider(c, &width, 560, 1400).Step(20).Label(a.t("Preview text width")).FillWidth()
		a.preferences.PreviewWidth = int(width)
	})
}
func (a *App) fontSettings(c *ui.Context, reading bool) {
	s := a.visual(c)
	label, hint, sizeLabel := "Interface font", "uiFontHint", "Interface text size"
	font, size, upper := &a.preferences.UIFont, &a.preferences.TreeFontSize, 21
	if reading {
		label, hint, sizeLabel = "Reading font", "readingFontHint", "Document text size"
		font, size, upper = &a.preferences.ReadingFont, &a.preferences.ContentFontSize, 26
	}
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(16).Padding(16, 0).BorderWidth(0, 0, 1, 0).BorderColor(s.softBorder).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, a.t(label)).Bold().FontSize(14)
			ui.Text(c, a.settingsText(hint)).FontSize(12).TextColor(s.muted)
		})
		ui.Row(c).Gap(8).Shrink(0).Children(func() {
			fonts := []selectOption{{id: "system", label: a.t("System")}, {id: "humanist", label: a.t("Humanist")}, {id: "serif", label: a.t("Serif")}, {id: "mono", label: a.t("Monospace")}}
			a.selectControl(c, "settings-"+label, label, *font, fonts, func(value string) { *font = value }).Width(145)
			var sizes []selectOption
			for n := 13; n <= upper; n++ {
				value := strconv.Itoa(n)
				sizes = append(sizes, selectOption{id: value, label: value + " px"})
			}
			a.selectControl(c, "settings-"+sizeLabel, sizeLabel, strconv.Itoa(*size), sizes, func(value string) { *size, _ = strconv.Atoi(value) }).Width(82)
		})
	})
}

func (a *App) themeSettings(c *ui.Context, dark bool) {
	label, hint, mode := "Day theme", "theme.dayHint", "light"
	value := &a.preferences.DayTheme
	if dark {
		label, hint, mode, value = "Night theme", "theme.nightHint", "dark", &a.preferences.NightTheme
	}
	s := a.visual(c)
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		ui.Text(c, a.t(label)).Bold().FontSize(14)
		ui.Text(c, a.settingsText(hint)).FontSize(12).TextColor(s.muted)
		ui.Grid(c).Columns(2).Gap(13).FillWidth().Children(func() {
			for _, id := range settings.ThemeOrder {
				definition := settings.Themes[id]
				palette := definition.Light
				if dark {
					palette = definition.Dark
				}
				button := ui.ButtonBase(c).Key("theme-"+mode+"-"+id).Label(a.t(label)+": "+definition.Label).Height(58).Padding(12).Gap(10).Radius(9).Background(s.control).Border(1, s.softBorder).Selected(*value == id)
				button.Background(s.control).TextColor(s.text)
				if *value == id || button.Hovered() {
					button.Border(1, s.accent)
					if *value == id {
						button.Shadow(0, 0, 0, 1, s.accent)
					}
				}
				button.Children(func() {
					ui.Row(c).Gap(0).Radius(5).Clip().Children(func() {
						for _, color := range []string{palette.Page, palette.Control, palette.Text, palette.Accent} {
							ui.Box(c).Size(18, 28).Background(ui.Hex(color))
						}
					})
					ui.Text(c, definition.Label).FontSize(13.5).Bold().Grow(1)
					if *value == id {
						ui.Icon(c, icons.Get("check")).Size(14, 14).TextColor(s.accent)
					}
				})
				if button.Clicked() {
					*value = id
					a.settingsPreviewMode = mode
				}
			}
		})
	})
}

func (a *App) languageSettings(c *ui.Context) {
	s := a.visual(c)
	ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Padding(16, 0).Children(func() {
		ui.Text(c, a.settingsText("section.language")).FontSize(14).Bold()
		options := []selectOption{{id: "system", label: a.t("System"), key: "settings-language-system"}, {id: "en", label: "English", key: "settings-language-en"}, {id: "ru", label: "Русский", key: "settings-language-ru"}}
		a.selectControl(c, "settings-language-select", "Language", a.preferences.Language, options, func(value string) { a.preferences.Language = value }).Width(200)
	})
	ui.Text(c, a.settingsText("language.systemHint")).FontSize(13).TextColor(s.muted)
}
