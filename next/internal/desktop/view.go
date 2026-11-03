package desktop

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"

	"plainmark/internal/preview"
	"plainmark/internal/session"
	"plainmark/internal/workspace"
)

type fileDrag struct{ Path string }
type tabDrag struct{ Path string }

func (a *App) view(c *ui.Context) {
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
	root.Children(func() {
		ui.Row(c).Padding(8).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			if ui.Button(c, "Open folder").Key("open-folder").Disabled(a.busy).Clicked() {
				a.chooseFolder()
			}
			if ui.Button(c, "Open file").Key("open").Disabled(a.busy).Clicked() {
				a.chooseFile()
			}
			if ui.Button(c, "New note").Key("new-note").Disabled(a.busy || a.store == nil).Clicked() {
				a.form = &pathForm{operation: "note", path: a.activeFolder}
			}
			d := a.current()
			if ui.PrimaryButton(c, "Save").Key("save").Disabled(a.busy || d == nil || (d != nil && !d.Tab.Dirty())).Clicked() {
				a.save()
			}
			if d != nil {
				mode := int(d.Tab.Mode)
				ui.Segmented(c, &mode, "Preview", "Edit", "Split").Label("View mode")
				d.Tab.Mode = session.Mode(mode)
			}
			ui.Box(c).Grow(1)
			ui.Checkbox(c, &a.wrap, "Wrap")
			ui.MenuButton(c, "More", func(m *ui.Menu) {
				if m.Item("Toggle sidebar").Chosen() {
					a.sidebarVisible = !a.sidebarVisible
				}
				if m.Item("Refresh").Disabled(a.busy || a.store == nil).Chosen() {
					a.refresh()
				}
				if m.Item("New folder").Disabled(a.busy || a.store == nil).Chosen() {
					a.form = &pathForm{operation: "folder", path: a.activeFolder}
				}
				m.Submenu("Recent folders", func(m *ui.Menu) {
					for _, p := range a.recent {
						if m.Item(p).Disabled(a.busy).Chosen() {
							a.switchTo(p)
						}
					}
				})
			})
		})
		if a.sidebarVisible {
			ui.Split(c, &a.sidebarWidth, func() { a.sidebar(c) }, func() { a.documentArea(c) }).Grow(1).FillWidth()
		} else {
			a.documentArea(c)
		}
		ui.Row(c).Padding(4, 12).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, a.status).Key("status").FontSize(12).TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			if d := a.current(); d != nil {
				p := d.Code.CursorPos()
				ui.Text(c, fmt.Sprintf("Ln %d, Col %d", p.Line+1, p.Col+1)).FontSize(12)
			}
		})
	})
	a.modals(c)
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
	ui.Column(c).Fill().Background(t.Surface).Children(func() {
		ui.SearchField(c, &a.query).Key("note-search").Label("Search notes").Placeholder("Search notes").Margin(8)
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			if a.store == nil {
				return
			}
			root := ui.Button(c, filepath.Base(a.store.Root())).Key("workspace-root").FillWidth().Selected(a.activeFolder == "")
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
				ui.Text(c, "Pinned").FontSize(12).Bold().Margin(8)
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
		row := ui.TreeItem(c, f.Name, open, func() { a.folderChildren(c, f.Path) }).Selected(a.activeFolder == f.Path).Drag(fileDrag{f.Path})
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
		row = ui.Button(c, n.Title).FillWidth().Selected(a.session.Active == n.Path)
	} else {
		row = ui.TreeItem(c, n.Title, nil, nil).Selected(a.session.Active == n.Path)
	}
	row.Tooltip(n.Path).Drag(fileDrag{n.Path})
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
		ui.ScrollHorizontal(c).FillWidth().Shrink(0).Children(func() {
			ui.Row(c).Gap(2).Children(func() {
				for _, tab := range slices.Clone(a.session.Tabs) {
					ui.Row(c).Key(tab).Drag(tabDrag{tab.Note.Path}).Children(func() {
						label := tab.Note.Title
						if tab.Dirty() {
							label += " •"
						}
						button := ui.Button(c, label).Label("Tab " + tab.Note.Title).Selected(a.session.Active == tab.Note.Path).Tooltip(tab.Note.Path)
						if button.Clicked() {
							a.session.Active = tab.Note.Path
						}
						if drag, ok := ui.Drop[tabDrag](button); ok {
							a.session.Reorder(drag.Path, tab.Note.Path)
						}
						button.ContextMenu(func(m *ui.Menu) {
							if m.Item("Close tab").Chosen() {
								a.closeTabs([]string{tab.Note.Path})
							}
							if m.Item("Close other tabs").Chosen() {
								var paths []string
								for _, t := range a.session.Tabs {
									if t != tab {
										paths = append(paths, t.Note.Path)
									}
								}
								a.closeTabs(paths)
							}
							if m.Item("Close all tabs").Chosen() {
								a.closeTabs(a.allPaths())
							}
						})
						if ui.Button(c, "×").Label("Close " + tab.Note.Title).Disabled(a.busy).Clicked() {
							a.closeTabs([]string{tab.Note.Path})
						}
					})
				}
			})
		})
		d := a.current()
		if d == nil {
			ui.Column(c).Grow(1).FillWidth().Center().Children(func() { ui.Text(c, "Open a note from the sidebar").TextColor(c.Theme().TextMuted) })
			return
		}
		ui.Text(c, d.Tab.Note.Path).FontSize(12).TextColor(c.Theme().TextMuted).Padding(4, 12)
		if d.Tab.Conflict {
			ui.Row(c).Gap(8).Padding(8).Background(c.Theme().Surface).Children(func() {
				message := "File changed on disk. Your draft is kept."
				if d.Tab.Removed {
					message = "File was removed. Your draft is kept."
				}
				ui.Text(c, message)
				if ui.Button(c, "Reload").Disabled(a.busy || d.Tab.Removed).Clicked() {
					a.reload(d)
				}
				if ui.Button(c, "Save copy").Disabled(a.busy).Clicked() {
					a.saveCopy(d)
				}
				if ui.Button(c, "Overwrite").Disabled(a.busy || d.Tab.Removed).Clicked() {
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
func (a *App) preview(c *ui.Context, d *Document) {
	a.math.Configure(&d.MD, c.Theme().Text, 20)
	if !d.previewReady || d.previewVersion != d.Code.Buffer.Version() {
		d.previewVersion = d.Code.Buffer.Version()
		d.previewSource = preview.Normalize(d.Code.Buffer.String())
		d.previewReady = true
	}
	ui.Scroll(c).TrackScroll(&d.Scroll).Grow(1).FillWidth().Padding(20).Children(func() { markdown.View(c, d.previewSource, &d.MD).FillWidth() })
}
func (a *App) edit(c *ui.Context, d *Document) {
	d.Code.SoftWrap = a.wrap
	editor.View(c, d.Code).Key("editor").Label("Markdown editor").Disabled(a.busy).Grow(1).FillWidth()
}
