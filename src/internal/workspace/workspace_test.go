package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) *Workspace {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "folder"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "folder", "note.md"), []byte("# Original\nBody"), 0640); err != nil {
		t.Fatal(err)
	}
	w, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return w
}
func TestHeadingsAndTitles(t *testing.T) {
	got := ParseHeadings("# One\n## Two\n\n```md\n# Not a heading\n```")
	if !reflect.DeepEqual(got, []Heading{{1, "One"}, {2, "Two"}}) {
		t.Fatal(got)
	}
	for _, tc := range []struct{ in, want string }{
		{"Engineering --- Specification", "Engineering — Specification"},
		{"Version 1 -- draft...", "Version 1 – draft…"},
		{"**Bold** and [linked](https://example.com) `code`", "Bold and linked code"},
		{"Title&nbsp; <small>internal</small>   text", "Title internal text"},
	} {
		if got := DisplayTitle(tc.in); got != tc.want {
			t.Errorf("%q -> %q", tc.in, got)
		}
	}
	if got := TitleFor("fallback.md", "# **Tree** --- Window", nil); got != "Tree — Window" {
		t.Fatal(got)
	}
}
func TestPathSecurity(t *testing.T) {
	w := fixture(t)
	for _, p := range []string{"../x", "/tmp/x", "folder/../note.md", "folder//note.md", ".git/config", "node_modules/x", "folder\\note.md", "."} {
		if _, err := w.Absolute(p); err == nil {
			t.Errorf("accepted %q", p)
		}
	}
	if _, err := w.CreateFolder("", ".x"); err == nil {
		t.Fatal("hidden name accepted")
	}
	if _, err := w.CreateFolder("", "a/b"); err == nil {
		t.Fatal("separator accepted")
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "note.md"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(w.Root(), "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Read("escape/note.md"); !IsCode(err, "pathOutsideWorkspace") {
		t.Fatal(err)
	}
	if _, err := w.CreateNote("escape", "new", nil); !IsCode(err, "pathOutsideWorkspace") {
		t.Fatal(err)
	}
	if _, err := Open(filepath.Join(w.Root(), "folder/note.md")); !IsCode(err, "notAFolder") {
		t.Fatal(err)
	}
}
func TestCRUDAndNoOverwrite(t *testing.T) {
	w := fixture(t)
	if _, err := w.CreateFolder("", "destination"); err != nil {
		t.Fatal(err)
	}
	title := "Title"
	r, err := w.CreateNote("folder", "new", &title)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.CreateNote("folder", "new.md", nil); !errors.Is(err, os.ErrExist) {
		t.Fatal(err)
	}
	if _, err = w.Rename(r.Path, "note.md"); !IsCode(err, "itemExists") {
		t.Fatal(err)
	}
	r, err = w.Rename(r.Path, "renamed")
	if err != nil {
		t.Fatal(err)
	}
	r, err = w.Move(r.Path, "destination")
	if err != nil {
		t.Fatal(err)
	}
	n, err := w.Read(r.Path)
	if err != nil || r.Path != "destination/renamed.md" || n.Markdown != "# Title\n" {
		t.Fatalf("%+v %+v %v", r, n, err)
	}
	if _, err = w.CreateFolder("folder", "child"); err != nil {
		t.Fatal(err)
	}
	if _, err = w.Move("folder", "folder/child"); !IsCode(err, "moveIntoDescendant") {
		t.Fatal(err)
	}
	if _, err = w.Rename(r.Path, "bad.txt"); !IsCode(err, "notMarkdown") {
		t.Fatal(err)
	}
}
func TestScanFoldersExclusionsAndSymlinks(t *testing.T) {
	w := fixture(t)
	for _, p := range []string{"folder/child/empty", ".hidden", "node_modules", "target", "dist"} {
		if err := os.MkdirAll(filepath.Join(w.Root(), p), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"alpha.md", "UPPER.MDX", ".hidden/no.md", "node_modules/no.md", "target/no.md", "dist/no.md"} {
		if err := os.WriteFile(filepath.Join(w.Root(), p), []byte("# Note"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(w.Root(), "folder"), filepath.Join(w.Root(), "alias")); err != nil {
		t.Fatal(err)
	}
	s, err := w.Scan()
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, n := range s.Notes {
		paths = append(paths, n.Path)
	}
	if !reflect.DeepEqual(paths, []string{"UPPER.MDX", "alpha.md", "folder/note.md"}) {
		t.Fatal(paths)
	}
	if len(s.Folders) != 3 || s.Folders[2].Path != "folder/child/empty" {
		t.Fatal(s.Folders)
	}
}
func TestSaveConflictsRapidRevisionsAndPermissions(t *testing.T) {
	w := fixture(t)
	n, err := w.Read("folder/note.md")
	if err != nil {
		t.Fatal(err)
	}
	large := "# Large\n" + strings.Repeat("content line\n", 150000)
	saved, err := w.Save(n.Path, large, n.Revision, false)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision.ModifiedMS <= n.Revision.ModifiedMS {
		t.Fatal("mtime did not advance")
	}
	if _, err = w.Save(n.Path, "stale", n.Revision, false); !IsCode(err, "conflict") {
		t.Fatal(err)
	}
	third, err := w.Save(saved.Path, "# Immediate revision", saved.Revision, false)
	if err != nil {
		t.Fatal(err)
	}
	if third.Markdown != "# Immediate revision" {
		t.Fatal("lost revision")
	}
	path := filepath.Join(w.Root(), "folder/note.md")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Fatal("permissions changed")
	}
	if err = os.WriteFile(path, []byte("external"), 0640); err != nil {
		t.Fatal(err)
	}
	stamp := time.UnixMilli(third.Revision.ModifiedMS)
	if err = os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err = w.Save(third.Path, "local", third.Revision, false); !IsCode(err, "conflict") {
		t.Fatal("missed same-mtime external write", err)
	}
	forced, err := w.Save(third.Path, "forced", third.Revision, true)
	if err != nil || forced.Markdown != "forced" {
		t.Fatal(err)
	}
	matches, err := filepath.Glob(filepath.Join(w.Root(), "folder/.notes-save-*"))
	if err != nil || len(matches) != 0 {
		t.Fatal("leaked temporary files", matches, err)
	}
}
func TestTrashInfoAndDelegation(t *testing.T) {
	w := fixture(t)
	if err := os.WriteFile(filepath.Join(w.Root(), "folder/t.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := w.TrashInfo("folder")
	if err != nil || info.NestedFiles != 2 || !info.IsDirectory {
		t.Fatalf("%+v %v", info, err)
	}
	called := false
	err = w.Trash("folder", func(path string) error { called = true; return os.Rename(path, filepath.Join(t.TempDir(), "trashed")) })
	if err != nil || !called {
		t.Fatal(err)
	}
	if _, err = w.TrashInfo(""); err == nil {
		t.Fatal("accepted root trash")
	}
}

func receive(t *testing.T, w *Watcher) []string {
	t.Helper()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case paths, ok := <-w.Changes:
		if !ok {
			t.Fatal("watch closed")
		}
		return paths
	case err := <-w.Errors:
		t.Fatal(err)
	case <-timer.C:
		t.Fatal("watch did not deliver a filesystem event")
	}
	return nil
}
func TestRecursiveWatchAndCancellation(t *testing.T) {
	w := fixture(t)
	watch, err := w.Watch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer watch.Close()
	if err = os.WriteFile(filepath.Join(w.Root(), "folder/note.md"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, watch); !reflect.DeepEqual(got, []string{"folder/note.md"}) {
		t.Fatal(got)
	}
	if err = os.Mkdir(filepath.Join(w.Root(), "new"), 0755); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, watch); !reflect.DeepEqual(got, []string{"new"}) {
		t.Fatal(got)
	}
	if err = os.WriteFile(filepath.Join(w.Root(), "new/child.md"), []byte("child"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, watch); !reflect.DeepEqual(got, []string{"new/child.md"}) {
		t.Fatal(got)
	}
	watch.Close()
	if _, ok := <-watch.Changes; ok {
		t.Fatal("watch delivered after Close")
	}
}
