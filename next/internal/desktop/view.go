package desktop

import (
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"plainmark/internal/icons"
	"plainmark/internal/session"
	"plainmark/internal/settings"
	"plainmark/internal/workspace"
)

type fileDrag struct{ Path string }
type tabDrag struct{ Path string }

func (a *App) view(c *ui.Context) {
	a.applyStyle(c)
	a.flush()
	a.syncMenu()
	t := c.Theme()
	root := ui.Column(c).Fill()
	if root.Shortcut(ui.Cmd, ui.KeyS) {
		a.save()
	}
	if root.Shortcut(ui.Cmd, ui.KeyO) {
		a.chooseFile()
	}
	if root.Shortcut(ui.Cmd|ui.Shift, ui.KeyO) {
		a.chooseFolder()
	}
	if root.Shortcut(ui.Cmd, ui.KeyW) {
		if d := a.current(); d != nil {
			a.closeTabs([]string{d.Tab.Note.Path})
		}
	}
	if root.Shortcut(ui.Cmd, ui.KeyN) && a.store != nil && !a.busy {
		a.form = &pathForm{operation: "note", path: a.activeFolder}
	}
	if d := a.current(); d != nil && d.Tab.Mode == session.Preview && root.Shortcut(ui.Cmd, ui.KeyF) {
		a.find()
	}
	root.Children(func() {
		a.toolbar(c)
		if a.sidebarVisible {
			ui.Split(c, &a.sidebarWidth, func() { a.sidebar(c) }, func() { a.documentArea(c) }).Grow(1).FillWidth()
		} else {
			a.documentArea(c)
		}
		ui.Row(c).Padding(4, 12).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, a.displayStatus()).Key("status").FontSize(12).TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			if d := a.current(); d != nil {
				p := d.Code.CursorPos()
				ui.Text(c, a.positionText(p.Line+1, p.Col+1)).FontSize(12)
			}
		})
	})
	a.modals(c)
	a.cryptoModal(c)
	a.preferencesModal(c)
	a.persistState()
	a.pumpFiles()
}

func (a *App) pinned() map[string]bool {
	root := ""
	if a.store != nil {
		root = a.store.Root()
	}
	if a.pins[root] == nil {
		a.pins[root] = map[string]bool{}
	}
	return a.pins[root]
}
func (a *App) matches(n workspace.Entry) bool {
	q := strings.ToLower(strings.TrimSpace(a.query))
	if q == "" {
		return true
	}
	text := n.Path + " " + n.Title + " " + n.Folder
	for _, h := range n.Headings {
		text += " " + h.Text
	}
	return strings.Contains(strings.ToLower(text), q)
}
func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().FontSize(float32(a.preferences.TreeFontSize)).Background(t.Background).Children(func() {
		search := ui.SearchField(c, &a.query).Key("note-search").Label(a.t("Search notes")).Placeholder(a.t("Search notes")).Margin(8)
		if a.focusSearch {
			search.Focus()
			a.focusSearch = false
		}
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			if a.store == nil {
				return
			}
			root := ui.ButtonBase(c).Key("workspace-root").Label(filepath.Base(a.store.Root())).FillWidth().Height(30).Padding(0, 12).Gap(8).Justify(ui.Start).Children(func() {
				ui.Icon(c, icons.Get("folder-open")).Size(16, 16).TextColor(t.TextMuted)
				ui.Text(c, filepath.Base(a.store.Root())).Bold().SingleLine().Ellipsis("…")
			})
			if root.Clicked() {
				a.activeFolder = ""
			}
			root.ContextMenu(func(m *ui.Menu) { a.pathMenu(m, "", true) })
			a.dropFolder(root, "")
			var pinned []workspace.Entry
			for _, n := range a.snapshot.Notes {
				if a.pinned()[n.Path] && a.matches(n) {
					pinned = append(pinned, n)
				}
			}
			sort.Slice(pinned, func(i, j int) bool { return pinned[i].Title < pinned[j].Title })
			if len(pinned) > 0 {
				ui.Text(c, a.t("Pinned")).FontSize(12).Bold().Margin(8)
				for _, n := range pinned {
					a.noteRow(c, n, true)
				}
			}
			ui.Tree(c, func() { a.folderChildren(c, "") }).FillWidth()
		})
	})
}
func parent(path string) string {
	p := filepath.ToSlash(filepath.Dir(path))
	if p == "." {
		return ""
	}
	return p
}
func (a *App) folderChildren(c *ui.Context, folder string) {
	for _, f := range a.snapshot.Folders {
		if parent(f.Path) != folder {
			continue
		}
		visible := strings.TrimSpace(a.query) == ""
		if !visible {
			for _, n := range a.snapshot.Notes {
				if strings.HasPrefix(n.Path, f.Path+"/") && a.matches(n) {
					visible = true
					break
				}
			}
		}
		if !visible {
			continue
		}
		open := a.expanded[f.Path]
		if open == nil {
			value := false
			open = &value
			a.expanded[f.Path] = open
		}
		forced := true
		if strings.TrimSpace(a.query) != "" {
			open = &forced
		}
		c.Key("folder:" + f.Path)
		row := ui.TreeItem(c, "", open, func() { a.folderChildren(c, f.Path) }).Label(f.Name).Children(func() {
			icon := "folder"
			if *open {
				icon = "folder-open"
			}
			ui.Icon(c, icons.Get(icon)).Size(14, 14).TextColor(c.Theme().TextMuted)
			ui.Text(c, f.Name).SingleLine().Ellipsis("…")
		}).Gap(6).Selected(a.activeFolder == f.Path).Drag(fileDrag{f.Path})
		if a.activeFolder == f.Path {
			row.Background(c.Theme().SurfaceHover).TextColor(c.Theme().Text)
		}
		if row.Clicked() {
			a.activeFolder = f.Path
		}
		row.ContextMenu(func(m *ui.Menu) { a.pathMenu(m, f.Path, true) })
		a.dropFolder(row, f.Path)
	}
	for _, n := range a.snapshot.Notes {
		if n.Folder == folder && a.matches(n) && !a.pinned()[n.Path] {
			a.noteRow(c, n, false)
		}
	}
}
func (a *App) noteRow(c *ui.Context, n workspace.Entry, pinned bool) {
	prefix := "note:"
	if pinned {
		prefix = "pinned:"
	}
	c.Key(prefix + n.Path)
	var row ui.Element
	if pinned {
		row = ui.ButtonBase(c).FillWidth().Height(28).Padding(0, 12).Gap(8).Selected(a.session.Active == n.Path)
	} else {
		row = ui.TreeItem(c, "", nil, nil).Selected(a.session.Active == n.Path)
	}
	row.Label(n.Title).Gap(6).Children(func() {
		name := "file-text"
		if pinned {
			name = "pin"
		}
		ui.Icon(c, icons.Get(name)).Size(14, 14).TextColor(c.Theme().TextMuted)
		ui.Text(c, n.Title).SingleLine().Ellipsis("…")
	})
	if a.session.Active == n.Path {
		row.Background(c.Theme().SurfaceHover).TextColor(c.Theme().Text)
	}
	row.Tooltip(n.Path).Disabled(a.busy).Drag(fileDrag{n.Path})
	if row.Clicked() {
		a.activeFolder = n.Folder
		a.openNote(n.Path, "")
	}
	row.ContextMenu(func(m *ui.Menu) { a.pathMenu(m, n.Path, false) })
}
func (a *App) dropFolder(row ui.Element, path string) {
	if _, over := ui.DragOver[fileDrag](row); over {
		row.Border(1, ui.Hex("#4f8cff"))
	}
	if drag, ok := ui.Drop[fileDrag](row); ok && !a.busy {
		a.movePath(drag.Path, path)
	}
}

func (a *App) documentArea(c *ui.Context) {
	ui.Column(c).Grow(1).FillHeight().MinWidth(0).Children(func() {
		ui.ScrollHorizontal(c).FillWidth().Shrink(0).Background(c.Theme().Background).BorderWidth(0, 0, 1, 0).BorderColor(c.Theme().Border).Children(func() {
			ui.Row(c).Gap(4).Padding(5, 8).Children(func() {
				for _, tab := range slices.Clone(a.session.Tabs) {
					tabRow := ui.Row(c).Key(tab).Drag(tabDrag{tab.Note.Path}).AlignItems(ui.Center).Radius(6)
					if a.session.Active == tab.Note.Path {
						tabRow.Background(c.Theme().Surface).Border(1, c.Theme().Border)
					}
					tabRow.Children(func() {
						label := tab.Note.Title
						if tab.Dirty() {
							label += " •"
						}
						button := ui.ButtonBase(c).Height(28).Padding(0, 10).Gap(6).Children(func() {
							ui.Icon(c, icons.Get("file-text")).Size(14, 14).TextColor(c.Theme().TextMuted)
							ui.Text(c, label).SingleLine().MaxWidth(180).Ellipsis("…")
						}).Label(a.t("Tab {title}", "title", tab.Note.Title)).Selected(a.session.Active == tab.Note.Path).Tooltip(tab.Note.Path)
						if a.session.Active == tab.Note.Path {
							button.Background(c.Theme().Surface).TextColor(c.Theme().Text)
						}
						if button.Clicked() {
							a.activate(tab.Note.Path)
						}
						if drag, ok := ui.Drop[tabDrag](button); ok {
							a.session.Reorder(drag.Path, tab.Note.Path)
						}
						button.ContextMenu(func(m *ui.Menu) {
							if m.Item(a.t("Close tab")).Chosen() {
								a.closeTabs([]string{tab.Note.Path})
							}
							if m.Item(a.t("Close other tabs")).Chosen() {
								var paths []string
								for _, t := range a.session.Tabs {
									if t != tab {
										paths = append(paths, t.Note.Path)
									}
								}
								a.closeTabs(paths)
							}
							if m.Item(a.t("Close all tabs")).Chosen() {
								a.closeTabs(a.allPaths())
							}
						})
						if a.iconButton(c, "x", a.t("Close {title}", "title", tab.Note.Title), "").Size(24, 24).Disabled(a.busy).Clicked() {
							a.closeTabs([]string{tab.Note.Path})
						}
					})
				}
			})
		})
		d := a.current()
		if d == nil {
			ui.Column(c).Grow(1).FillWidth().Center().Children(func() { ui.Text(c, a.t("Open a note from the sidebar")).TextColor(c.Theme().TextMuted) })
			return
		}
		ui.Text(c, d.Tab.Note.Path).FontSize(12).TextColor(c.Theme().TextMuted).Padding(4, 12)
		if d.Tab.Conflict {
			ui.Row(c).Gap(8).Padding(8).Background(c.Theme().Surface).Children(func() {
				message := "File changed on disk. Your draft is kept."
				if d.Tab.Removed {
					message = "File was removed. Your draft is kept."
				}
				ui.Text(c, a.t(message))
				if ui.Button(c, a.t("Reload")).Disabled(a.busy || d.Tab.Removed).Clicked() {
					a.reload(d)
				}
				if ui.Button(c, a.t("Save copy")).Disabled(a.busy).Clicked() {
					a.saveCopy(d)
				}
				if ui.Button(c, a.t("Overwrite")).Disabled(a.busy || d.Tab.Removed).Clicked() {
					a.saveDocument(d, true, nil)
				}
			})
		}
		ui.Column(c).Key(d.Tab).Grow(1).FillWidth().MinHeight(0).Children(func() {
			switch d.Tab.Mode {
			case session.Preview:
				a.preview(c, d)
			case session.Edit:
				a.edit(c, d)
			default:
				ui.SplitPercent(c, &a.ratio, func() { a.edit(c, d) }, func() { a.preview(c, d) }).Grow(1).FillWidth()
			}
		})
	})
}
func (a *App) edit(c *ui.Context, d *Document) {
	d.Code.SoftWrap = a.wrap
	d.Code.Font = ui.Font{Family: settings.Font("mono"), Size: float32(a.preferences.ContentFontSize)}
	editor.View(c, d.Code).Key("editor").Label(a.t("Markdown editor")).Disabled(a.busy).Grow(1).FillWidth()
}
