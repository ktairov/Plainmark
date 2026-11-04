package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
