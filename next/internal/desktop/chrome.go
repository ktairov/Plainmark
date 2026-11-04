package desktop

import (
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"plainmark/internal/icons"
	"plainmark/internal/session"
)

// Keep labels and tooltips even when the visual content is only a vector.
func (a *App) iconButton(c *ui.Context, icon, label, key string) ui.Element {
	b := ui.Button(c, "").Label(a.t(label)).Tooltip(a.t(label)).Size(30, 30).Padding(6).Radius(6).BorderWidth(0)
	if key != "" {
		b = b.Key(key)
	}
	b.Children(func() { ui.Icon(c, icons.Get(icon)).Size(16, 16) })
	return b
}

func (a *App) toolbar(c *ui.Context) {
	t := c.Theme()
	bar := c.TitleBar()
	left := float32(8)
	if bar.Left > 0 {
		left = max(float32(78), bar.Left+8)
	}
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		sidebarIcon := "panel-left-open"
		if a.sidebarVisible {
			sidebarIcon = "panel-left-close"
		}
		if a.windowButton(c, sidebarIcon, "Toggle sidebar", "toggle-sidebar").Clicked() {
			a.sidebarVisible = !a.sidebarVisible
		}
		label := a.t("Open folder")
		if a.store != nil {
			label = filepath.Base(a.store.Root())
		}
		folder := ui.ButtonBase(c).Key("open-folder").Label(a.t("Open folder")).Tooltip(a.t("Open folder")).Height(28).MaxWidth(150).Padding(0, 6).Gap(4).Radius(6).Disabled(a.busy)
		if folder.Hovered() {
			folder.Background(t.SurfaceHover)
		}
		folder.Children(func() {
			ui.Text(c, label).SingleLine().Ellipsis("…").MinWidth(0)
			ui.Icon(c, icons.Get("chevron-down")).Size(13, 13).TextColor(t.TextMuted)
		})
		folder.PopoverMenuButton(func(m *ui.Menu) {
			for _, path := range a.recent {
				if m.Item(path).Disabled(a.busy).Chosen() {
					a.switchTo(path)
				}
			}
			m.Separator()
			if m.Item(a.t("Open folder")).Disabled(a.busy).Chosen() {
				a.chooseFolder()
			}
			if m.Item(a.t("Open file")).Disabled(a.busy).Chosen() {
				a.chooseFile()
			}
		})
		a.windowButton(c, "plus", "Create", "create").Disabled(a.busy || a.store == nil).PopoverMenuButton(func(m *ui.Menu) {
			if m.Item(a.t("New note")).Chosen() {
				a.form = &pathForm{operation: "note", path: a.activeFolder}
			}
			if m.Item(a.t("New folder")).Chosen() {
				a.form = &pathForm{operation: "folder", path: a.activeFolder}
			}
		})
		d := a.current()
		if a.windowButton(c, "save", "Save", "save").Disabled(a.busy || d == nil || (d != nil && !d.Tab.Dirty())).Clicked() {
			a.save()
		}
		ui.Box(c).Width(1).Height(18).Margin(0, 6).Background(t.Border)
		a.tabs(c)

		if d != nil {
			ui.Row(c).Gap(2).Padding(2).Radius(7).Background(t.Surface).Children(func() {
				for _, mode := range []struct {
					icon, label string
					value       session.Mode
				}{{"eye", "Preview", session.Preview}, {"pencil-line", "Edit", session.Edit}, {"columns-2", "Split", session.Split}} {
					button := a.windowButton(c, mode.icon, mode.label, "mode-"+mode.icon).Selected(d.Tab.Mode == mode.value)
					if d.Tab.Mode == mode.value {
						button.Background(t.SurfaceHover).TextColor(t.Text)
					}
					if button.Clicked() {
						d.Tab.Mode = mode.value
					}
				}
			})
		}
		themeIcon := "moon"
		if t.Dark {
			themeIcon = "sun"
		}
		if a.windowButton(c, themeIcon, "Toggle color mode", "toggle-color").Clicked() {
			a.toggleColor()
		}
		if a.windowButton(c, "settings", "Settings", "settings").Disabled(a.busy).Clicked() {
			a.openSettings()
		}
		a.windowButton(c, "ellipsis", "More", "more").PopoverMenuButton(func(m *ui.Menu) {
			if m.Item(a.t("Open file")).Disabled(a.busy).Chosen() {
				a.chooseFile()
			}
			if m.Item(a.t("Find")).Disabled(d == nil || a.busy).Chosen() {
				a.find()
			}
			if m.Item(a.t("Wrap")).Checked(a.wrap).Chosen() {
				a.wrap = !a.wrap
			}
			if m.Item(a.t("Settings")).Chosen() {
				a.openSettings()
			}
			if m.Item(a.t("Toggle color mode")).Chosen() {
				a.toggleColor()
			}
			if m.Item(a.t("Encrypt selection")).Disabled(a.busy || a.current() == nil).Chosen() {
				a.beginEncrypt()
			}
			if m.Item(a.t("Toggle sidebar")).Chosen() {
				a.sidebarVisible = !a.sidebarVisible
			}
			if m.Item(a.t("Refresh")).Disabled(a.busy || a.store == nil).Chosen() {
				a.refresh()
			}
			if m.Item(a.t("New folder")).Disabled(a.busy || a.store == nil).Chosen() {
				a.form = &pathForm{operation: "folder", path: a.activeFolder}
			}
			m.Submenu(a.t("Recent folders"), func(m *ui.Menu) {
				for _, p := range a.recent {
					if m.Item(p).Disabled(a.busy).Chosen() {
						a.switchTo(p)
					}
				}
			})
		})
	}).Role(ui.RoleToolbar).Label(a.t("View mode")).Height(max(float32(44), bar.Height)).Shrink(0).FillWidth().Padding(0, max(float32(8), bar.Right), 0, left).Gap(4).Background(t.Background).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).DragWindow()
}

// Window chrome uses flat controls like the original title bar.
func (a *App) windowButton(c *ui.Context, icon, label, key string) ui.Element {
	button := ui.ButtonBase(c).Size(28, 28).Padding(6).Radius(6).Label(a.t(label)).Tooltip(a.t(label)).TextColor(c.Theme().TextMuted)
	if key != "" {
		button = button.Key(key)
	}
	if button.Hovered() {
		button.Background(c.Theme().Accent.Mix(c.Theme().Surface, .86)).TextColor(c.Theme().Text)
	}
	button.Children(func() { ui.Icon(c, icons.Get(icon)).Size(16, 16) })
	return button
}
