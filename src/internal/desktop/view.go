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
	if a.popup != nil {
		a.popup.seen = false
	}
	a.applyStyle(c)
	a.flush()
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
	if !a.menuBlocked() {
		if c.Shortcut(ui.Cmd, ui.KeyComma) {
			a.closePopup()
			a.openSettings()
		}
		for i := range min(len(a.session.Tabs), 9) {
			if c.Shortcut(ui.Cmd, ui.Key1+ui.Key(i)) {
				a.activate(a.session.Tabs[i].Note.Path)
			}
		}
		if d := a.current(); d != nil {
			for _, choice := range viewModes {
				if c.Shortcut(ui.Alt, choice.key) {
					d.Tab.Mode = choice.mode
				}
			}
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
			a.sidebarSplit(c)
		} else {
			a.documentArea(c)
		}
		ui.Row(c).FillWidth().Padding(4, 14, 8).Children(func() {
			ui.Text(c, a.displayStatus()).Key("status").FontSize(12).TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			if d := a.current(); d != nil && d.Tab.Mode != session.Preview {
				p := d.Code.CursorPos()
				ui.Text(c, a.positionText(p.Line+1, p.Col+1)).FontSize(12)
			}
		})
	})
	a.modals(c)
	a.cryptoModal(c)
	a.preferencesModal(c)
	a.drawPopup(c)
	a.flush()
	if a.refreshNeeded && !a.busy && !a.readingNote {
		a.refreshNeeded = false
		a.refresh()
	}
	a.syncMenu()
	a.persistState()
	a.pumpFiles()
}

func (a *App) sidebarSplit(c *ui.Context) {
	theme := c.Theme()
	splitTheme := *theme
	// MyGo paints the divider with Border at rest and Accent on hover or drag.
	splitTheme.Border = ui.Transparent
	splitTheme.Accent = ui.Transparent
	ui.Themed(c, &splitTheme).Background(ui.Transparent).Grow(1).FillWidth().MinHeight(0).Children(func() {
		ui.Split(c, &a.sidebarWidth, func() {
			ui.Themed(c, theme).Background(ui.Transparent).Fill().Children(func() { a.sidebar(c) })
		}, func() {
			ui.Themed(c, theme).Background(ui.Transparent).Fill().MinWidth(0).Children(func() { a.documentArea(c) })
		}).Grow(1).FillWidth()
	})
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
	ui.Column(c).Fill().FontSize(float32(a.preferences.TreeFontSize)).Children(func() {
		c.Key("note-search")
		search := ui.SearchField(c, &a.query).Label(a.t("Search notes")).Placeholder(a.t("Search notes")).Height(30).Radius(8).Background(t.Surface).Border(1, t.Border).Margin(10, 12, 8, 12)
		if a.focusSearch {
			search.Focus()
			a.focusSearch = false
		}
		list := ui.Scroll(c).Grow(1).FillWidth().Padding(0, 10, 16, 10).Children(func() {
			if a.store == nil {
				return
			}

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
			root := ui.ButtonBase(c).Key("workspace-root").Label(filepath.Base(a.store.Root())).FillWidth().Height(30).Padding(0, 12).Gap(8).Justify(ui.Start).Children(func() {
				ui.Icon(c, icons.Get("folder-open")).Size(16, 16).TextColor(t.TextMuted)
				ui.Text(c, filepath.Base(a.store.Root())).Bold().SingleLine().Ellipsis("…")
			})
			rootContext := a.pathPopup(c, root, "", true)
			if root.Clicked() && !rootContext {
				a.activeFolder = ""
			}
			a.dropFolder(root, "", t.Accent)
			ui.Tree(c, func() { a.folderChildren(c, "") }).FillWidth()
		})
		if a.store != nil {
			a.pathPopup(c, list, "", true)
		}
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
			ui.Text(c, f.Name).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
		}).Gap(7).Height(30).Radius(7).TextColor(c.Theme().TextMuted).Drag(fileDrag{f.Path})
		if a.activeFolder == f.Path {
			row.Background(c.Theme().SurfaceHover)
		}
		context := a.pathPopup(c, row, f.Path, true)
		if row.Clicked() && !context {
			a.activeFolder = f.Path
			*open = !*open
		}
		a.dropFolder(row, f.Path, c.Theme().Accent)
	}
	for _, n := range a.snapshot.Notes {
		if n.Folder == folder && a.matches(n) && !a.pinned()[n.Path] {
			a.noteRow(c, n, false)
		}
	}
}
func (a *App) noteRow(c *ui.Context, n workspace.Entry, pinned bool) {
	active := a.session.Active == n.Path
	color := c.Theme().TextMuted
	if active {
		color = c.Theme().Text
	}
	prefix := "note:"
	if pinned {
		prefix = "pinned:"
	}
	c.Key(prefix + n.Path)
	var row ui.Element
	if pinned {
		row = ui.ButtonBase(c).FillWidth().Padding(0, 12).Justify(ui.Start).Selected(active)
	} else {
		row = ui.TreeItem(c, "", nil, nil).Selected(active)
	}
	row.Label(n.Title).Gap(7).Height(30).Radius(7).TextColor(color).Children(func() {
		name := "file-text"
		if pinned {
			name = "pin"
		}
		ui.Icon(c, icons.Get(name)).Size(14, 14).TextColor(color)
		ui.Text(c, n.Title).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
	})
	if active {
		row.Background(c.Theme().SurfaceHover)
	} else if row.Hovered() {
		row.Background(c.Theme().Surface)
	}
	row.Tooltip(n.Path).Disabled(a.busy).Drag(fileDrag{n.Path})
	context := a.pathPopup(c, row, n.Path, false)
	if row.Clicked() && !context {
		a.activeFolder = n.Folder
		a.navigateNote(n.Path, "", row.ClickModifiers()&(ui.Cmd|ui.Ctrl) != 0)
	}
}
func (a *App) dropFolder(row ui.Element, path string, accent ui.Color) {
	if _, over := ui.DragOver[fileDrag](row); over {
		row.Border(1, accent)
	}
	if drag, ok := ui.Drop[fileDrag](row); ok && !a.busy {
		a.movePath(drag.Path, path)
	}
}

func (a *App) documentArea(c *ui.Context) {
	t := c.Theme()
	left := float32(10)
	if a.sidebarVisible {
		// Put the document border inside the split handle's hit area.
		left = 0
	}
	ui.Column(c).Grow(1).FillHeight().MinWidth(0).MinHeight(0).Padding(8, 12, 4, left).Children(func() {
		ui.Column(c).Grow(1).FillWidth().MinHeight(0).Radius(12).Border(1, t.Border).Background(t.Background).Clip().Children(func() {
			a.documentContent(c)
		})
	})
}

func (a *App) documentContent(c *ui.Context) {
	d := a.current()
	if d == nil {
		ui.Column(c).Grow(1).FillWidth().Center().Children(func() { ui.Text(c, a.t("Open a note from the sidebar")).TextColor(c.Theme().TextMuted) })
		return
	}
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
			d.splitScroll.ready = false
			a.preview(c, d)
		case session.Edit:
			d.splitScroll.ready = false
			a.edit(c, d)
		default:
			ui.SplitPercent(c, &a.ratio, func() { a.edit(c, d) }, func() { a.preview(c, d) }).Grow(1).FillWidth()
			d.syncSplitScroll(c)
		}
	})
}
func (a *App) edit(c *ui.Context, d *Document) {
	d.Code.SoftWrap = a.wrap
	d.Code.Font = ui.Font{Family: settings.Font("mono"), Size: float32(a.preferences.ContentFontSize)}
	c.Key("editor")
	view := editor.View(c, d.Code).Label(a.t("Markdown editor")).Disabled(a.busy).Grow(1).FillWidth()
	if d.pendingEditorY != nil {
		if view.Bounds().W > 0 && view.Bounds().H > 0 {
			d.Code.SetScrollY(*d.pendingEditorY)
			d.pendingEditorY = nil
		}
		c.After(0)
	}
}

func (a *App) tabs(c *ui.Context) ui.Element {
	strip := ui.ScrollHorizontal(c).Key("tab-strip").Grow(1).MinWidth(0).FillHeight().Children(func() {
		ui.Row(c).Gap(3).Padding(0, 2).Height(max(float32(44), c.TitleBar().Height)).AlignItems(ui.Center).Children(func() {
			for _, tab := range slices.Clone(a.session.Tabs) {
				tabRow := ui.Row(c).Key(tab).Drag(tabDrag{tab.Note.Path}).AlignItems(ui.Center).Radius(7).Width(190).MinWidth(104).MaxWidth(220).Height(28)
				if a.session.Active == tab.Note.Path {
					tabRow.Background(c.Theme().SurfaceHover).Border(1, c.Theme().Border)
					if a.lastVisibleTab != tab {
						tabRow.ScrollIntoView()
						a.lastVisibleTab = tab
					}
				}
				tabRow.Children(func() {
					button := ui.ButtonBase(c).Height(28).Grow(1).MinWidth(0).Justify(ui.Start).Padding(0, 10).TextColor(c.Theme().TextMuted).Children(func() {
						ui.Text(c, tab.Note.Title).SingleLine().MaxWidth(180).MinWidth(0).Shrink(1).Ellipsis("…")
						if tab.Dirty() {
							ui.Box(c).Key("tab-dirty:"+tab.Note.Path).Size(6, 6).Shrink(0).Radius(3).Margin(0, 0, 0, 6).Background(c.Theme().Accent).Role(ui.RoleImage).Label(a.t("Unsaved changes"))
						}
					}).Label(a.t("Tab {title}", "title", tab.Note.Title)).Selected(a.session.Active == tab.Note.Path).Tooltip(tab.Note.Path)
					if a.session.Active == tab.Note.Path {
						button.Background(ui.Transparent).TextColor(c.Theme().Text)
					}
					if tabRow.Hovered() && a.session.Active != tab.Note.Path {
						tabRow.Background(c.Theme().SurfaceHover)
					}
					if button.Clicked() {
						a.activate(tab.Note.Path)
					}
					a.headerDoubleClick(button)
					if drag, ok := ui.Drop[tabDrag](button); ok {
						a.session.Reorder(drag.Path, tab.Note.Path)
					}
					button.ContextMenu(func(m *ui.Menu) {
						if m.Item(a.menuText("closeTab")).Chosen() {
							a.closeTabs([]string{tab.Note.Path})
						}
						if m.Item(a.menuText("closeOtherTabs")).Disabled(a.busy || len(a.session.Tabs) < 2).Chosen() {
							var paths []string
							for _, t := range a.session.Tabs {
								if t != tab {
									paths = append(paths, t.Note.Path)
								}
							}
							a.closeTabs(paths)
						}
						if m.Item(a.menuText("closeAllTabs")).Chosen() {
							a.closeTabs(a.allPaths())
						}
					})
					closeLabel := a.t("Close {title}", "title", tab.Note.Title)
					closeButton := ui.ButtonBase(c).Size(20, 20).Shrink(0).Padding(2).Center().Margin(0, 4, 0, 0).
						Label(closeLabel).Tooltip(closeLabel).TextColor(c.Theme().TextMuted).Disabled(a.busy).
						Children(func() { ui.Icon(c, icons.Get("x")).Size(16, 16).Shrink(0) })
					if closeButton.Clicked() {
						a.closeTabs([]string{tab.Note.Path})
					}
				})
				a.headerDoubleClick(tabRow)
			}
		})
	})
	a.headerDoubleClick(strip)
	return strip
}
