package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"plainmark/internal/session"
	"plainmark/internal/workspace"
)

func (a *App) pathItems(path string, folder bool) []popupItem {
	blocked := a.menuBlocked()
	var items []popupItem
	add := func(key, icon string, danger bool, action func()) {
		items = append(items, popupItem{label: a.contextText(key), icon: icon, key: "context-" + key, disabled: blocked, danger: danger, action: action})
	}
	if folder {
		add("newNote", "file-plus-2", false, func() { a.form = &pathForm{operation: "note", path: path} })
		add("newFolder", "folder-plus", false, func() { a.form = &pathForm{operation: "folder", path: path} })
	} else {
		add("copyPath", "clipboard-copy", false, func() { a.pathAction(path, false) })
		add("openInNewTab", "file-plus-2", false, func() { a.openNote(path, "") })
		pinKey := "pin"
		if a.pinned()[path] {
			pinKey = "unpin"
		}
		add(pinKey, "pin", false, func() { a.pinned()[path] = !a.pinned()[path] })
	}
	if path != "" {
		add("rename", "ellipsis-vertical", false, func() { a.form = &pathForm{operation: "rename", path: path, value: filepath.Base(path)} })
		add("move", "folder-open", false, func() { a.form = &pathForm{operation: "move", path: path} })
		add("trash", "trash-2", true, func() { a.askDelete(path) })
	}
	return items
}
func (a *App) pathMenu(m *ui.Menu, path string, folder bool) {
	for _, item := range a.pathItems(path, folder) {
		if m.Item(item.label).Disabled(item.disabled).Chosen() {
			item.action()
		}
	}
}
func (a *App) pathPopup(c *ui.Context, row ui.Element, path string, folder bool) bool {
	return a.bindPopup(c, row, "context:"+path, a.pathItems(path, folder), true)
}
func (a *App) pathAction(path string, reveal bool) {
	w := a.store
	a.async(func() (func(), error) {
		full, err := w.Absolute(path)
		if err != nil {
			return nil, err
		}
		if reveal {
			mygo.Shell.ShowItemInFolder(full)
			return nil, nil
		}
		mygo.Clipboard.WriteText(full)
		return nil, nil
	})
}
func (a *App) operation(from string, openCreated bool, work func(*workspace.Workspace) (workspace.Result, error)) {
	if a.store == nil {
		return
	}
	w := a.store
	a.async(func() (func(), error) {
		r, err := work(w)
		if err != nil {
			return nil, err
		}
		s, scanErr := w.Scan()
		var note *workspace.Note
		if openCreated && r.Kind == "note" {
			n, err := w.Read(r.Path)
			if err == nil {
				note = &n
			} else {
				scanErr = err
			}
		}
		return func() {
			// The filesystem operation already succeeded, so remap even if a
			// subsequent rescan failed; never leave a draft at the old path.
			if from != "" {
				a.session.Remap(from, r.Path)
				a.activeFolder = session.Remap(a.activeFolder, from, r.Path)
				pins := a.pinned()
				next := map[string]bool{}
				for p, v := range pins {
					next[session.Remap(p, from, r.Path)] = v
				}
				a.pins[w.Root()] = next
				expanded := map[string]*bool{}
				for p, v := range a.expanded {
					expanded[session.Remap(p, from, r.Path)] = v
				}
				a.expanded = expanded
			}
			if scanErr == nil {
				a.snapshot = s
			}
			if note != nil {
				a.openTab(*note, session.Edit)
			}
			a.form = nil
			a.status = "Updated " + r.Path
			if scanErr != nil {
				a.status = "Updated " + r.Path + "; refresh failed: " + scanErr.Error()
			}
		}, nil
	})
}
func (a *App) submitForm() {
	if a.form == nil || a.busy {
		return
	}
	f := *a.form
	from := ""
	if f.operation == "rename" || f.operation == "move" {
		from = f.path
	}
	a.operation(from, f.operation == "note", func(w *workspace.Workspace) (workspace.Result, error) {
		switch f.operation {
		case "note":
			return w.CreateNote(f.path, f.value, nil)
		case "folder":
			return w.CreateFolder(f.path, f.value)
		case "rename":
			return w.Rename(f.path, f.value)
		default:
			return w.Move(f.path, f.value)
		}
	})
}
func (a *App) movePath(path, to string) {
	a.operation(path, false, func(w *workspace.Workspace) (workspace.Result, error) { return w.Move(path, to) })
}
func (a *App) askDelete(path string) {
	var paths []string
	for _, t := range a.session.Tabs {
		if session.Under(t.Note.Path, path) {
			paths = append(paths, t.Note.Path)
		}
	}
	a.guard(paths, func() {
		w := a.store
		a.async(func() (func(), error) {
			info, err := w.TrashInfo(path)
			return func() {
				if err == nil {
					a.deleting = &info
				}
			}, err
		})
	})
}
func (a *App) deleteConfirmed() {
	if a.deleting == nil || a.busy {
		return
	}
	w, path, trash := a.store, a.deleting.Path, a.trash
	a.async(func() (func(), error) {
		if err := w.Trash(path, trash); err != nil {
			return nil, err
		}
		s, err := w.Scan()
		return func() {
			var paths []string
			for _, t := range a.session.Tabs {
				if session.Under(t.Note.Path, path) {
					paths = append(paths, t.Note.Path)
				}
			}
			a.session.Close(paths)
			a.cleanupDocs()
			for p := range a.pinned() {
				if session.Under(p, path) {
					delete(a.pinned(), p)
				}
			}
			if session.Under(a.activeFolder, path) {
				a.activeFolder = parent(path)
			}
			if err == nil {
				a.snapshot = s
			}
			a.deleting = nil
			a.status = "Moved to Trash: " + path
		}, err
	})
}
func (a *App) reload(d *Document) {
	w, path := a.store, d.Tab.Note.Path
	a.async(func() (func(), error) {
		n, err := w.Read(path)
		if err != nil {
			return nil, err
		}
		return func() {
			d.Tab.Note = n
			d.Tab.Draft = session.Normalize(n.Markdown)
			d.Tab.Conflict = false
			d.Tab.Removed = false
			d.Tab.External = nil
			d.Code.Buffer.Reset(n.Markdown)
			d.Code.Buffer.MarkSaved()
			d.previewReady = false
			d.Unlocked = map[string]string{}
			d.unlockRevision++
			a.status = "Reloaded " + path
		}, nil
	})
}
func (a *App) saveCopy(d *Document) {
	text := d.Code.Buffer.String()
	name := strings.TrimSuffix(filepath.Base(d.Tab.Note.Path), filepath.Ext(d.Tab.Note.Path)) + "-copy.md"
	a.async(func() (func(), error) {
		var path string
		var err error
		if a.chooseSave != nil {
			path, err = a.chooseSave(name)
		} else {
			path, err = mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.win, DefaultPath: name})
		}
		if err != nil || path == "" {
			return nil, err
		}
		// Save copies through a sibling temporary file so a failed write does
		// not truncate an existing destination selected in the save dialog.
		f, err := os.CreateTemp(filepath.Dir(path), ".notes-save-*")
		if err != nil {
			return nil, err
		}
		defer os.Remove(f.Name())
		defer f.Close()
		if _, err = f.WriteString(text); err != nil {
			return nil, err
		}
		if err = f.Sync(); err != nil {
			return nil, err
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
		if err = os.Rename(f.Name(), path); err != nil {
			return nil, err
		}
		return func() { a.status = "Saved copy: " + path }, nil
	})
}

func (a *App) modals(c *ui.Context) {
	s := a.visual(c)
	if p := a.pending; p != nil {
		open := true
		a.dialog(c, &open, "Unsaved changes", func() {
			ui.Text(c, a.t("Save changed notes before continuing?")).FontSize(16).LineHeight(1.5).TextColor(s.secondary)
			ui.Row(c).Wrap().FillWidth().Justify(ui.End).Gap(9).Margin(20, 0, 0, 0).Children(func() {
				if a.actionButton(c, "Cancel", "cancel-unsaved", "").Disabled(a.busy).Clicked() {
					a.pending = nil
				}
				if a.actionButton(c, "Discard", "discard-unsaved", "").Disabled(a.busy).Clicked() {
					a.pending = nil
					p.action()
				}
				if a.actionButton(c, "Save all", "save-unsaved", "primary").Disabled(a.busy).Clicked() {
					a.saveGuard(p)
				}
			})
		})
		if !open && !a.busy {
			a.pending = nil
		}
	}
	if f := a.form; f != nil {
		open := true
		title := map[string]string{"note": "New note", "folder": "New folder", "rename": "Rename", "move": "Move"}[f.operation]
		submit := false
		a.dialog(c, &open, title, func() {
			if f.operation == "move" {
				ui.Text(c, a.t("Destination folder")).FontSize(13).TextColor(s.muted)
				options := []selectOption{{id: "", label: "/"}}
				for _, dir := range a.snapshot.Folders {
					if !session.Under(dir.Path, f.path) {
						options = append(options, selectOption{id: dir.Path, label: dir.Path})
					}
				}
				a.selectControl(c, "path-destination", "Destination folder", f.value, options, func(value string) { f.value = value }).FillWidth().Disabled(a.busy)
			} else {
				input := a.formInput(c, "path-name", "Name", &f.value).Disabled(a.busy).AutoFocus()
				submit = input.Submitted()
			}
			ui.Row(c).FillWidth().Justify(ui.End).Gap(9).Margin(20, 0, 0, 0).Children(func() {
				if a.actionButton(c, "Cancel", "cancel-path", "").Disabled(a.busy).Clicked() {
					a.form = nil
				}
				if a.actionButton(c, "Confirm", "confirm-path", "primary").Disabled(a.busy || (f.operation != "move" && strings.TrimSpace(f.value) == "")).Clicked() {
					submit = true
				}
			})
		})
		if submit && !a.busy && a.form != nil && (f.operation == "move" || strings.TrimSpace(f.value) != "") {
			a.submitForm()
		}
		if !open && !a.busy {
			a.form = nil
		}
	}
	if info := a.deleting; info != nil {
		open := true
		a.dialog(c, &open, "Move to Trash?", func() {
			ui.Text(c, a.t("{name} — {count} nested files", "name", info.Name, "count", fmtCount(info.NestedFiles))).FontSize(16).LineHeight(1.5).TextColor(s.secondary)
			ui.Row(c).FillWidth().Justify(ui.End).Gap(9).Margin(20, 0, 0, 0).Children(func() {
				if a.actionButton(c, "Cancel", "cancel-trash", "").Disabled(a.busy).Clicked() {
					a.deleting = nil
				}
				if a.actionButton(c, "Move to Trash", "confirm-trash", "danger").Disabled(a.busy).Clicked() {
					a.deleteConfirmed()
				}
			})
		})
		if !open && !a.busy {
			a.deleting = nil
		}
	}
}
func fmtCount(n int) string { return fmt.Sprintf("%d", n) }
