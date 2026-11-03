package workspace

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui/markdown/parser"
)

var titleLinks = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)

func DisplayTitle(src string) string {
	src = titleLinks.ReplaceAllString(src, "$1")
	for _, m := range []string{"**", "__", "~~", "*", "`"} {
		src = strings.ReplaceAll(src, m, "")
	}
	var b strings.Builder
	tag, escape := false, false
	for _, r := range src {
		if escape {
			b.WriteRune(r)
			escape = false
		} else if r == '\\' {
			escape = true
		} else if r == '<' {
			tag = true
		} else if r == '>' && tag {
			tag = false
		} else if !tag && !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	s := strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'").Replace(b.String())
	s = strings.NewReplacer("---", "—", "--", "–", "...", "…").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func headingLine(line string) (Heading, bool) {
	s := strings.TrimLeftFunc(line, unicode.IsSpace)
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return Heading{}, false
	}
	text := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(s[n:]), "#"))
	return Heading{n, text}, text != ""
}

// Retain the current app's ATX heading metadata contract, including its fallback
// for malformed/non-CommonMark headings; setext rendering remains MyGo's job.
func ParseHeadings(src string) []Heading {
	d := parser.Parse(src)
	out := []Heading{}
	for i := 1; i < d.Len(); i++ {
		b := d.Block(i)
		if b.Kind != parser.Heading {
			continue
		}
		line := src[b.Start:]
		if end := strings.IndexByte(line, '\n'); end >= 0 {
			line = line[:end]
		}
		if h, ok := headingLine(line); ok {
			out = append(out, h)
		}
	}
	if len(out) == 0 {
		for _, line := range strings.Split(src, "\n") {
			if h, ok := headingLine(line); ok {
				out = append(out, h)
			}
		}
	}
	return out
}
func TitleFor(path, src string, headings []Heading) string {
	if len(headings) > 0 {
		return DisplayTitle(headings[0].Text)
	}
	for _, line := range strings.Split(src, "\n") {
		if h, ok := headingLine(line); ok {
			return DisplayTitle(h.Text)
		}
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return DisplayTitle(strings.NewReplacer("-", " ", "_", " ").Replace(stem))
}
