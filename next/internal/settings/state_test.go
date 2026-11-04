package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriterFlushesLastStateAndNormalizesNavigation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "config/state.json")
	w := NewWriter(path)
	s := NewState()
	s.Root = root
	s.Recent = []string{root, root, "relative"}
	s.Roots[root] = RootState{Tabs: []StoredTab{{Path: "a.md", Mode: "split", PreviewY: 99}, {Path: "../escape.md"}, {Path: "a.md"}, {Path: "b.txt"}}, Active: "absent.md", Pinned: []string{"a.md", "../outside.md"}}
	for i := 0; i < 30; i++ {
		s.Preferences.ContentFontSize = 13 + i%14
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if !w.Queue(data) {
			t.Fatal("writer unexpectedly closed")
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Preferences.ContentFontSize != s.Preferences.ContentFontSize || len(loaded.Recent) != 1 || len(loaded.Roots[root].Tabs) != 1 || loaded.Roots[root].Active != "a.md" || len(loaded.Roots[root].Pinned) != 1 {
		t.Fatalf("%+v", loaded)
	}
	if w.Queue([]byte("after close")) {
		t.Fatal("writer accepted state after Close")
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("config permissions", err)
	}
}
func TestStateRejectsUnknownVersionAndReportsWriteErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(`{"version":999}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); !errors.Is(err, ErrVersion) {
		t.Fatal(err)
	}
	w := NewWriter(filepath.Join(path, "not-a-directory.json"))
	w.Queue([]byte("{}"))
	if err := w.Close(); err == nil {
		t.Fatal("missing write error")
	}
}
