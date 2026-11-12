package desktop

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/editor"
	"plainmark/internal/session"
	"plainmark/internal/workspace"
)

func TestDraftSyncTracksInactiveEditsUndoAndSave(t *testing.T) {
	a, _ := workspaceApp(t)
	first := a.current()
	a.openNote("b.md", "")
	second := a.current()
	first.Code.Buffer.Edit(editor.Edit{Start: 0, End: first.Code.Buffer.Len(), Text: "# Changed inactive note"})
	second.Code.Buffer.Edit(editor.Edit{Start: 0, End: second.Code.Buffer.Len(), Text: "# Changed active note"})
	a.flush()
	if first.Tab.Draft != "# Changed inactive note" || second.Tab.Draft != "# Changed active note" {
		t.Fatal("active or inactive edits were not synchronized")
	}
	first.Code.Buffer.Undo()
	a.flush()
	if first.Tab.Dirty() || first.Tab.Draft != session.Normalize(first.Tab.Note.Markdown) {
		t.Fatal("Undo did not synchronize the saved content")
	}
	first.Code.Buffer.Redo()
	a.flush()
	if first.Tab.Draft != "# Changed inactive note" || !first.Tab.Dirty() {
		t.Fatal("Redo did not synchronize the draft")
	}
	version := second.Code.Buffer.Version()
	a.save()
	a.flush()
	if second.Code.Buffer.Version() != version || second.Tab.Dirty() || second.Tab.Draft != "# Changed active note" {
		t.Fatal("saving with an unchanged buffer version lost the draft or dirty state")
	}
}

func TestUnchangedDirtyDraftSyncDoesNotAllocate(t *testing.T) {
	a, _ := workspaceApp(t)
	a.current().Code.Buffer.Set(strings.Repeat("Some markdown text.\n", 500000))
	a.flush()
	if allocations := testing.AllocsPerRun(100, a.flush); allocations != 0 {
		t.Fatalf("unchanged dirty buffer still allocates: %.1f allocations per flush", allocations)
	}
}

func BenchmarkFlushUnchangedDirty10MB(b *testing.B) {
	buffer := editor.NewBuffer("# Original")
	tab := &session.Tab{Note: workspace.Note{Markdown: "# Original"}, Draft: "# Original"}
	doc := &Document{Tab: tab, Code: &editor.State{Buffer: buffer}}
	a := &App{docs: map[*session.Tab]*Document{tab: doc}}
	buffer.Set(strings.Repeat("Some markdown text.\n", 500000))
	a.flush()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		a.flush()
	}
}
