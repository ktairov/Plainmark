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
	}
	a.settingsPreviewMode = ""
	a.settingsOpen = false
}

func (a *App) preferencesModal(c *ui.Context) {
	if !a.settingsOpen {
		return
	}
	w, h := c.Size()
	w, h = min(float32(820), w-40), min(float32(650), h-40)
	t := c.Theme()
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Label(a.t("Settings")).Width(w).Height(h).Radius(12).Border(1, t.Border).Background(t.Background).Shadow(0, 18, 60, 0, ui.RGBA(0, 0, 0, .3)).Clip()
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Height(56).Shrink(0).Padding(0, 20).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
				ui.Text(c, a.t("Settings")).FontSize(18).Bold()
				ui.Box(c).Grow(1)
				if a.iconButton(c, "x", "Cancel", "close-settings").Clicked() {
					a.closeSettings(false)
				}
			})
			ui.Row(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
				ui.Column(c).Width(166).Shrink(0).FillHeight().Padding(12).Gap(5).Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
					for _, item := range []struct{ id, icon string }{{"appearance", "type"}, {"theme", "palette"}, {"language", "languages"}} {
						label := a.settingsText("section." + item.id)
						button := ui.ButtonBase(c).Key("settings-section-"+item.id).Label(label).FillWidth().Height(38).Padding(0, 10).Gap(8).Justify(ui.Start).Radius(7).Selected(a.settingsSection == item.id)
						if a.settingsSection == item.id {
							button.Background(t.Accent.Mix(t.Background, .88)).TextColor(t.Accent)
						} else if button.Hovered() {
							button.Background(t.SurfaceHover)
						}
						button.Children(func() { ui.Icon(c, icons.Get(item.icon)).Size(17, 17); ui.Text(c, label).FontSize(13) })
						if button.Clicked() {
							a.settingsSection = item.id
						}
					}
				})
				c.Key("settings-content-" + a.settingsSection)
				ui.Scroll(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
					ui.Column(c).FillWidth().Padding(22, 24).Gap(18).Children(func() {
						ui.Column(c).FillWidth().Gap(6).Padding(0, 0, 16, 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
							ui.Text(c, a.settingsText("section."+a.settingsSection)).FontSize(20).Bold()
							ui.Text(c, a.settingsText(a.settingsSection+".description")).FontSize(13).TextColor(t.TextMuted)
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
			ui.Row(c).FillWidth().Height(62).Shrink(0).Padding(0, 20).Gap(8).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
				if ui.Button(c, a.t("Reset defaults")).Key("reset-settings").Clicked() {
					a.preferences = settings.Defaults()
					a.settingsPreviewMode = ""
				}
				ui.Box(c).Grow(1)
				if ui.PrimaryButton(c, a.t("Apply")).Key("apply-settings").Clicked() {
					a.closeSettings(true)
				}
				if ui.Button(c, a.t("Cancel")).Key("cancel-settings").Clicked() {
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
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		ui.Text(c, a.t("Color mode")).Bold().FontSize(13)
		modes := []string{"system", "light", "dark"}
		selected := 0
		for i, id := range modes {
			if a.preferences.ColorMode == id {
				selected = i
			}
		}
		c.Key("settings-color-mode")
		control := ui.Segmented(c, &selected, a.settingsText("colorMode.system"), a.settingsText("colorMode.light"), a.settingsText("colorMode.dark")).Label(a.t("Color mode")).FillWidth()
		if control.Changed() {
			a.preferences.ColorMode = modes[selected]
			a.settingsPreviewMode = ""
		}
	})
	a.fontSettings(c, false)
	a.fontSettings(c, true)
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		ui.Row(c).FillWidth().Children(func() {
			ui.Text(c, a.t("Preview text width")).Bold().FontSize(13)
			ui.Box(c).Grow(1)
			ui.Text(c, strconv.Itoa(a.preferences.PreviewWidth)+" px").TextColor(t.TextMuted).FontSize(12)
		})
		width := float64(a.preferences.PreviewWidth)
		c.Key("settings-preview-width")
		ui.Slider(c, &width, 560, 1400).Step(20).Label(a.t("Preview text width")).FillWidth()
		a.preferences.PreviewWidth = int(width)
	})
}

func (a *App) fontSettings(c *ui.Context, reading bool) {
	label, hint, sizeLabel := "Interface font", "uiFontHint", "Interface text size"
	font, size, upper := &a.preferences.UIFont, &a.preferences.TreeFontSize, 21
	if reading {
		label, hint, sizeLabel = "Reading font", "readingFontHint", "Document text size"
		font, size, upper = &a.preferences.ReadingFont, &a.preferences.ContentFontSize, 26
	}
	ui.Column(c).FillWidth().Gap(10).Padding(0, 0, 18, 0).BorderWidth(0, 0, 1, 0).BorderColor(c.Theme().Border).Children(func() {
		ui.Text(c, a.t(label)).Bold().FontSize(13)
		ui.Text(c, a.settingsText(hint)).FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Row(c).FillWidth().Gap(10).Children(func() {
			ids := []string{"system", "humanist", "serif", "mono"}
			labels := []string{a.t("System"), a.t("Humanist"), a.t("Serif"), a.t("Monospace")}
			selected := labels[0]
			for i, id := range ids {
				if *font == id {
					selected = labels[i]
				}
			}
			c.Key("settings-" + label)
			ui.Select(c, &selected, labels).Label(a.t(label)).Grow(1)
			for i, text := range labels {
				if selected == text {
					*font = ids[i]
					break
				}
			}
			var sizes []string
			for n := 13; n <= upper; n++ {
				sizes = append(sizes, strconv.Itoa(n)+" px")
			}
			selectedSize := strconv.Itoa(*size) + " px"
			c.Key("settings-" + sizeLabel)
			ui.Select(c, &selectedSize, sizes).Label(a.t(sizeLabel)).Width(100)
			for i, text := range sizes {
				if selectedSize == text {
					*size = i + 13
					break
				}
			}
		})
	})
}

func (a *App) themeSettings(c *ui.Context, dark bool) {
	label, hint, mode := "Day theme", "theme.dayHint", "light"
	value := &a.preferences.DayTheme
	if dark {
		label, hint, mode, value = "Night theme", "theme.nightHint", "dark", &a.preferences.NightTheme
	}
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		ui.Text(c, a.t(label)).Bold().FontSize(14)
		ui.Text(c, a.settingsText(hint)).FontSize(12).TextColor(t.TextMuted)
		ui.Grid(c).Columns(2).Gap(10).FillWidth().Children(func() {
			for _, id := range settings.ThemeOrder {
				definition := settings.Themes[id]
				palette := definition.Light
				if dark {
					palette = definition.Dark
				}
				button := ui.ButtonBase(c).Key("theme-"+mode+"-"+id).Label(a.t(label)+": "+definition.Label).Height(60).Padding(10).Gap(10).Radius(8).Background(t.Surface).Border(1, t.Border).Selected(*value == id)
				button.Background(t.Surface).TextColor(t.Text)
				if *value == id || button.Hovered() {
					button.Border(2, t.Accent)
				}
				button.Children(func() {
					ui.Row(c).Gap(0).Children(func() {
						for _, color := range []string{palette.Page, palette.Control, palette.Text, palette.Accent} {
							ui.Box(c).Size(12, 26).Background(ui.Hex(color))
						}
					})
					ui.Text(c, definition.Label).FontSize(13).Grow(1)
					if *value == id {
						ui.Icon(c, icons.Get("check")).Size(14, 14).TextColor(t.Accent)
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
	for _, option := range []struct{ id, name string }{{"system", a.t("System")}, {"en", "English"}, {"ru", "Русский"}} {
		button := ui.Button(c, option.name).Key("settings-language-" + option.id).FillWidth().Height(42).Selected(a.preferences.Language == option.id)
		if a.preferences.Language == option.id {
			button.Background(c.Theme().SurfaceHover).TextColor(c.Theme().Text)
		}
		if button.Clicked() {
			a.preferences.Language = option.id
		}
	}
	ui.Text(c, a.settingsText("language.systemHint")).FontSize(12).TextColor(c.Theme().TextMuted)
}
