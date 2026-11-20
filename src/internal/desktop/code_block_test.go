package desktop

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	"plainmark/internal/session"
)

func TestTextDiagramKeepsVerticalStrokesConnected(t *testing.T) {
	const code = "┌── GitHub\n│\n├── Slack\n│\n└── SaaS"
	plainBounds := make(map[int]ui.Rect)
	for _, size := range []int{13, 16, 22, 26} {
		for _, search := range []bool{false, true} {
			t.Run(fmt.Sprintf("size=%d/search=%v", size, search), func(t *testing.T) {
				a, _ := workspaceApp(t)
				source := "```text\n" + code + "\n```\n"
				a.setDocument("diagram.md", []byte(source))
				d := a.current()
				d.Tab.Mode = session.Preview
				a.preferences.ContentFontSize = size
				d.FindOpen, d.Query = search, "Slack"
				tt := ui.NewTester(a.view, 1220, 700)
				tt.Frame()
				body, ok := tt.Find(code)
				if !ok {
					body, ok = tt.Find(code + "\n")
				}
				if !ok {
					t.Fatalf("diagram text missing: %q", tt.Texts())
				}
				if !search {
					plainBounds[size] = body
				} else if want := plainBounds[size]; math.Abs(float64(body.H-want.H)) > 1 || math.Abs(float64(body.W-want.W)) > 1 {
					t.Errorf("search changed diagram dimensions: got %+v, plain %+v", body, want)
				}
				// Find the vertical stem in the first character cell, then
				// require ink on every row between its upper and lower ends.
				img := tt.Image()
				x0, x1 := int(body.X), int(body.X+float32(size)*.875*.7)
				first, last := -1, -1
				var rows []bool
				for y := int(math.Ceil(float64(body.Y))); y < int(body.Y+body.H); y++ {
					ink := false
					for x := x0; x < x1; x++ {
						p := img.RGBAAt(x, y)
						if int(p.R)+int(p.G)+int(p.B) < 500 {
							ink = true
						}
					}
					rows = append(rows, ink)
					if ink {
						if first < 0 {
							first = len(rows) - 1
						}
						last = len(rows) - 1
					}
				}
				if first < 0 || last-first < size*3 {
					t.Fatalf("vertical stroke not found: body=%+v", body)
				}
				for y := first; y <= last; y++ {
					if !rows[y] {
						t.Errorf("vertical stroke has a gap at row %d", y)
						break
					}
				}
				if d.Code.Buffer.String() != source {
					t.Fatal("rendering changed the note")
				}
				if err := tt.Click("Copy code"); err != nil {
					t.Fatal(err)
				}
				if tt.Clipboard() != code+"\n" {
					t.Fatalf("copy changed diagram whitespace: %q", tt.Clipboard())
				}
				if dir := os.Getenv("PLAINMARK_SNAPSHOTS"); dir != "" {
					saveFrame(t, tt, dir, fmt.Sprintf("code-diagram-%d-search-%v.png", size, search))
				}
			})
		}
	}
}
