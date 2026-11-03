package preview

import (
	"github.com/egoist/mygo/ui/markdown/parser"
	"strings"
	"testing"
)

func TestMetadataAndTableConventionsPreserveSourceLines(t *testing.T) {
	src := "---\ntitle: Hidden\n---\n<!-- private\ncomment -->\n# Visible --- heading\n\n| Notes |\n| --- |\n| <ul><li>One</li><li>**Two**</li></ul> |\n\n```html\n<!-- code comment -->\n```"
	got := Clean(src)
	if strings.Contains(got, "title: Hidden") || strings.Contains(got, "private") || !strings.Contains(got, "Visible — heading") || !strings.Contains(got, "• One<br>• **Two**") || !strings.Contains(got, "<!-- code comment -->") {
		t.Fatal(got)
	}
	if strings.Count(src, "\n") != strings.Count(got, "\n") {
		t.Fatal("line mapping changed")
	}
}
func TestSearchVisibleTextAcrossFormattingAndLockedBlocks(t *testing.T) {
	src := Clean("# Heading\n\nSome **bold** text, [label](https://hidden.example).\n\n```encrypted\nenc:v1:secret\n```\n")
	got := Prepare(src, "bold text", 0, nil)
	if len(got.Matches) != 1 || !strings.Contains(got.Markdown, "**<mark><u>bold</u></mark>**<mark><u> text</u></mark>") {
		t.Fatalf("%+v %s", got.Matches, got.Markdown)
	}
	if len(Prepare(src, "hidden.example", 0, nil).Matches) != 0 || len(Prepare(src, "secret", 0, nil).Matches) != 0 {
		t.Fatal("searched hidden destination or ciphertext")
	}
	if len(Prepare(src, "opened", 0, map[string]string{"enc:v1:secret": "opened text"}).Matches) != 1 {
		t.Fatal("did not index unlocked text")
	}
}
func TestSearchUnicodeAndActiveBlock(t *testing.T) {
	src := Clean("# Заголовок\n\nКириллица **КИРИЛЛИЦА**.\n\n- First\n- Second\n")
	got := Prepare(src, "кириллица", 1, nil)
	if len(got.Matches) != 2 {
		t.Fatal(got.Matches)
	}
	d := parser.Parse(got.Markdown)
	i := BlockAtLine(d, got.Markdown, 6)
	if i < 0 || d.Block(i).Kind != parser.Paragraph {
		t.Fatal("did not choose nested paragraph")
	}
}

func TestAutolinkSearchKeepsLinkAndMathTypography(t *testing.T) {
	src := Clean("https://example.com and $x--y$\n")
	if !strings.Contains(src, `\(x--y\)`) {
		t.Fatal("typography changed formula", src)
	}
	got := Prepare(src, "example", 0, nil)
	d := parser.Parse(got.Markdown)
	links := 0
	for i := 1; i < d.Len(); i++ {
		for _, n := range d.Inlines(i) {
			if n.Kind == parser.Link {
				links++
			}
		}
	}
	if links != 1 || len(got.Matches) != 1 {
		t.Fatal("search broke autolink", got)
	}
}
