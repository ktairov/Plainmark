package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func waitWriteResult(t *testing.T, w *Writer, success bool) {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case err, ok := <-w.Results:
			if !ok {
				t.Fatal("writer stopped before completing the expected attempt")
			}
			if (err == nil) == success {
				return
			}
		case <-timer.C:
			t.Fatal("writer did not complete the expected attempt")
		}
	}
}

func TestWriterRetriesUnchangedStateAfterDestinationRecovers(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(parent, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "state.json")
	w := NewWriter(path)
	defer w.Close()
	w.Queue([]byte(`{"version":1}`))
	waitWriteResult(t, w, false)
	if err := os.Remove(parent); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	// No second Queue: a failed snapshot must retry even if the UI is idle.
	waitWriteResult(t, w, true)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != `{"version":1}` {
		t.Fatalf("state not recovered: %q %v", data, err)
	}
}

func TestWriterRetriesLatestSnapshotInsteadOfFailedOlderOne(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(parent, nil, 0600); err != nil {
		t.Fatal(err)
	}
	w := NewWriter(filepath.Join(parent, "state.json"))
	defer w.Close()
	w.Queue([]byte("older"))
	waitWriteResult(t, w, false)
	newer := []byte("newer")
	w.Queue(newer)
	newer[0] = 'X' // Queue must retain its own immutable snapshot.
	if err := os.Remove(parent); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	waitWriteResult(t, w, true)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(parent, "state.json"))
	if err != nil || string(data) != "newer" {
		t.Fatalf("stale or mutable snapshot written: %q %v", data, err)
	}
}
