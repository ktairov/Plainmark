package desktop

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"plainmark/internal/session"
	"plainmark/internal/settings"
)

func TestOwnStateRestoresTabsPreferencesAndLayoutWithoutDocumentData(t *testing.T) {
	a, root := workspaceApp(t)
	a.openNote("b.md", "")
	d := a.current()
	d.Tab.Mode = session.Split
	d.Code.GoTo(editor.Pos{Line: 1, Col: 2})
	a.preferences.Language = "ru"
	a.preferences.ContentFontSize = 21
	a.pinned()["b.md"] = true
	open := true
	a.expanded["folder"] = &open
	a.sidebarWidth = 355
	d.Code.Buffer.Set("private-draft-should-never-be-in-state")
	d.Unlocked["test-payload"] = "private-unlocked-should-never-be-in-state"
	a.crypto = &cryptoForm{doc: d, password: "private-password-should-never-be-in-state"}
	data, err := a.captureState()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private-") || strings.Contains(string(data), "test-payload") {
		t.Fatal("state contains document data or credentials")
	}
	path := filepath.Join(t.TempDir(), "state.json")
	if err = settings.Save(path, data); err != nil {
		t.Fatal(err)
	}
	b, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err = b.readState(path); err != nil {
		t.Fatal(err)
	}
	if err = b.load(root); err != nil {
		t.Fatal(err)
	}
	if len(b.session.Tabs) != 2 || b.session.Active != "b.md" || b.current().Tab.Mode != session.Split || !b.pinned()["b.md"] || !(*b.expanded["folder"]) || b.preferences.Language != "ru" || b.preferences.ContentFontSize != 21 || b.sidebarWidth != 355 {
		t.Fatal("restoration lost metadata")
	}
	if b.current().Code.Buffer.String() != "# Second\nText" || b.current().Tab.Dirty() || len(b.current().Unlocked) != 0 {
		t.Fatal("restoration did not read disk independently")
	}
}

func TestTabModesAndBothScrollPositionsSurviveRestart(t *testing.T) {
	a, root := workspaceApp(t)
	long := "# Long note\n\n" + strings.Repeat("Paragraph with enough text to scroll.\n\n", 180)
	for _, path := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(long), 0600); err != nil {
			t.Fatal(err)
		}
	}
	a.reload(a.current())
	first := a.current()
	tt := ui.NewTester(a.view, 1100, 760)
	tt.Frame()
	if tt.HasText("Ln 1, Col 1") {
		t.Fatal("cursor position is visible in preview")
	}
	tt.Scroll(850, 400, 0, 1250)
	tt.Frame()
	if first.Scroll.Y < 500 {
		t.Fatal("preview did not scroll")
	}
	first.Tab.Mode = session.Edit
	tt.Frame()
	if !tt.HasText("Ln 1, Col 1") {
		t.Fatal("cursor position is hidden in editor")
	}
	first.Code.GoTo(editor.Pos{Line: 70, Col: 3})
	tt.Frame()
	first.Code.SetScrollY(950)
	tt.Frame()
	a.openNote("b.md", "")
	second := a.current()
	second.Tab.Mode = session.Split
	tt.Frame()
	second.Code.SetScrollY(710)
	second.Scroll.Y = 840
	tt.Frame()
	data, err := a.captureState()
	if err != nil {
		t.Fatal(err)
	}
	var expected settings.State
	if err = json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "state.json")
	if err = settings.Save(path, data); err != nil {
		t.Fatal(err)
	}
	b, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err = b.readState(path); err != nil {
		t.Fatal(err)
	}
	if err = b.load(root); err != nil {
		t.Fatal(err)
	}
	restored := ui.NewTester(b.view, 1100, 760)
	restored.Frame()
	restored.Frame()
	for _, saved := range expected.Roots[root].Tabs {
		b.activate(saved.Path)
		restored.Frame()
		restored.Frame()
		d := b.current()
		if d.Tab.Mode != modeValue(saved.Mode) {
			t.Fatalf("%s: mode changed", saved.Path)
		}
		if d.pendingEditorY != nil || math.Abs(d.Code.ScrollY()-saved.EditorY) > 2 {
			t.Fatalf("%s: editor scroll got %v want %v", saved.Path, d.Code.ScrollY(), saved.EditorY)
		}
		if math.Abs(float64(d.Scroll.Y-saved.PreviewY)) > 2 {
			t.Fatalf("%s: preview scroll got %v want %v", saved.Path, d.Scroll.Y, saved.PreviewY)
		}
	}
	b.current().Tab.Mode = session.Preview
	restored.Frame()
	if restored.HasText(b.positionText(b.current().Code.CursorPos().Line+1, b.current().Code.CursorPos().Col+1)) {
		t.Fatal("preview shows line/column")
	}
}

func TestNativeWindowStateUsesOwnStoreAndFlushes(t *testing.T) {
	a, _ := workspaceApp(t)
	states, err := a.LoadWindowStates()
	if err != nil || states != nil {
		t.Fatal("fresh window should use centered defaults")
	}
	path := filepath.Join(t.TempDir(), "state.json")
	a.writer = settings.NewWriter(path)
	defer a.writer.Close()
	want := mygo.WindowState{X: 0, Y: 0, Width: 1040, Height: 720, Maximized: true}
	if err = a.SaveWindowStates(map[string]mygo.WindowState{"main": want}); err != nil {
		t.Fatal(err)
	}
	if err = a.writer.Close(); err != nil {
		t.Fatal(err)
	}
	b, _ := New()
	if err = b.readState(path); err != nil {
		t.Fatal(err)
	}
	states, err = b.LoadWindowStates()
	if err != nil || states["main"] != want || b.windowOptions().StateKey != "main" {
		t.Fatal("window geometry/mode did not survive own store", states, err)
	}
}

func TestMissingPathsAndIntentionallyClosedTabs(t *testing.T) {
	a, root := workspaceApp(t)
	a.openNote("b.md", "")
	a.pinned()["b.md"] = true
	data, err := a.captureState()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "state.json")
	if err = settings.Save(path, data); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(root, "b.md")); err != nil {
		t.Fatal(err)
	}
	b, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err = b.readState(path); err != nil {
		t.Fatal(err)
	}
	if err = b.load(root); err != nil {
		t.Fatal(err)
	}
	if len(b.session.Tabs) != 1 || b.session.Active != "a.md" || b.pinned()["b.md"] {
		t.Fatal("missing tab/pin was not pruned")
	}
	b.session.Close(b.allPaths())
	b.cleanupDocs()
	data, err = b.captureState()
	if err != nil {
		t.Fatal(err)
	}
	if err = settings.Save(path, data); err != nil {
		t.Fatal(err)
	}
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.readState(path); err != nil {
		t.Fatal(err)
	}
	if err = c.load(root); err != nil {
		t.Fatal(err)
	}
	if len(c.session.Tabs) != 0 {
		t.Fatal("intentionally empty session auto-opened a note")
	}
	if err = c.load(filepath.Join(root, "a.md")); err != nil {
		t.Fatal(err)
	}
	if c.current() == nil || c.current().Tab.Note.Path != "a.md" {
		t.Fatal("explicit file argument was ignored")
	}
}
func TestCancelledSettingsPreviewIsNotPersisted(t *testing.T) {
	a, _ := workspaceApp(t)
	before := a.preferences
	a.openSettings()
	a.preferences.Language = "ru"
	a.preferences.ColorMode = "dark"
	data, err := a.captureState()
	if err != nil {
		t.Fatal(err)
	}
	var saved settings.State
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Preferences != before {
		t.Fatal("unapplied preferences were persisted")
	}
}
