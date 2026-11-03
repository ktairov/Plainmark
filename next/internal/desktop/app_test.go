package desktop

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestEditorPreviewAndSave(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	a.run = func(fn func()) { fn() }
	path := filepath.Join(t.TempDir(), "note.md")
	if err = os.WriteFile(path, []byte("# Исходный\n\nFormula $x_i^2 + \\frac{1}{2}$\n"), 0640); err != nil {
		t.Fatal(err)
	}
	if err = a.load(path); err != nil {
		t.Fatal(err)
	}
	a.current().Tab.Mode = 2
	tt := ui.NewTester(a.view, 1220, 820)
	tt.Frame()
	if err = tt.Click("Markdown editor"); err != nil {
		t.Fatal(err)
	}
	tt.Command("selectAll")
	tt.Type("# Изменённый\n\nFormula $x_i^2 + \\frac{1}{2}$\n")
	if !a.current().Code.Buffer.Modified() {
		t.Fatal("editing did not mark buffer dirty")
	}
	tt.Command("undo")
	if !strings.Contains(a.current().Code.Buffer.String(), "Исходный") {
		t.Fatal("undo did not restore document")
	}
	tt.Command("redo")
	if !strings.Contains(a.current().Code.Buffer.String(), "Изменённый") {
		t.Fatal("redo lost Unicode edit")
	}
	tt.Command("selectAll")
	tt.Command("copy")
	if tt.Clipboard() != a.current().Code.Buffer.String() {
		t.Fatal("clipboard lost selected Unicode text")
	}
	a.current().Code.Find("Formula")
	tt.Frame()
	if n, _ := a.current().Code.FindMatches(); n != 1 {
		t.Fatalf("editor search: got %d matches", n)
	}
	tt.Key(0, ui.KeyEscape)
	if err = tt.Click("Save"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != a.current().Code.Buffer.String() || a.current().Code.Buffer.Modified() {
		t.Fatal("save did not persist draft and mark it saved")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Fatal("save changed permissions")
	}
	if err = tt.Click("Preview"); err != nil {
		t.Fatal(err)
	}
	if a.current().Tab.Mode != 0 || !tt.HasText("Изменённый") {
		t.Fatalf("preview did not show edited heading: %q", tt.Texts())
	}
	if err = tt.Click("Edit"); err != nil {
		t.Fatal(err)
	}
	if a.current().Tab.Mode != 1 {
		t.Fatal("edit mode not activated")
	}
	if err = tt.Click("Split"); err != nil {
		t.Fatal(err)
	}
	if a.current().Tab.Mode != 2 {
		t.Fatal("split mode not activated")
	}
}

func TestSaveKeepsDraftOnExternalConflict(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	a.run = func(fn func()) { fn() }
	path := filepath.Join(t.TempDir(), "note.md")
	if err = os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = a.load(path); err != nil {
		t.Fatal(err)
	}
	a.current().Code.Buffer.Set("draft")
	if err = os.WriteFile(path, []byte("external"), 0600); err != nil {
		t.Fatal(err)
	}
	a.save()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "external" || a.current().Code.Buffer.String() != "draft" || !a.current().Code.Buffer.Modified() || !a.current().Tab.Conflict {
		t.Fatalf("conflict lost data: disk=%q draft=%q status=%q", data, a.current().Code.Buffer.String(), a.status)
	}
}

// Optional snapshot output for visual review; regular tests leave no artifacts.
func TestMathDocumentSnapshot(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err = a.load("../../../testdata/workspace/Math.md"); err != nil {
		t.Fatal(err)
	}
	a.current().Tab.Mode = 0
	tt := ui.NewTester(a.view, 1100, 1000)
	tt.Frame()
	if !tt.HasText("Формулы") {
		t.Fatalf("missing heading: %q", tt.Texts())
	}
	if dir := os.Getenv("PLAINMARK_SNAPSHOTS"); dir != "" {
		if err = os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{false, true} {
			tt.SetDark(dark)
			name := "math-light.png"
			if dark {
				name = "math-dark.png"
			}
			f, err := os.Create(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, tt.Image())
			closeErr := f.Close()
			if err != nil {
				t.Fatal(err)
			}
			if closeErr != nil {
				t.Fatal(closeErr)
			}
		}
	}
}

func TestTablesAndHeadingAnchors(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	a.setDocument("", []byte("# Document\n\n[Jump](#target) and [External](https://example.com)\n\n| Type | Value |\n| --- | --- |\n| Bold | **Yes** |\n\n"+strings.Repeat("Paragraph.\n\n", 60)+"## Target\n\nDestination"))
	a.current().Tab.Mode = 0
	tt := ui.NewTester(a.view, 1000, 500)
	tt.Frame()
	if !tt.HasText("Yes") {
		t.Fatalf("table missing: %q", tt.Texts())
	}
	if err = tt.Click("External"); err != nil {
		t.Fatal(err)
	}
	if urls := tt.OpenedURLs(); len(urls) != 1 || urls[0] != "https://example.com" {
		t.Fatalf("external link: %v", urls)
	}
	if err = tt.Click("Jump"); err != nil {
		t.Fatal(err)
	}
	if a.current().Scroll.Y <= 0 {
		t.Fatal("heading anchor did not scroll")
	}
}

func TestLongDocumentEditingAndPreview(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	src := "# Long document\n\n" + strings.Repeat("A paragraph with **formatting** and $x_i$.\n\n", 2000)
	a.setDocument("", []byte(src))
	a.current().Tab.Mode = 1
	tt := ui.NewTester(a.view, 1000, 700)
	tt.Frame()
	if err = tt.Click("Markdown editor"); err != nil {
		t.Fatal(err)
	}
	a.current().Code.Select(len(src), len(src))
	tt.Frame()
	tt.Type("End of document.")
	if !strings.HasSuffix(a.current().Code.Buffer.String(), "End of document.") {
		t.Fatal("long document edit failed")
	}
	a.current().Tab.Mode = 0
	tt.Frame()
	if !tt.HasText("Long document") {
		t.Fatal("long document preview failed")
	}
}
