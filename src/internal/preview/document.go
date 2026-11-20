package preview

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui/markdown/parser"
)

type edit struct {
	start, end int
	text       string
}

func edits(src string, changes []edit) string {
	sort.Slice(changes, func(i, j int) bool { return changes[i].start < changes[j].start })
	var b strings.Builder
	at := 0
	for _, e := range changes {
		if e.start < at || e.start < 0 || e.end > len(src) {
			continue
		}
		b.WriteString(src[at:e.start])
		b.WriteString(e.text)
		at = e.end
	}
	b.WriteString(src[at:])
	return b.String()
}
func blank(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' {
			b.WriteByte('\n')
		} else if r == '\r' {
			b.WriteByte('\r')
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

var comment = regexp.MustCompile(`(?s)^\s*<!--.*-->\s*$`)
var listTag = regexp.MustCompile(`(?i)^</?(ul|li)\s*>$`)

// Clean applies display-only metadata/typography/table conventions. Newlines
// stay in place, so preview block lines still map to the original editor.
func Clean(src string) string {
	if strings.HasPrefix(src, "---\n") || strings.HasPrefix(src, "---\r\n") {
		at := strings.IndexByte(src, '\n') + 1
		for _, line := range strings.SplitAfter(src[at:], "\n") {
			at += len(line)
			trim := strings.TrimSpace(line)
			if trim == "---" || trim == "..." {
				src = blank(src[:at]) + src[at:]
				break
			}
		}
	}
	src = Normalize(src)
	d := parser.Parse(src)
	var changes []edit
	for i := 1; i < d.Len(); i++ {
		b := d.Block(i)
		if b.Kind == parser.HTMLBlock && comment.MatchString(d.Lines(i)) {
			changes = append(changes, edit{b.Start, b.End, blank(src[b.Start:b.End])})
			continue
		}
		hasList := false
		for _, n := range d.Inlines(i) {
			if n.Kind == parser.HTML && comment.MatchString(n.Text) {
				changes = append(changes, edit{n.Start, n.End, blank(src[n.Start:n.End])})
			}
			if b.Kind == parser.TableCell && n.Kind == parser.HTML && listTag.MatchString(n.Text) {
				replacement := ""
				if strings.HasPrefix(strings.ToLower(n.Text), "<li") {
					if hasList {
						replacement = "<br>"
					}
					replacement += "• "
					hasList = true
				}
				changes = append(changes, edit{n.Start, n.End, replacement})
			}
			if n.Kind == parser.Text && src[n.Start:n.End] == n.Text {
				text := strings.NewReplacer("---", "—", "--", "–").Replace(n.Text)
				if text != n.Text {
					changes = append(changes, edit{n.Start, n.End, text})
				}
			}
		}
	}
	return Normalize(edits(src, changes))
}

type Match struct {
	Start, End, Line int
	Kind             string
}
type Prepared struct {
	Markdown string
	Matches  []Match
}
type mapped struct {
	start, end, node int
	kind             string
}

// Prepare searches rendered text, including across inline formatting, while
// omitting link destinations, syntax, metadata and locked ciphertext. Search
// markup exists only in the preview string; it never touches the editor buffer.
func Prepare(clean, query string, current int, unlocked map[string]string) Prepared {
	result := Prepared{Markdown: clean}
	if query == "" {
		return result
	}
	d := parser.Parse(clean)
	var visible strings.Builder
	var mapping []mapped
	node := 0
	appendText := func(text string, start, end int, kind string) {
		node++
		raw := clean[start:end]
		offset := strings.Index(raw, text)
		exact := offset >= 0
		for off, r := range text {
			width := utf8.RuneLen(r)
			from, to := start, end
			if exact {
				from = start + offset + off
				to = from + width
			}
			visible.WriteRune(r)
			for range width {
				mapping = append(mapping, mapped{from, to, node, kind})
			}
		}
	}
	for i := 1; i < d.Len(); i++ {
		b := d.Block(i)
		if b.Kind == parser.CodeBlock {
			text, kind := d.Lines(i), "code"
			if b.Language() == "encrypted" {
				text = unlocked[strings.TrimSpace(text)]
				kind = "encrypted"
			}
			appendText(text, b.Start, b.End, kind)
		} else if b.Kind == parser.MathBlock {
			appendText(d.Lines(i), b.Start, b.End, "math")
		} else {
			autoEnd := -1
			for j, n := range d.Inlines(i) {
				if n.Kind == parser.Link && !strings.HasPrefix(clean[n.Start:n.End], "[") {
					autoEnd = n.Next
				}
				switch n.Kind {
				case parser.Text:
					kind := "text"
					if j < autoEnd {
						kind = "link"
					}
					appendText(n.Text, n.Start, n.End, kind)
				case parser.Code:
					appendText(n.Text, n.Start, n.End, "code")
				case parser.Math:
					appendText(n.Text, n.Start, n.End, "math")
				case parser.SoftBreak, parser.LineBreak:
					appendText(" ", n.Start, n.End, "space")
				}
			}
		}
		visible.WriteByte('\n')
		mapping = append(mapping, mapped{b.End, b.End, -1, "separator"})
	}
	matcher, err := regexp.Compile("(?i)" + regexp.QuoteMeta(query))
	if err != nil {
		return result
	}
	ranges := matcher.FindAllStringIndex(visible.String(), 2000)
	var highlights []edit
	for index, r := range ranges {
		if r[0] >= len(mapping) || r[1] <= r[0] {
			continue
		}
		first, last := mapping[r[0]], mapping[r[1]-1]
		result.Matches = append(result.Matches, Match{first.start, last.end, strings.Count(clean[:first.start], "\n") + 1, first.kind})
		for at := r[0]; at < r[1]; {
			m := mapping[at]
			end := at + 1
			for end < r[1] && mapping[end].node == m.node {
				end++
			}
			to := mapping[end-1].end
			if m.kind == "text" && m.start < to {
				before, after := "<mark>", "</mark>"
				if index == current {
					before = "<mark><u>"
					after = "</u></mark>"
				}
				highlights = append(highlights, edit{m.start, to, before + clean[m.start:to] + after})
			}
			at = end
		}
	}
	result.Markdown = edits(clean, highlights)
	return result
}

// BlockAtLine picks a rendered block, preferring a paragraph within a list or
// quote to the whole container. Table cells/items are rendered by their parent.
func BlockAtLine(d *parser.Document, src string, line int) int {
	best, span := -1, len(src)+1
	for i := 1; i < d.Len(); i++ {
		b := d.Block(i)
		switch b.Kind {
		case parser.Paragraph, parser.Heading, parser.CodeBlock, parser.MathBlock, parser.Table, parser.Quote, parser.List:
		default:
			continue
		}
		from, to := strings.Count(src[:b.Start], "\n")+1, strings.Count(src[:b.End], "\n")+1
		if line >= from && line <= to && b.End-b.Start < span {
			best, span = i, b.End-b.Start
		}
	}
	return best
}
