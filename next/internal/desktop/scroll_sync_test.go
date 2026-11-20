package desktop

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"plainmark/internal/session"
)

func TestSplitScrollFollowsEitherPaneAndSettles(t *testing.T) {
	a, _ := workspaceApp(t)
	var source strings.Builder
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&source, "# Section %02d\n\nParagraph with **formatting** and some text.\n\n", i)
	}
	a.setDocument("split.md", []byte(source.String()))
	d := a.current()
	d.Tab.Mode = session.Split
	tt := ui.NewTester(a.view, 1200, 650)
	settle := func() {
		for i := 0; i < 5; i++ {
			tt.Frame()
		}
	}
	tt.Frame()

	// Scrolling the source heading to the top must show that same heading
	// in preview, despite different font sizes and paragraph spacing.
	tt.Scroll(450, 350, 0, float32(d.Code.LineTop(20*4)))
	settle()
	heading, ok := tt.Find("Section 20")
	if !ok || heading.H < 20 || heading.Y < 40 || heading.Y > 100 {
		t.Fatalf("editor scroll did not reveal corresponding preview heading: %+v, editor=%v preview=%v", heading, d.Code.ScrollY(), d.Scroll.Y)
	}

	// Move another preview heading to the top using wheel input on the right.
	heading, ok = tt.Find("Section 35")
	if !ok {
		t.Fatal("preview heading missing")
	}
	editor, ok := tt.Find("Markdown editor")
	if !ok {
		t.Fatal("split editor missing")
	}
	tt.Scroll(1050, 350, 0, heading.Y-editor.Y)
	settle()
	if delta := math.Abs(d.Code.ScrollY() - d.Code.LineTop(35*4)); delta > 5 {
		t.Fatalf("preview scroll did not align source: delta=%v editor=%v preview=%v", delta, d.Code.ScrollY(), d.Scroll.Y)
	}
	editorY, previewY := d.Code.ScrollY(), d.Scroll.Y
	settle()
	if math.Abs(d.Code.ScrollY()-editorY) > .5 || math.Abs(float64(d.Scroll.Y-previewY)) > .5 {
		t.Fatal("synchronized panes keep moving without input")
	}

	// Both extremes work in either direction, including scrollbar jumps.
	tt.Scroll(1050, 350, 0, -100000)
	settle()
	if d.Scroll.Y != 0 || d.Code.ScrollY() != 0 {
		t.Fatalf("top: editor=%v preview=%v", d.Code.ScrollY(), d.Scroll.Y)
	}
	tt.Scroll(450, 350, 0, 100000)
	settle()
	if math.Abs(float64(d.Scroll.Y-d.Scroll.MaxY)) > 1 {
		t.Fatalf("bottom: preview=%v max=%v", d.Scroll.Y, d.Scroll.MaxY)
	}
}
