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
	"plainmark/internal/settings"
	"plainmark/internal/workspace"
)

type Document struct {
	Tab                *session.Tab
	Code               *editor.State
	MD                 markdown.Options
	Scroll             ui.ScrollState
	draftVersion       uint64
	draftReady         bool
	previewVersion     uint64
	previewSource      string
	previewReady       bool
	cleanSource        string
	previewDoc         *parser.Document
	previewLines       []int
	listMarkerDigits   int
	splitScroll        splitScrollState
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
	pendingEditorY     *float64
}
type pendingAction struct {
	paths  []string
	action func()
}
type pathForm struct{ operation, path, value string }
type noteRead struct{ newTab bool }

type App struct {
	win                 *mygo.Window
	math                *mathrender.Renderer
	store               *workspace.Workspace
	snapshot            workspace.Snapshot
	session             session.Session
	docs                map[*session.Tab]*Document
	watcher             *workspace.Watcher
	epoch               uint64
	navigationVersion   uint64
	noteReads           map[*noteRead]struct{}
	readingNote         bool
	refreshNeeded       bool
	busy                bool
	status              string
	wrap                bool
	ratio, sidebarWidth float32
	sidebarVisible      bool
	lastVisibleTab      *session.Tab
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
	preferences         settings.Preferences
	beforePreferences   settings.Preferences
	settingsOpen        bool
	settingsSection     string
	settingsPreviewMode string
	windowColorOverride string
	appearanceDark      bool
	resolvedLanguage    string
	nativeLocale        string
	systemDark          bool
	systemDarkKnown     bool
	nativeButtonsKnown  bool
	focusSearch         bool
	saved               settings.State
	writer              *settings.Writer
	windowStateLoaded   bool
	popup               *popupState
	popupReturnID       string
	pendingFiles        []string
}

func New() (*App, error) {
	if err := settings.RegisterFonts(); err != nil {
		return nil, err
	}
	m, err := mathrender.New()
	if err != nil {
		return nil, err
	}
	return &App{math: m, docs: map[*session.Tab]*Document{}, preferences: settings.Defaults(), saved: settings.NewState(), resolvedLanguage: "en", wrap: true, ratio: 50, sidebarWidth: settings.DefaultSidebarWidth, sidebarVisible: true,
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
	// A newer operation takes precedence over an unfinished note navigation.
	a.cancelNavigation()
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
			a.persistState()
			a.pumpFiles()
		})
	})
}
func (a *App) flush() {
	for _, d := range a.docs {
		if version := d.Code.Buffer.Version(); d.draftReady && version == d.draftVersion {
			continue
		}
		// An immutable snapshot keeps text and version from the same revision.
		snapshot := d.Code.Buffer.Snapshot()
		d.Tab.Draft = snapshot.String()
		d.draftVersion, d.draftReady = snapshot.Version(), true
	}
}
func (a *App) openTab(n workspace.Note, mode session.Mode) *Document {
	return a.openTabIn(n, mode, true)
}
func (a *App) openTabIn(n workspace.Note, mode session.Mode, newTab bool) *Document {
	if a.session.Active != n.Path {
		a.lockAll()
	}
	previous := a.session.Current()
	t := a.session.Navigate(n, mode, newTab)
	if previous != nil && a.session.Find(previous.Note.Path) == nil {
		delete(a.docs, previous)
	}
	return a.documentFor(t)
}

func (a *App) documentFor(t *session.Tab) *Document {
	if d := a.docs[t]; d != nil {
		return d
	}
	n := t.Note
	b := editor.NewBuffer(n.Markdown)
	b.SetLanguage(editor.LanguageFor(n.Path))
	b.MarkSaved()
	d := &Document{Tab: t, Code: &editor.State{Buffer: b, SoftWrap: true}, MD: markdown.Options{Extensions: parser.Default | parser.ExtDollarMath}, Unlocked: map[string]string{}}
	d.draftVersion, d.draftReady = b.Version(), true
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
	a.cancelNavigation()
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
func (a *App) applyWorkspace(loaded loadedWorkspace) {
	a.cancelNavigation()
	a.rememberRoot()
	w, s := loaded.store, loaded.snapshot
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
	a.restoreRoot(loaded)
	a.status = "Opened " + w.Root()
}

// Synchronous initialization before the event loop, and in headless tests.
func (a *App) load(path string) error {
	loaded, err := prepareWorkspace(path, a.rootStates())
	if err != nil {
		return err
	}
	a.applyWorkspace(loaded)
	return nil
}
func (a *App) switchTo(path string) {
	a.guard(nil, func() {
		old := a.watcher
		watchEnabled := a.win != nil
		roots := a.rootStates()
		a.async(func() (func(), error) {
			loaded, err := prepareWorkspace(path, roots)
			if err != nil {
				return nil, err
			}
			var watch *workspace.Watcher
			if watchEnabled {
				watch, err = loaded.store.Watch(context.Background())
				if err != nil {
					return nil, err
				}
			}
			if old != nil {
				old.Close()
			}
			return func() { a.applyWorkspace(loaded); a.attachWatch(loaded.store, watch) }, nil
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
	a.navigateNote(path, anchor, true)
}
func (a *App) navigateNote(path, anchor string, newTab bool) {
	if a.busy || a.store == nil {
		return
	}
	a.cancelReplacementReads()
	if t := a.session.Find(path); t != nil {
		a.flush()
		a.activate(path)
		if anchor != "" {
			a.docs[t].MD.Anchor = anchor
			a.docs[t].MD.AnchorVersion++
		}
		return
	}
	a.flush()
	previous := a.current()
	w := a.store
	epoch, version := a.epoch, a.navigationVersion
	request := &noteRead{newTab: newTab}
	if a.noteReads == nil {
		a.noteReads = map[*noteRead]struct{}{}
	}
	a.noteReads[request] = struct{}{}
	a.readingNote = true
	// Reading a note must not disable and dim the toolbar/tree for one frame.
	// Keep the old document interactive until the latest requested read finishes.
	a.run(func() {
		n, err := w.Read(path)
		a.update(func() {
			if _, pending := a.noteReads[request]; !pending || a.epoch != epoch {
				return
			}
			delete(a.noteReads, request)
			a.readingNote = len(a.noteReads) > 0
			defer func() {
				if a.refreshNeeded && !a.readingNote {
					a.refreshNeeded = false
					a.refresh()
				}
				a.persistState()
				a.pumpFiles()
			}()
			if err != nil {
				a.status = err.Error()
				return
			}
			a.flush()
			alreadyOpen := a.session.Find(n.Path) != nil
			var d *Document
			if version == a.navigationVersion {
				d = a.openTabIn(n, session.Preview, newTab || a.current() != previous)
			} else {
				// Earlier explicit new-tab requests still open, but never steal
				// focus from a newer click or an already-open tab selection.
				active := a.session.Active
				tab := a.session.Open(n, session.Preview)
				a.session.Active = active
				d = a.documentFor(tab)
			}
			if version == a.navigationVersion || !alreadyOpen {
				d.MD.Anchor = anchor
				d.MD.AnchorVersion++
			}
			a.status = "Opened " + path
		})
	})
}

func (a *App) cancelNavigation() {
	a.navigationVersion++
	clear(a.noteReads)
	a.readingNote = false
}

func (a *App) cancelReplacementReads() {
	a.navigationVersion++
	for request := range a.noteReads {
		if !request.newTab {
			delete(a.noteReads, request)
		}
	}
	a.readingNote = len(a.noteReads) > 0
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
	if a.busy || a.readingNote {
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
	mygo.App.SetName("Plainmark")
	mygo.App.SetLanguages("en", "ru")
	// Keep receiving OS appearance changes even while the window uses a local palette.
	mygo.Theme.SetSource(mygo.ThemeSystem)
	a, err := New()
	if err != nil {
		return err
	}
	configPath, configErr := settings.ConfigPath()
	writable := configErr == nil
	if configErr == nil {
		configErr = a.readState(configPath)
		writable = configErr == nil
	}
	if configErr != nil {
		a.status = "Settings: " + configErr.Error()
	}
	if path == "" {
		path = os.Getenv("NOTES_ROOT")
	}
	if path == "" {
		path = a.saved.Root
	}
	if path != "" {
		if err = a.load(path); err != nil {
			a.status = err.Error()
		}
	}
	locale := a.preferences.Language
	if locale == "system" {
		locale = ""
	}
	mygo.App.SetLocale(locale)
	mygo.App.SetWindowStateStore(a)
	a.nativeLocale = a.preferences.Language
	mygo.App.OnOpenFile(func(path string) { a.update(func() { a.pendingFiles = append(a.pendingFiles, path); a.pumpFiles() }) })
	var removeModeShortcuts func()
	mygo.App.WhenReady(func() {
		if writable {
			a.writer = settings.NewWriter(configPath)
		}
		options := a.windowOptions()
		options.Content = ui.View(a.view)
		a.win = mygo.NewWindow(options)
		a.installMenu()
		removeModeShortcuts = a.installModeShortcuts()
		a.watchStateWrites()
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
			e.PreventDefault()
			a.requestQuit()
		})
		mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
			if a.closeAllowed {
				return
			}
			e.PreventDefault()
			a.requestQuit()
		})
		a.pumpFiles()
	})
	mygo.App.OnQuit(func() {
		if removeModeShortcuts != nil {
			removeModeShortcuts()
		}
		if a.watcher != nil {
			a.watcher.Close()
		}
		if a.writer != nil {
			if err := a.writer.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "Settings:", err)
			}
		}
	})
	return mygo.App.Run()
}
