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
	ui.Toolbar(c, func() {
		sidebarIcon := "panel-left-open"
		if a.sidebarVisible {
			sidebarIcon = "panel-left-close"
		}
		if a.iconButton(c, sidebarIcon, "Toggle sidebar", "toggle-sidebar").Clicked() {
			a.sidebarVisible = !a.sidebarVisible
		}
		if a.iconButton(c, "folder-open", "Open folder", "open-folder").Disabled(a.busy).Clicked() {
			a.chooseFolder()
		}
		if a.store != nil {
			ui.Text(c, filepath.Base(a.store.Root())).FontSize(13).MaxWidth(140).SingleLine().Ellipsis("…").Tooltip(a.store.Root())
		}
		if a.iconButton(c, "file-text", "Open file", "open").Disabled(a.busy).Clicked() {
			a.chooseFile()
		}
		if a.iconButton(c, "file-plus-2", "New note", "new-note").Disabled(a.busy || a.store == nil).Clicked() {
			a.form = &pathForm{operation: "note", path: a.activeFolder}
		}
		d := a.current()
		if a.iconButton(c, "save", "Save", "save").Disabled(a.busy || d == nil || (d != nil && !d.Tab.Dirty())).Clicked() {
			a.save()
		}
		ui.Box(c).Grow(1).MinWidth(8).FillHeight().DragWindow()
		if d != nil {
			ui.Row(c).Gap(2).Padding(2).Radius(7).Background(t.Surface).Children(func() {
				for _, mode := range []struct {
					icon, label string
					value       session.Mode
				}{{"eye", "Preview", session.Preview}, {"pencil-line", "Edit", session.Edit}, {"columns-2", "Split", session.Split}} {
					button := a.iconButton(c, mode.icon, mode.label, "mode-"+mode.icon).Selected(d.Tab.Mode == mode.value)
					if d.Tab.Mode == mode.value {
						button.Background(t.SurfaceHover).TextColor(t.Text)
					}
					if button.Clicked() {
						d.Tab.Mode = mode.value
					}
				}
			})
		}
		if a.iconButton(c, "search", "Find", "find").Disabled(d == nil || a.busy).Clicked() {
			a.find()
		}
		ui.Checkbox(c, &a.wrap, a.t("Wrap"))
		themeIcon := "moon"
		if t.Dark {
			themeIcon = "sun"
		}
		if a.iconButton(c, themeIcon, "Toggle color mode", "toggle-color").Clicked() {
			a.toggleColor()
		}
		a.iconButton(c, "ellipsis", "More", "more").PopoverMenuButton(func(m *ui.Menu) {
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
	}).Label(a.t("View mode")).Height(max(float32(44), bar.Height)).Shrink(0).FillWidth().Padding(0, max(float32(8), bar.Right), 0, max(float32(8), bar.Left)).Gap(4).Background(t.Background).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).DragWindow()
}
