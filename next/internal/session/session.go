// Package session is the UI-independent state machine for open documents.
package session

import (
	"slices"
	"strings"

	"plainmark/internal/workspace"
)

type Mode int

const (
	Preview Mode = iota
	Edit
	Split
)

type Tab struct {
	Note     workspace.Note
	Draft    string
	Mode     Mode
	External *workspace.Note
	Conflict bool
	Removed  bool
}

func Normalize(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
func (t *Tab) Dirty() bool      { return t.Draft != Normalize(t.Note.Markdown) }

type Session struct {
	Tabs   []*Tab
	Active string
}

func (s *Session) Find(path string) *Tab {
	for _, t := range s.Tabs {
		if t.Note.Path == path {
			return t
		}
	}
	return nil
}
func (s *Session) Current() *Tab { return s.Find(s.Active) }
func (s *Session) Open(n workspace.Note, mode Mode) *Tab {
	if t := s.Find(n.Path); t != nil {
		s.Active = n.Path
		return t
	}
	t := &Tab{Note: n, Draft: Normalize(n.Markdown), Mode: mode}
	s.Tabs = append(s.Tabs, t)
	s.Active = n.Path
	return t
}
func (s *Session) Dirty(paths []string) []*Tab {
	var out []*Tab
	for _, t := range s.Tabs {
		if t.Dirty() && (paths == nil || slices.Contains(paths, t.Note.Path)) {
			out = append(out, t)
		}
	}
	return out
}
func (s *Session) Close(paths []string) {
	index := 0
	for i, t := range s.Tabs {
		if t.Note.Path == s.Active {
			index = i
			break
		}
	}
	s.Tabs = slices.DeleteFunc(s.Tabs, func(t *Tab) bool { return slices.Contains(paths, t.Note.Path) })
	if s.Find(s.Active) == nil {
		s.Active = ""
		if len(s.Tabs) > 0 {
			s.Active = s.Tabs[min(index, len(s.Tabs)-1)].Note.Path
		}
	}
}
func (s *Session) Reorder(from, to string) {
	a, b := -1, -1
	for i, t := range s.Tabs {
		if t.Note.Path == from {
			a = i
		}
		if t.Note.Path == to {
			b = i
		}
	}
	if a < 0 || b < 0 || a == b {
		return
	}
	t := s.Tabs[a]
	s.Tabs = slices.Delete(s.Tabs, a, a+1)
	s.Tabs = slices.Insert(s.Tabs, b, t)
}
func Remap(path, from, to string) string {
	if path == from {
		return to
	}
	if strings.HasPrefix(path, from+"/") {
		return to + path[len(from):]
	}
	return path
}
func Under(path, parent string) bool { return path == parent || strings.HasPrefix(path, parent+"/") }
func (s *Session) Remap(from, to string) {
	for _, t := range s.Tabs {
		t.Note.Path = Remap(t.Note.Path, from, to)
		if t.External != nil {
			t.External.Path = Remap(t.External.Path, from, to)
		}
	}
	s.Active = Remap(s.Active, from, to)
}

// External replaces a clean document, but preserves a dirty one and records a
// conflict. Return true only when the editor should load the new disk content.
func (s *Session) External(path string, n *workspace.Note) bool {
	t := s.Find(path)
	if t == nil {
		return false
	}
	if n != nil && n.Revision == t.Note.Revision {
		return false
	}
	if t.Dirty() {
		t.External = n
		t.Conflict = true
		t.Removed = n == nil
		return false
	}
	if n == nil {
		s.Close([]string{path})
		return false
	}
	t.Note = *n
	t.Draft = Normalize(n.Markdown)
	t.External = nil
	t.Conflict = false
	t.Removed = false
	return true
}
func (t *Tab) Saved(n workspace.Note, submitted string) {
	t.Note = n
	t.External = nil
	t.Conflict = false
	t.Removed = false
	// Draft may have advanced while the submitted revision was on its way to
	// disk. Never replace it with submitted or incorrectly mark it clean.
}
