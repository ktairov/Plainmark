package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"plainmark/internal/session"
	"plainmark/internal/workspace"
)

func workspaceApp(t *testing.T) (*App, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "folder"), 0755); err != nil {
		t.Fatal(err)
	}
	for path, src := range map[string]string{"a.md": "# First\nBody", "b.md": "# Second\nText", "folder/child.md": "# Child\nContent"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
	}
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	a.run = func(fn func()) { fn() }
	if err = a.load(root); err != nil {
		t.Fatal(err)
	}
	return a, root
}
func TestTabsKeepBuffersAndUndo(t *testing.T) {
	a, _ := workspaceApp(t)
	first := a.current()
	first.Tab.Mode = session.Edit
	tt := ui.NewTester(a.view, 1220, 820)
	tt.Frame()
	if err := tt.Click("Markdown editor"); err != nil {
		t.Fatal(err)
	}
	tt.Command("selectAll")
	tt.Type("# First draft\nChanged")
	a.openNote("b.md", "")
	second := a.current()
	second.Tab.Mode = session.Edit
	tt.Frame()
	if err := tt.Click("Markdown editor"); err != nil {
		t.Fatal(err)
	}
	tt.Command("selectAll")
	tt.Type("# Second draft\nChanged")
	a.openNote("a.md", "")
	tt.Frame()
	if a.current() != first || !strings.Contains(first.Code.Buffer.String(), "First draft") || !strings.Contains(second.Code.Buffer.String(), "Second draft") {
		t.Fatal("switch lost separate buffers")
	}
	if err := tt.Click("Markdown editor"); err != nil {
		t.Fatal(err)
	}
	tt.Command("undo")
	if first.Code.Buffer.String() != "# First\nBody" {
		t.Fatal("undo history was not retained")
	}
	if !strings.Contains(second.Code.Buffer.String(), "Second draft") {
		t.Fatal("undo modified another tab")
	}
}
func TestGuardCancelDiscardAndSaveAll(t *testing.T) {
	a, root := workspaceApp(t)
	a.current().Code.Buffer.Set("# Draft")
	tt := ui.NewTester(a.view, 1220, 820)
	tt.Frame()
	a.closeTabs([]string{"a.md"})
	tt.Frame()
	if a.pending == nil {
		t.Fatal("dirty close did not ask")
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if a.pending != nil || a.session.Find("a.md") == nil {
		t.Fatal("cancel closed tab")
	}
	a.closeTabs([]string{"a.md"})
	tt.Frame()
	if err := tt.Click("Save all"); err != nil {
		t.Fatal(err)
	}
	if a.session.Find("a.md") != nil || a.pending != nil {
		t.Fatal("save-and-close did not finish")
	}
	data, err := os.ReadFile(filepath.Join(root, "a.md"))
	if err != nil || string(data) != "# Draft" {
		t.Fatalf("save guard: %q %v", data, err)
	}
	a.openNote("b.md", "")
	a.current().Code.Buffer.Set("discard me")
	a.closeTabs([]string{"b.md"})
	tt.Frame()
	if err := tt.Click("Discard"); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(root, "b.md"))
	if err != nil || string(data) != "# Second\nText" {
		t.Fatal("discard wrote draft")
	}
	if len(a.session.Tabs) != 0 {
		t.Fatal("discard failed to close")
	}
}
func TestRenameRemapsDraftAndPinnedPaths(t *testing.T) {
	a, _ := workspaceApp(t)
	a.openNote("folder/child.md", "")
	d := a.current()
	d.Code.Buffer.Set("# Unsaved child")
	a.pinned()["folder/child.md"] = true
	open := true
	a.expanded["folder"] = &open
	a.activeFolder = "folder"
	a.operation("folder", false, func(w *workspace.Workspace) (workspace.Result, error) { return w.Rename("folder", "renamed") })
	if a.current() != d || d.Tab.Note.Path != "renamed/child.md" || d.Code.Buffer.String() != "# Unsaved child" || !a.pinned()["renamed/child.md"] || a.activeFolder != "renamed" {
		t.Fatal("rename lost draft or paths")
	}
	a.save()
	n, err := a.store.Read("renamed/child.md")
	if err != nil || n.Markdown != "# Unsaved child" {
		t.Fatal(err)
	}
}
func TestExternalChangesAndDeletionKeepDirtyTabs(t *testing.T) {
	a, root := workspaceApp(t)
	first := a.current()
	a.openNote("b.md", "")
	second := a.current()
	second.Code.Buffer.Set("# Local draft")
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# External clean"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.md"), []byte("# External conflict"), 0600); err != nil {
		t.Fatal(err)
	}
	a.refresh()
	if first.Code.Buffer.String() != "# External clean" {
		t.Fatal("clean tab did not reload")
	}
	if second.Code.Buffer.String() != "# Local draft" || !second.Tab.Conflict {
		t.Fatal("dirty tab lost draft")
	}
	if err := os.Remove(filepath.Join(root, "b.md")); err != nil {
		t.Fatal(err)
	}
	a.refresh()
	if !second.Tab.Removed || second.Code.Buffer.String() != "# Local draft" {
		t.Fatal("deletion lost dirty tab")
	}
}
func TestWorkspaceSwitchProtectsDrafts(t *testing.T) {
	a, _ := workspaceApp(t)
	original := a.store.Root()
	a.current().Code.Buffer.Set("unsaved")
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "other.md"), []byte("# Other"), 0600); err != nil {
		t.Fatal(err)
	}
	a.switchTo(other)
	if a.pending == nil || a.store.Root() != original {
		t.Fatal("switch did not protect drafts")
	}
	p := a.pending
	a.pending = nil
	p.action()
	if a.store.Root() == original || a.session.Find("other.md") == nil {
		t.Fatal("confirmed switch failed")
	}
}
func TestFormCreatesAndTrashProtectsDraft(t *testing.T) {
	a, _ := workspaceApp(t)
	a.form = &pathForm{operation: "note", path: "folder", value: "new"}
	a.submitForm()
	if a.current().Tab.Note.Path != "folder/new.md" {
		t.Fatal("new note did not open")
	}
	a.current().Code.Buffer.Set("draft")
	a.askDelete("folder/new.md")
	if a.pending == nil {
		t.Fatal("delete did not guard draft")
	}
	p := a.pending
	a.pending = nil
	p.action()
	if a.deleting == nil {
		t.Fatal("trash info missing")
	}
	a.trash = func(path string) error { return os.Rename(path, filepath.Join(t.TempDir(), "trashed.md")) }
	a.deleteConfirmed()
	if a.session.Find("folder/new.md") != nil {
		t.Fatal("trashed tab remained")
	}
}

func TestObsoleteReadCannotReplaceNewWorkspace(t *testing.T) {
	a, _ := workspaceApp(t)
	var queue []func()
	a.run = func(fn func()) { queue = append(queue, fn) }
	a.openNote("b.md", "")
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "other.md"), []byte("# Other"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.load(other); err != nil {
		t.Fatal(err)
	}
	if len(queue) != 1 {
		t.Fatal("read not queued")
	}
	queue[0]()
	if a.current().Tab.Note.Path != "other.md" || a.session.Find("b.md") != nil || a.busy {
		t.Fatal("obsolete read changed new workspace")
	}
}

func TestFileDragMovesAndTabDragReorders(t *testing.T) {
	a, _ := workspaceApp(t)
	open := true
	a.expanded["folder"] = &open
	if _, err := a.store.CreateFolder("", "destination"); err != nil {
		t.Fatal(err)
	}
	a.refresh()
	tt := ui.NewTester(a.view, 1220, 820)
	tt.Frame()
	source, ok := tt.Find("Child")
	if !ok {
		t.Fatal("missing child row")
	}
	target, ok := tt.Find("destination")
	if !ok {
		t.Fatal("missing destination")
	}
	tt.Press(source.X+source.W/2, source.Y+source.H/2)
	tt.Move(source.X+source.W/2+10, source.Y+source.H/2+10)
	tt.Move(target.X+target.W/2, target.Y+target.H/2)
	tt.Release(target.X+target.W/2, target.Y+target.H/2)
	if _, err := a.store.Read("destination/child.md"); err != nil {
		t.Fatal("file drag did not move", err)
	}
	a.openNote("b.md", "")
	tt.Frame()
	source, ok = tt.Find("Tab Second")
	if !ok {
		t.Fatal("missing second tab")
	}
	target, ok = tt.Find("Tab First")
	if !ok {
		t.Fatal("missing first tab")
	}
	tt.Press(source.X+source.W/2, source.Y+source.H/2)
	tt.Move(source.X+source.W/2+10, source.Y+source.H/2+10)
	tt.Move(target.X+target.W/2, target.Y+target.H/2)
	tt.Release(target.X+target.W/2, target.Y+target.H/2)
	if a.session.Tabs[0].Note.Path != "b.md" {
		t.Fatal("tab reorder failed")
	}
}

func TestConflictSaveCopyDoesNotOverwriteOriginal(t *testing.T) {
	a, root := workspaceApp(t)
	d := a.current()
	d.Code.Buffer.Set("# Local draft")
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# External"), 0600); err != nil {
		t.Fatal(err)
	}
	a.refresh()
	copyPath := filepath.Join(t.TempDir(), "copy.md")
	a.chooseSave = func(string) (string, error) { return copyPath, nil }
	a.saveCopy(d)
	copy, err := os.ReadFile(copyPath)
	if err != nil || string(copy) != "# Local draft" {
		t.Fatalf("copy: %q %v", copy, err)
	}
	original, err := os.ReadFile(filepath.Join(root, "a.md"))
	if err != nil || string(original) != "# External" || !d.Tab.Conflict || !d.Tab.Dirty() {
		t.Fatal("save copy changed original or discarded draft")
	}
}
