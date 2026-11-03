package desktop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/markdown/parser"

	"plainmark/internal/mathrender"
	"plainmark/internal/preview"
	"plainmark/internal/session"
	"plainmark/internal/workspace"
)

type Document struct {
	Tab                *session.Tab
	Code               *editor.State
	MD                 markdown.Options
	Scroll             ui.ScrollState
	previewVersion     uint64
	previewSource      string
	previewReady       bool
	cleanSource        string
	previewDoc         *parser.Document
	FindOpen           bool
	Query              string
	Match              int
	Matches            []preview.Match
	lastQuery          string
	lastMatch          int
	searchRevision     uint64
	scrolledRevision   uint64
	Unlocked           map[string]string
	unlockRevision     uint64
	lastUnlockRevision uint64
}
type pendingAction struct {
	paths  []string
	action func()
}
type pathForm struct{ operation, path, value string }

type App struct {
	win                 *mygo.Window
	math                *mathrender.Renderer
	store               *workspace.Workspace
	snapshot            workspace.Snapshot
	session             session.Session
	docs                map[*session.Tab]*Document
	watcher             *workspace.Watcher
	epoch               uint64
	refreshNeeded       bool
	busy                bool
	status              string
	wrap                bool
	ratio, sidebarWidth float32
	sidebarVisible      bool
	query, activeFolder string
	expanded            map[string]*bool
	pins                map[string]map[string]bool
	recent              []string
	pending             *pendingAction
	form                *pathForm
	deleting            *workspace.TrashInfo
	closeAllowed        bool
	menuState           string
	run                 func(func())
	trash               func(string) error
	chooseSave          func(string) (string, error)
	crypto              *cryptoForm
}

func New() (*App, error) {
	m, err := mathrender.New()
	if err != nil {
		return nil, err
	}
	return &App{math: m, docs: map[*session.Tab]*Document{}, wrap: true, ratio: 50, sidebarWidth: 280, sidebarVisible: true,
		expanded: map[string]*bool{}, pins: map[string]map[string]bool{}, status: "Open a Markdown folder or file to begin",
		run: func(fn func()) { go fn() }, trash: func(path string) error { return mygo.Shell.TrashItem(path) }}, nil
}
func (a *App) current() *Document { return a.docs[a.session.Current()] }
func (a *App) update(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
	} else {
		fn()
	}
}

// State belongs to the UI thread. Workers capture inputs and return closures
// applied there; an obsolete workspace generation cannot mutate a new session.
func (a *App) async(work func() (func(), error)) {
	if a.busy {
		return
	}
	a.busy = true
	epoch := a.epoch
	a.run(func() {
		apply, err := work()
		a.update(func() {
			if a.epoch != epoch {
				return
			}
			a.busy = false
			if err != nil {
				a.status = err.Error()
			}
			if apply != nil {
				apply()
			}
			if a.refreshNeeded && !a.busy {
				a.refreshNeeded = false
				a.refresh()
			}
		})
	})
}
func (a *App) flush() {
	for _, d := range a.docs {
		if d.Code.Buffer.Modified() {
			d.Tab.Draft = d.Code.Buffer.String()
		} else {
			d.Tab.Draft = session.Normalize(d.Tab.Note.Markdown)
		}
	}
}
func (a *App) openTab(n workspace.Note, mode session.Mode) *Document {
	if a.session.Active != n.Path {
		a.lockAll()
	}
	t := a.session.Open(n, mode)
	if d := a.docs[t]; d != nil {
		return d
	}
	b := editor.NewBuffer(n.Markdown)
	b.SetLanguage(editor.LanguageFor(n.Path))
	b.MarkSaved()
	d := &Document{Tab: t, Code: &editor.State{Buffer: b, SoftWrap: true}, MD: markdown.Options{Extensions: parser.Default | parser.ExtDollarMath}, Unlocked: map[string]string{}}
	a.docs[t] = d
	return d
}
func (a *App) setDocument(path string, data []byte) {
	n := workspace.Note{Path: path, Markdown: string(data), Title: workspace.TitleFor(path, string(data), nil)}
	a.session = session.Session{}
	a.docs = map[*session.Tab]*Document{}
	a.openTab(n, session.Split)
}
func (a *App) cleanupDocs() {
	for t := range a.docs {
		if a.session.Find(t.Note.Path) != t {
			delete(a.docs, t)
		}
	}
}
func (a *App) closeTabs(paths []string) {
	a.guard(paths, func() { a.session.Close(paths); a.cleanupDocs() })
}
func (a *App) allPaths() []string {
	var paths []string
	for _, t := range a.session.Tabs {
		paths = append(paths, t.Note.Path)
	}
	return paths
}
func (a *App) guard(paths []string, action func()) {
	if a.busy {
		return
	}
	a.flush()
	if len(a.session.Dirty(paths)) == 0 {
		action()
		return
	}
	a.pending = &pendingAction{paths, action}
}

func resolveInitial(path string) (*workspace.Workspace, workspace.Snapshot, *workspace.Note, error) {
	full, err := filepath.Abs(path)
	if err != nil {
		return nil, workspace.Snapshot{}, nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return nil, workspace.Snapshot{}, nil, err
	}
	root := full
	if !info.IsDir() {
		root = filepath.Dir(full)
		if !workspace.IsMarkdown(full) {
			return nil, workspace.Snapshot{}, nil, errors.New("Only Markdown files can be opened")
		}
	}
	w, err := workspace.Open(root)
	if err != nil {
		return nil, workspace.Snapshot{}, nil, err
	}
	s, err := w.Scan()
	if err != nil {
		return nil, s, nil, err
	}
	var n *workspace.Note
	if !info.IsDir() {
		note, err := w.Read(filepath.Base(full))
		if err != nil {
			return nil, s, nil, err
		}
		n = &note
	} else if len(s.Notes) > 0 {
		note, err := w.Read(s.Notes[0].Path)
		if err != nil {
			return nil, s, nil, err
		}
		n = &note
	}
	return w, s, n, nil
}
func (a *App) applyWorkspace(w *workspace.Workspace, s workspace.Snapshot, n *workspace.Note) {
	a.epoch++
	a.busy = false
	a.refreshNeeded = false
	a.store = w
	a.snapshot = s
	a.session = session.Session{}
	a.docs = map[*session.Tab]*Document{}
	a.activeFolder = ""
	a.query = ""
	a.expanded = map[string]*bool{}
	if a.pins[w.Root()] == nil {
		a.pins[w.Root()] = map[string]bool{}
	}
	recent := []string{w.Root()}
	for _, p := range a.recent {
		if p != w.Root() && len(recent) < 5 {
			recent = append(recent, p)
		}
	}
	a.recent = recent
	if n != nil {
		a.openTab(*n, session.Preview)
	}
	a.status = "Opened " + w.Root()
}

// Synchronous initialization before the event loop, and in headless tests.
func (a *App) load(path string) error {
	w, s, n, err := resolveInitial(path)
	if err != nil {
		return err
	}
	a.applyWorkspace(w, s, n)
	return nil
}
func (a *App) switchTo(path string) {
	a.guard(nil, func() {
		old := a.watcher
		watchEnabled := a.win != nil
		a.async(func() (func(), error) {
			w, s, n, err := resolveInitial(path)
			if err != nil {
				return nil, err
			}
			var watch *workspace.Watcher
			if watchEnabled {
				watch, err = w.Watch(context.Background())
				if err != nil {
					return nil, err
				}
			}
			if old != nil {
				old.Close()
			}
			return func() { a.applyWorkspace(w, s, n); a.attachWatch(w, watch) }, nil
		})
	})
}
func (a *App) openAbsolute(path string) {
	full, err := filepath.Abs(path)
	if err != nil {
		a.status = err.Error()
		return
	}
	if a.store != nil {
		rel, err := filepath.Rel(a.store.Root(), full)
		rel = filepath.ToSlash(rel)
		if err == nil && workspace.ValidateRelative(rel, false) == nil {
			a.openNote(rel, "")
			return
		}
	}
	a.switchTo(full)
}
func (a *App) openNote(path, anchor string) {
	if a.busy || a.store == nil {
		return
	}
	if t := a.session.Find(path); t != nil {
		a.flush()
		a.activate(path)
		if anchor != "" {
			a.docs[t].MD.Anchor = anchor
			a.docs[t].MD.AnchorVersion++
		}
		return
	}
	w := a.store
	a.async(func() (func(), error) {
		n, err := w.Read(path)
		if err != nil {
			return nil, err
		}
		return func() {
			d := a.openTab(n, session.Preview)
			d.MD.Anchor = anchor
			d.MD.AnchorVersion++
			a.status = "Opened " + path
		}, nil
	})
}
func (a *App) chooseFolder() {
	a.async(func() (func(), error) {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Directory: true})
		if err != nil || len(paths) == 0 {
			return nil, err
		}
		return func() { a.switchTo(paths[0]) }, nil
	})
}
func (a *App) chooseFile() {
	a.async(func() (func(), error) {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Filters: []mygo.FileFilter{{Name: "Markdown", Extensions: []string{"md", "mdx"}}}})
		if err != nil || len(paths) == 0 {
			return nil, err
		}
		return func() { a.openAbsolute(paths[0]) }, nil
	})
}

func (a *App) attachWatch(w *workspace.Workspace, watch *workspace.Watcher) {
	a.watcher = watch
	if watch == nil {
		return
	}
	a.run(func() {
		for {
			select {
			case _, ok := <-watch.Changes:
				if !ok {
					return
				}
				a.update(func() {
					if a.store == w {
						if a.busy {
							a.refreshNeeded = true
						} else {
							a.refresh()
						}
					}
				})
			case err, ok := <-watch.Errors:
				if !ok {
					return
				}
				a.update(func() {
					if a.store == w {
						a.status = "Watcher: " + err.Error()
					}
				})
			}
		}
	})
}
func (a *App) refresh() {
	if a.store == nil {
		return
	}
	if a.busy {
		a.refreshNeeded = true
		return
	}
	a.flush()
	w := a.store
	paths := a.allPaths()
	a.async(func() (func(), error) {
		s, err := w.Scan()
		if err != nil {
			return nil, err
		}
		found := map[string]*workspace.Note{}
		for _, path := range paths {
			n, err := w.Read(path)
			if err == nil {
				found[path] = &n
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
		return func() {
			a.flush()
			a.snapshot = s
			for _, path := range paths {
				if a.session.External(path, found[path]) {
					t := a.session.Find(path)
					d := a.docs[t]
					d.Code.Buffer.Reset(t.Note.Markdown)
					d.Code.Buffer.MarkSaved()
					d.previewReady = false
					d.Unlocked = map[string]string{}
					d.unlockRevision++
				}
			}
			a.cleanupDocs()
			a.status = "Workspace refreshed"
		}, nil
	})
}

func (a *App) save() {
	if d := a.current(); d != nil {
		a.saveDocument(d, false, nil)
	}
}
func (a *App) saveDocument(d *Document, force bool, done func(error)) {
	if a.busy || a.store == nil {
		return
	}
	a.flush()
	w, n := a.store, d.Tab.Note
	var content bytes.Buffer
	_, err := d.Code.Buffer.Snapshot().WriteTo(&content)
	if err != nil {
		a.status = err.Error()
		return
	}
	submitted := content.String()
	a.async(func() (func(), error) {
		saved, err := w.Save(n.Path, submitted, n.Revision, force)
		if err != nil {
			return func() {
				if workspace.IsCode(err, "conflict") || errors.Is(err, os.ErrNotExist) {
					d.Tab.Conflict = true
					d.Tab.Removed = errors.Is(err, os.ErrNotExist)
				}
				if done != nil {
					done(err)
				}
			}, fmt.Errorf("Save: %w", err)
		}
		return func() {
			a.flush()
			d.Tab.Saved(saved, session.Normalize(submitted))
			d.Unlocked = map[string]string{}
			d.unlockRevision++
			for i := range a.snapshot.Notes {
				entry := &a.snapshot.Notes[i]
				if entry.Path == saved.Path {
					entry.Title = saved.Title
					entry.Headings = saved.Headings
					entry.ModifiedMS = saved.Revision.ModifiedMS
					entry.Size = saved.Size
				}
			}
			if d.Code.Buffer.String() == session.Normalize(submitted) {
				d.Code.Buffer.MarkSaved()
			}
			a.status = "Saved " + n.Path
			if done != nil {
				done(nil)
			}
		}, nil
	})
}
func (a *App) saveGuard(p *pendingAction) {
	a.flush()
	dirty := a.session.Dirty(p.paths)
	if len(dirty) == 0 {
		a.pending = nil
		p.action()
		return
	}
	d := a.docs[dirty[0]]
	a.saveDocument(d, false, func(err error) {
		if err == nil {
			a.saveGuard(p)
		} else {
			a.pending = nil
		}
	})
}

func Run(path string) error {
	a, err := New()
	if err != nil {
		return err
	}
	if path != "" {
		if err = a.load(path); err != nil {
			return err
		}
	}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "Plainmark — Go", Width: 1220, Height: 820, MinWidth: 880, MinHeight: 620, Content: ui.View(a.view)})
		a.installMenu()
		if a.store != nil {
			w := a.store
			a.async(func() (func(), error) {
				watch, err := w.Watch(context.Background())
				return func() { a.attachWatch(w, watch) }, err
			})
		}
		a.win.OnClose(func(e *mygo.CloseEvent) {
			if a.closeAllowed {
				return
			}
			if a.busy {
				e.PreventDefault()
				return
			}
			a.flush()
			if len(a.session.Dirty(nil)) > 0 {
				e.PreventDefault()
				a.guard(nil, func() { a.closeAllowed = true; a.win.Close() })
			}
		})
		mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
			if a.closeAllowed {
				return
			}
			if a.busy {
				e.PreventDefault()
				return
			}
			a.flush()
			if len(a.session.Dirty(nil)) > 0 {
				e.PreventDefault()
				a.guard(nil, func() { a.closeAllowed = true; mygo.App.Quit() })
			}
		})
	})
	mygo.App.OnQuit(func() {
		if a.watcher != nil {
			a.watcher.Close()
		}
	})
	return mygo.App.Run()
}
