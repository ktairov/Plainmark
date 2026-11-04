package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"

	"plainmark/internal/workspace"
)

const StateVersion = 1

var ErrVersion = errors.New("unsupported settings version")

// Stored state contains navigation/layout only. Document bodies, drafts,
// plaintext, passwords and encryption keys deliberately have no fields here.
type StoredTab struct {
	Path         string  `json:"path"`
	Mode         string  `json:"mode"`
	PreviewY     float32 `json:"previewY,omitempty"`
	EditorY      float64 `json:"editorY,omitempty"`
	CursorLine   int     `json:"cursorLine,omitempty"`
	CursorColumn int     `json:"cursorColumn,omitempty"`
}
type RootState struct {
	Tabs           []StoredTab `json:"tabs"`
	Active         string      `json:"active"`
	Pinned         []string    `json:"pinned"`
	Expanded       []string    `json:"expanded"`
	SelectedFolder string      `json:"selectedFolder,omitempty"`
}
type WindowState struct {
	X          int  `json:"x"`
	Y          int  `json:"y"`
	Width      int  `json:"width"`
	Height     int  `json:"height"`
	Maximized  bool `json:"maximized,omitempty"`
	FullScreen bool `json:"fullScreen,omitempty"`
}
type State struct {
	Version        int                  `json:"version"`
	Preferences    Preferences          `json:"preferences"`
	Root           string               `json:"root"`
	Recent         []string             `json:"recent"`
	Roots          map[string]RootState `json:"roots"`
	SidebarVisible bool                 `json:"sidebarVisible"`
	SidebarWidth   float32              `json:"sidebarWidth"`
	SplitPercent   float32              `json:"splitPercent"`
	Wrap           bool                 `json:"wrap"`
	Window         WindowState          `json:"window"`
}

func NewState() State {
	return State{Version: StateVersion, Preferences: Defaults(), Roots: map[string]RootState{}, SidebarVisible: true, SidebarWidth: 340, SplitPercent: 50, Wrap: true, Window: WindowState{Width: 1220, Height: 820}}
}
func ConfigPath() (string, error) {
	if dir := os.Getenv("PLAINMARK_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "state.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Plainmark-Go", "state.json"), nil
}
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return NewState(), err
	}
	s := NewState()
	if err = json.Unmarshal(data, &s); err != nil {
		return NewState(), err
	}
	if s.Version != StateVersion {
		return NewState(), fmt.Errorf("%w: %d", ErrVersion, s.Version)
	}
	s.Normalize()
	return s, nil
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func paths(input []string) []string {
	var out []string
	for _, p := range input {
		if workspace.ValidateRelative(p, false) == nil && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}
func (s *State) Normalize() {
	s.Preferences.Normalize()
	if s.Roots == nil {
		s.Roots = map[string]RootState{}
	}
	for root, r := range s.Roots {
		if !filepath.IsAbs(root) {
			delete(s.Roots, root)
			continue
		}
		seen := map[string]bool{}
		tabs := []StoredTab{}
		for _, tab := range r.Tabs {
			if workspace.ValidateRelative(tab.Path, false) != nil || !workspace.IsMarkdown(tab.Path) || seen[tab.Path] {
				continue
			}
			seen[tab.Path] = true
			if !slices.Contains([]string{"preview", "edit", "split"}, tab.Mode) {
				tab.Mode = "preview"
			}
			if !finite(float64(tab.PreviewY)) || tab.PreviewY < 0 {
				tab.PreviewY = 0
			}
			if !finite(tab.EditorY) || tab.EditorY < 0 {
				tab.EditorY = 0
			}
			tab.CursorLine = max(0, tab.CursorLine)
			tab.CursorColumn = max(0, tab.CursorColumn)
			tabs = append(tabs, tab)
		}
		r.Tabs = tabs
		r.Pinned = paths(r.Pinned)
		r.Expanded = paths(r.Expanded)
		if workspace.ValidateRelative(r.SelectedFolder, true) != nil {
			r.SelectedFolder = ""
		}
		if !seen[r.Active] {
			r.Active = ""
			if len(tabs) > 0 {
				r.Active = tabs[0].Path
			}
		}
		s.Roots[root] = r
	}
	if !filepath.IsAbs(s.Root) {
		s.Root = ""
	}
	var recent []string
	for _, p := range s.Recent {
		if filepath.IsAbs(p) && !slices.Contains(recent, p) && len(recent) < 5 {
			recent = append(recent, p)
		}
	}
	s.Recent = recent
	if !finite(float64(s.SidebarWidth)) {
		s.SidebarWidth = 340
	}
	s.SidebarWidth = min(max(s.SidebarWidth, 240), 800)
	if !finite(float64(s.SplitPercent)) {
		s.SplitPercent = 50
	}
	s.SplitPercent = min(max(s.SplitPercent, 10), 90)
	s.Window.Width = min(max(s.Window.Width, 880), 16000)
	s.Window.Height = min(max(s.Window.Height, 620), 16000)
}

func Save(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".state-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = f.Chmod(0600); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
