package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"sort"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/editor"
	"plainmark/internal/session"
	"plainmark/internal/settings"
	"plainmark/internal/workspace"
)

func modeName(mode session.Mode) string {
	switch mode {
	case session.Edit:
		return "edit"
	case session.Split:
		return "split"
	}
	return "preview"
}
func modeValue(mode string) session.Mode {
	switch mode {
	case "edit":
		return session.Edit
	case "split":
		return session.Split
	}
	return session.Preview
}
func (a *App) rememberRoot() {
	if a.store == nil {
		return
	}
	r := settings.RootState{Tabs: []settings.StoredTab{}, Active: a.session.Active, SelectedFolder: a.activeFolder}
	for _, tab := range a.session.Tabs {
		d := a.docs[tab]
		pos := d.Code.CursorPos()
		editorY := d.Code.ScrollY()
		if d.pendingEditorY != nil {
			editorY = *d.pendingEditorY
		}
		r.Tabs = append(r.Tabs, settings.StoredTab{Path: tab.Note.Path, Mode: modeName(tab.Mode), PreviewY: d.Scroll.Y, EditorY: editorY, CursorLine: pos.Line, CursorColumn: pos.Col})
	}
	for p, on := range a.pinned() {
		if on {
			r.Pinned = append(r.Pinned, p)
		}
	}
	sort.Strings(r.Pinned)
	for p, open := range a.expanded {
		if *open {
			r.Expanded = append(r.Expanded, p)
		}
	}
	sort.Strings(r.Expanded)
	a.saved.Roots[a.store.Root()] = r
}
func (a *App) captureState() ([]byte, error) {
	a.rememberRoot()
	a.saved.Version = settings.StateVersion
	a.saved.Preferences = a.preferences
	if a.settingsOpen {
		a.saved.Preferences = a.beforePreferences
	}
	a.saved.Recent = append([]string(nil), a.recent...)
	if a.store != nil {
		a.saved.Root = a.store.Root()
	}
	a.saved.SidebarVisible, a.saved.SidebarWidth, a.saved.SplitPercent, a.saved.Wrap = a.sidebarVisible, a.sidebarWidth, a.ratio, a.wrap
	return json.MarshalIndent(a.saved, "", "  ")
}
func (a *App) persistState() {
	if a.writer == nil {
		return
	}
	data, err := a.captureState()
	if err != nil {
		a.status = "Settings: " + err.Error()
		return
	}
	if string(data) == a.stateFingerprint {
		return
	}
	if a.writer.Queue(data) {
		a.stateFingerprint = string(data)
	}
}
func (a *App) readState(path string) error {
	s, err := settings.Load(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	a.saved = s
	a.windowStateLoaded = err == nil
	a.preferences = s.Preferences
	a.recent = append([]string(nil), s.Recent...)
	a.sidebarVisible, a.sidebarWidth, a.ratio, a.wrap = s.SidebarVisible, s.SidebarWidth, s.SplitPercent, s.Wrap
	for root, r := range s.Roots {
		pins := map[string]bool{}
		for _, p := range r.Pinned {
			pins[p] = true
		}
		a.pins[root] = pins
	}
	return nil
}
func (a *App) watchStateErrors() {
	writer := a.writer
	if writer == nil {
		return
	}
	a.run(func() {
		for err := range writer.Errors {
			a.update(func() { a.status = "Settings: " + err.Error() })
		}
	})
}
func (a *App) windowOptions() mygo.WindowOptions {
	b := a.saved.Window
	return mygo.WindowOptions{Title: "Plainmark Go", Width: b.Width, Height: b.Height, StateKey: "main", MinWidth: 880, MinHeight: 620, TitleBarStyle: mygo.TitleBarHidden, TitleBarHeight: 44, TrafficLightPosition: &mygo.Point{X: 12, Y: 14}}
}

// MyGo captures settled normal bounds and window mode before OnQuit. Keep its
// state in the same atomic writer as the session, flushed by OnQuit afterwards.
func (a *App) LoadWindowStates() (map[string]mygo.WindowState, error) {
	if !a.windowStateLoaded {
		return nil, nil
	}
	return map[string]mygo.WindowState{"main": mygo.WindowState(a.saved.Window)}, nil
}
func (a *App) SaveWindowStates(states map[string]mygo.WindowState) error {
	if state, ok := states["main"]; ok {
		a.saved.Window = settings.WindowState(state)
		a.windowStateLoaded = true
		a.persistState()
	}
	return nil
}
func (a *App) requestQuit() {
	a.update(func() {
		if a.busy {
			return
		}
		a.guard(nil, func() { a.closeAllowed = true; a.persistState(); mygo.App.Quit() })
	})
}
func (a *App) pumpFiles() {
	if a.win == nil || a.busy || a.pending != nil || a.form != nil || a.crypto != nil || a.settingsOpen || len(a.pendingFiles) == 0 {
		return
	}
	path := a.pendingFiles[0]
	a.pendingFiles = a.pendingFiles[1:]
	a.openAbsolute(path)
}

type loadedWorkspace struct {
	store     *workspace.Workspace
	snapshot  workspace.Snapshot
	initial   *workspace.Note
	rootState settings.RootState
	hasState  bool
	notes     map[string]workspace.Note
	explicit  bool
}

func prepareWorkspace(path string, roots map[string]settings.RootState) (loadedWorkspace, error) {
	w, s, n, err := resolveInitial(path)
	if err != nil {
		return loadedWorkspace{}, err
	}
	state, ok := roots[w.Root()]
	info, err := os.Stat(path)
	if err != nil {
		return loadedWorkspace{}, err
	}
	loaded := loadedWorkspace{store: w, snapshot: s, initial: n, rootState: state, hasState: ok, notes: map[string]workspace.Note{}, explicit: !info.IsDir()}
	if ok {
		for _, t := range state.Tabs {
			note, err := w.Read(t.Path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return loadedWorkspace{}, fmt.Errorf("restore %s: %w", t.Path, err)
			}
			loaded.notes[t.Path] = note
		}
	}
	return loaded, nil
}
func (a *App) restoreRoot(loaded loadedWorkspace) {
	if loaded.hasState {
		for _, tab := range loaded.rootState.Tabs {
			note, ok := loaded.notes[tab.Path]
			if !ok {
				continue
			}
			d := a.openTab(note, modeValue(tab.Mode))
			d.Scroll.Y = tab.PreviewY
			d.Code.GoTo(editor.Pos{Line: tab.CursorLine, Col: tab.CursorColumn})
			y := tab.EditorY
			d.pendingEditorY = &y
		}
		if a.session.Find(loaded.rootState.Active) != nil {
			a.activate(loaded.rootState.Active)
		}
		folders := map[string]bool{}
		for _, f := range loaded.snapshot.Folders {
			folders[f.Path] = true
		}
		for _, p := range loaded.rootState.Expanded {
			if folders[p] {
				open := true
				a.expanded[p] = &open
			}
		}
		if folders[loaded.rootState.SelectedFolder] {
			a.activeFolder = loaded.rootState.SelectedFolder
		}
		known := map[string]bool{}
		for _, n := range loaded.snapshot.Notes {
			known[n.Path] = true
		}
		pins := map[string]bool{}
		for _, p := range loaded.rootState.Pinned {
			if known[p] {
				pins[p] = true
			}
		}
		a.pins[loaded.store.Root()] = pins
	}
	if loaded.initial != nil && (loaded.explicit || !loaded.hasState) {
		a.openTab(*loaded.initial, session.Preview)
	}
}
func (a *App) rootStates() map[string]settings.RootState {
	a.rememberRoot()
	return maps.Clone(a.saved.Roots)
}
