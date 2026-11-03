package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"plainmark/internal/session"
)

func TestPreviewSearchMarkupCountNavigationAndSourceIntegrity(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	src := "# Search\n\nКириллица **КИРИЛЛИЦА**.\n\n" + strings.Repeat("Paragraph.\n\n", 50) + "Last кириллица."
	a.setDocument("note.md", []byte(src))
	d := a.current()
	d.Tab.Mode = session.Preview
	tt := ui.NewTester(a.view, 1000, 500)
	tt.Frame()
	a.find()
	d.Query = "кириллица"
	tt.Frame()
	if len(d.Matches) != 3 || !strings.Contains(d.previewSource, "<mark>") || d.Code.Buffer.String() != src {
		t.Fatal("search changed source or missed rendered text")
	}
	d.Match = 2
	tt.Frame()
	if d.Scroll.Y <= 0 {
		t.Fatal("next search match did not scroll")
	}
	tt.Key(0, ui.KeyEscape)
	if d.FindOpen {
		t.Fatal("Escape did not close find bar")
	}
}
func TestNoteLinksAndMetadata(t *testing.T) {
	a, root := workspaceApp(t)
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\ntitle: Hidden metadata\n---\n<!-- private -->\n# Start\n\nGo to [Child](folder/child.md#content)"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "folder/child.md"), []byte("# Child\n\n"+strings.Repeat("Paragraph.\n\n", 50)+"## Content\nDestination"), 0600); err != nil {
		t.Fatal(err)
	}
	a.reload(a.current())
	a.current().Tab.Mode = session.Preview
	tt := ui.NewTester(a.view, 1000, 500)
	tt.Frame()
	if strings.Contains(a.current().previewSource, "Hidden metadata") || strings.Contains(a.current().previewSource, "private") {
		t.Fatal("metadata visible")
	}
	if err := tt.Click("Child"); err != nil {
		t.Fatal(err)
	}
	if a.current().Tab.Note.Path != "folder/child.md" {
		t.Fatalf("relative link did not open note: active=%s status=%s texts=%q", a.session.Active, a.status, tt.Texts())
	}
	tt.Frame()
	if a.current().Scroll.Y <= 0 {
		t.Fatal("linked anchor did not scroll")
	}
	if len(a.session.Tabs) != 2 {
		t.Fatal("link did not keep previous tab")
	}
}
func TestEncryptedBlockUnlockLockAndNavigation(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	a.run = func(fn func()) { fn() }
	if err = a.load("../../../testdata/workspace/Encrypted.md"); err != nil {
		t.Fatal(err)
	}
	d := a.current()
	original := d.Code.Buffer.String()
	tt := ui.NewTester(a.view, 1220, 820)
	tt.Frame()
	if err = tt.Click("Unlock encrypted block"); err != nil {
		t.Fatal(err)
	}
	if a.crypto == nil {
		t.Fatal("unlock dialog missing")
	}
	a.crypto.password = "archive-pass"
	tt.Frame()
	if err = tt.Click("Unlock"); err != nil {
		t.Fatal(err)
	}
	if len(d.Unlocked) != 1 || a.crypto != nil {
		t.Fatal("unlock failed", a.status)
	}
	for _, plain := range d.Unlocked {
		if plain != "# Legacy secret\nold note" {
			t.Fatal(plain)
		}
	}
	if err = tt.Click("Lock"); err != nil {
		t.Fatal(err)
	}
	if len(d.Unlocked) != 0 || d.Code.Buffer.String() != original || d.Tab.Dirty() {
		t.Fatal("lock changed source or kept plaintext")
	}
	if err = tt.Click("Unlock encrypted block"); err != nil {
		t.Fatal(err)
	}
	a.crypto.password = "wrong"
	a.submitCrypto()
	if a.crypto == nil || a.crypto.message == "" || len(d.Unlocked) != 0 {
		t.Fatal("wrong password was accepted")
	}
	a.crypto.password = "archive-pass"
	a.submitCrypto()
	a.openNote("Math.md", "")
	if len(d.Unlocked) != 0 {
		t.Fatal("navigation retained unlocked blocks")
	}
}
func TestEncryptSelectionOneUndoAndStaleSelection(t *testing.T) {
	a, _ := workspaceApp(t)
	d := a.current()
	d.Tab.Mode = session.Edit
	src := "# Header\n\nСекретный текст\n"
	d.Code.Buffer.Set(src)
	tt := ui.NewTester(a.view, 1220, 820)
	tt.Frame()
	from := strings.Index(src, "Секретный")
	d.Code.Select(from, len(src)-1)
	tt.Frame()
	a.beginEncrypt()
	if a.crypto == nil {
		t.Fatal("encryption dialog missing")
	}
	a.crypto.password = "password"
	a.crypto.confirm = "password"
	a.submitCrypto()
	tt.Frame()
	if !strings.Contains(d.Code.Buffer.String(), "```encrypted\nenc:v1:") || strings.Contains(d.Code.Buffer.String(), "Секретный") {
		t.Fatal("selection was not replaced")
	}
	if err := tt.Click("Markdown editor"); err != nil {
		t.Fatal(err)
	}
	tt.Command("undo")
	if d.Code.Buffer.String() != src {
		t.Fatal("encryption was not one undo step")
	}
	d.Code.Select(from, len(src)-1)
	a.beginEncrypt()
	a.crypto.password = "password"
	a.crypto.confirm = "password"
	d.Code.Buffer.Set("changed during dialog")
	a.submitCrypto()
	if a.crypto == nil || a.crypto.message == "" || d.Code.Buffer.String() != "changed during dialog" {
		t.Fatal("stale selection was overwritten")
	}
}
