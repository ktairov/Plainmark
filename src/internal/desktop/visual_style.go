package desktop

import (
	"github.com/egoist/mygo/ui"
)

// Form controls are larger than the compact title-bar buttons.
type visualStyle struct{ page, control, text, secondary, muted, border, softBorder, strongBorder, accent, accentSoft, accentText, hover, controlHover, danger, dangerText ui.Color }

func (a *App) visual(c *ui.Context) visualStyle {
	p := a.palette(c.Theme().Dark)
	page, border, text, accent, control := ui.Hex(p.Page), ui.Hex(p.Border), ui.Hex(p.Text), ui.Hex(p.Accent), ui.Hex(p.Control)
	dangerText := ui.Hex("#ffffff")
	if c.Theme().Dark {
		dangerText = ui.Hex("#2a110e")
	}
	return visualStyle{page: page, control: control, text: text, secondary: ui.Hex(p.TextSecondary), muted: ui.Hex(p.Muted), border: border, softBorder: border.Mix(page, .35), strongBorder: border.Mix(text, .25), accent: accent, accentSoft: ui.Hex(p.AccentSoft), accentText: ui.Hex(p.AccentContrast), hover: accent.Mix(page, .91), controlHover: accent.Mix(control, .86), danger: ui.Hex(p.Danger), dangerText: dangerText}
}

func (a *App) actionButton(c *ui.Context, label, key, kind string) ui.Element {
	s := a.visual(c)
	b := ui.ButtonBase(c).Label(a.t(label)).Height(36).Padding(7, 12).Radius(6).Border(1, s.border).Background(s.page).TextColor(s.text).FontSize(16)
	if key != "" {
		b = b.Key(key)
	}
	if kind == "primary" {
		b.Background(s.accent).BorderColor(s.accent).TextColor(s.accentText)
	}
	if kind == "danger" {
		b.Background(s.danger).BorderColor(s.danger).TextColor(s.dangerText)
	}
	if b.Hovered() {
		switch kind {
		case "primary":
			b.Background(s.accent.Mix(s.text, .08))
		case "danger":
			b.Background(s.danger.Mix(s.text, .08))
		default:
			b.Background(s.controlHover)
		}
	}
	b.Children(func() { ui.Text(c, a.t(label)).SingleLine() })
	return b
}

func (a *App) formInput(c *ui.Context, key, label string, value *string) ui.Element {
	s := a.visual(c)
	var input ui.Element
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Text(c, a.t(label)).FontSize(13).TextColor(s.muted)
		c.Key(key)
		input = ui.TextInput(c, value).Label(a.t(label)).Height(38).FillWidth().Padding(9).Radius(6).Border(1, s.strongBorder).Background(s.control).TextColor(s.text).FontSize(16)
	})
	return input
}

func (a *App) dialog(c *ui.Context, open *bool, title string, content func()) {
	s := a.visual(c)
	w, _ := c.Size()
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.Hex("#1e2523aa"))
		panel.Label(a.t(title)).Width(min(float32(520), w-32)).Padding(20).Radius(10).Background(s.page).Shadow(0, 18, 60, 0, ui.RGBA(0, 0, 0, .33))
		ui.Column(c).FillWidth().Gap(12).Children(func() {
			ui.Text(c, a.t(title)).FontSize(19).Bold()
			content()
		})
	})
}
