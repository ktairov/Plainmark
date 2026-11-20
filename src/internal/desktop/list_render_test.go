package desktop

import (
	"fmt"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	"plainmark/internal/session"
)

func TestOrderedListMarkersStayOnOneLine(t *testing.T) {
	for _, font := range []string{"system", "humanist", "serif", "mono"} {
		for _, size := range []int{13, 16, 22, 26} {
			t.Run(fmt.Sprintf("%s-size=%d", font, size), func(t *testing.T) {
				a, err := New()
				if err != nil {
					t.Fatal(err)
				}
				a.setDocument("lists.md", []byte("8. Item eight\n9. Item nine\n10. Item ten\n\n### Next section\n\n11. Item eleven\n12. Item twelve\n13. Item thirteen\n\n### Three digits\n\n99. Item ninety-nine\n100. Item one hundred\n\n### Nested\n\n- Group\n\n  9) Nested nine\n  10) Nested ten\n"))
				a.current().Tab.Mode = session.Preview
				a.preferences.ContentFontSize = size
				a.preferences.ReadingFont = font
				tt := ui.NewTester(a.view, 1220, 1500)
				tt.Frame()
				for marker, text := range map[string]string{"10.": "Item ten", "11.": "Item eleven", "12.": "Item twelve", "100.": "Item one hundred", "10)": "Nested ten"} {
					mark, ok := tt.Find(marker)
					if !ok {
						t.Fatalf("marker %q missing", marker)
					}
					content, ok := tt.Find(text)
					if !ok {
						t.Fatalf("item %q missing", text)
					}
					if mark.H > content.H+1 {
						t.Errorf("marker %q wraps: height %.1f, item height %.1f", marker, mark.H, content.H)
					}
				}
				if dir := os.Getenv("PLAINMARK_SNAPSHOTS"); dir != "" {
					saveFrame(t, tt, dir, fmt.Sprintf("ordered-lists-%s-%d.png", font, size))
				}
			})
		}
	}
}
