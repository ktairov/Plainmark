package session

import (
	"plainmark/internal/workspace"
	"testing"
)

func note(path, text string, revision int64) workspace.Note {
	return workspace.Note{Path: path, Markdown: text, Revision: workspace.Revision{ModifiedMS: revision}}
}
func TestSeparateDraftsExternalChangesAndSave(t *testing.T) {
	s := &Session{}
	a := s.Open(note("a.md", "first", 1), Split)
	b := s.Open(note("b.md", "second", 1), Preview)
	a.Draft = "draft"
	s.Active = a.Note.Path
	if b.Dirty() || !a.Dirty() || s.Current() != a {
		t.Fatal("drafts leaked between tabs")
	}
	external := note("a.md", "external", 2)
	if s.External("a.md", &external) || a.Draft != "draft" || !a.Conflict {
		t.Fatal("external write destroyed draft")
	}
	clean := note("b.md", "updated", 2)
	if !s.External("b.md", &clean) || b.Draft != "updated" {
		t.Fatal("clean document did not update")
	}
	a.Draft = "newer draft"
	a.Saved(note("a.md", "draft", 3), "draft")
	if !a.Dirty() || a.Draft != "newer draft" || a.Conflict {
		t.Fatal("in-flight save lost newer draft")
	}
	s.External("a.md", nil)
	if !a.Removed || a.Draft != "newer draft" {
		t.Fatal("external deletion lost draft")
	}
}
func TestRemapReorderAndClose(t *testing.T) {
	s := &Session{}
	a := s.Open(note("folder/a.md", "a", 1), Edit)
	s.Open(note("folder/b.md", "b", 1), Preview)
	c := s.Open(note("c.md", "c", 1), Split)
	a.Draft = "changed"
	s.Reorder("c.md", "folder/a.md")
	if s.Tabs[0] != c {
		t.Fatal("reorder failed")
	}
	s.Active = "folder/a.md"
	s.Remap("folder", "moved")
	if s.Active != "moved/a.md" || a.Note.Path != "moved/a.md" || a.Draft != "changed" {
		t.Fatal("remap lost session")
	}
	s.Close([]string{"moved/a.md"})
	if s.Active != "moved/b.md" {
		t.Fatal(s.Active)
	}
	s.Close([]string{"moved/b.md", "c.md"})
	if len(s.Tabs) != 0 || s.Active != "" {
		t.Fatal("close all failed")
	}
}
