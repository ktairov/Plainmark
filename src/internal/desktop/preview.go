package desktop

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/markdown/parser"
	"plainmark/internal/preview"
	"plainmark/internal/session"
	"plainmark/internal/settings"
)

func (a *App) find() {
	d := a.current()
	if d == nil {
		return
	}
	if d.Tab.Mode == session.Preview {
		d.FindOpen = true
	} else {
		d.Code.Find("")
	}
}
func (d *Document) preparePreview() {
	changed := !d.previewReady || d.previewVersion != d.Code.Buffer.Version()
	if changed {
		d.cleanSource = preview.Clean(d.Code.Buffer.String())
		d.previewVersion = d.Code.Buffer.Version()
	}
	query := ""
	if d.FindOpen {
		query = d.Query
	}
	if changed || query != d.lastQuery || d.Match != d.lastMatch || d.unlockRevision != d.lastUnlockRevision {
		if query != d.lastQuery {
			d.Match = 0
		}
		prepared := preview.Prepare(d.cleanSource, query, d.Match, d.Unlocked)
		d.previewSource, d.Matches = prepared.Markdown, prepared.Matches
		if len(d.Matches) > 0 {
			d.Match = min(max(0, d.Match), len(d.Matches)-1)
		} else {
			d.Match = 0
		}
		d.previewDoc = parser.Parse(d.previewSource)
		clear(d.codeDiagrams)
		d.previewLines = append(d.previewLines[:0], 0)
		for i, ch := range d.previewSource {
			if ch == '\n' {
				d.previewLines = append(d.previewLines, i+1)
			}
		}
		d.listMarkerDigits = orderedListDigits(d.previewDoc)
		d.previewReady = true
		d.lastQuery, d.lastMatch, d.lastUnlockRevision = query, d.Match, d.unlockRevision
		d.searchRevision++
	}
}
func (a *App) preview(c *ui.Context, d *Document) {
	d.preparePreview()
	ui.Column(c).Grow(1).FillWidth().MinHeight(0).Children(func() {
		if ui.FindBar(c, &d.FindOpen, &d.Query, len(d.Matches), &d.Match).Changed() {
			// A navigation request must reveal even a single match again.
			d.searchRevision++
		}
		d.preparePreview()
		a.math.Configure(&d.MD, c.Theme().Text, float32(a.preferences.ContentFontSize)*1.25)
		d.MD.Theme = a.readingTheme(c.Theme())
		fitListMarkers(c, d.MD.Theme, d.listMarkerDigits)
		if a.store != nil {
			d.MD.LoadImage = markdown.FileImages(filepath.Join(a.store.Root(), filepath.FromSlash(parent(d.Tab.Note.Path))))
		}
		d.MD.OnLink = func(href string) {
			path, anchor, external, err := preview.ResolveLink(d.Tab.Note.Path, href)
			if err != nil {
				a.status = err.Error()
				return
			}
			if external != "" {
				c.OpenURL(external)
				return
			}
			if path == "" {
				d.MD.Anchor = anchor
				d.MD.AnchorVersion++
				return
			}
			a.navigateNote(path, anchor, c.Modifiers()&(ui.Cmd|ui.Ctrl) != 0)
		}
		active, selected, mapped := -1, -1, false
		d.splitScroll.blocks = d.splitScroll.blocks[:0]
		needsScroll := d.FindOpen && d.searchRevision != d.scrolledRevision
		d.MD.Block = func(c *ui.Context, doc *parser.Document, index int) bool {
			if d.Tab.Mode == session.Split && doc.Block(index).Parent == 0 {
				marker := ui.Box(c).Key("split-scroll-anchor").Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough()
				if bounds := marker.Bounds(); bounds.H > 0 {
					line := sort.SearchInts(d.previewLines, doc.Block(index).Start+1) - 1
					d.splitScroll.blocks = append(d.splitScroll.blocks, scrollBlock{line: line, y: float64(bounds.Y)})
				}
			}
			if !mapped {
				mapped = true
				if d.Tab.Mode == session.Split {
					active = preview.BlockAtLine(doc, doc.Source(), d.Code.CursorPos().Line+1)
				}
				if d.FindOpen && len(d.Matches) > 0 {
					selected = preview.BlockAtLine(doc, doc.Source(), d.Matches[d.Match].Line)
				}
			}
			if index == active {
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(c.Theme().Accent.Alpha(.08)).BorderStartWidth(2).BorderColor(c.Theme().Accent).PassThrough()
			}
			if index == selected {
				// The highlight is also a stable, full-block reveal target. A
				// conditional target disappears during eager input rebuilds,
				// losing the pending scroll before layout can apply it.
				target := ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Border(1, c.Theme().Accent).PassThrough()
				if needsScroll {
					target.ScrollIntoView()
				}
			}
			return false
		}
		d.MD.Code = map[string]func(*ui.Context, *markdown.CodeBlockView){"encrypted": func(c *ui.Context, b *markdown.CodeBlockView) { a.encryptedBlock(c, d, strings.TrimSpace(b.Code())) }}
		d.MD.CodeBlock = func(c *ui.Context, b *markdown.CodeBlockView) {
			search := d.FindOpen && d.Query != ""
			compact, known := d.codeDiagrams[b.Index]
			if !known {
				compact = textDiagram(b.Language, b.Code())
				if d.codeDiagrams == nil {
					d.codeDiagrams = make(map[int]bool)
				}
				d.codeDiagrams[b.Index] = compact
			}
			if !search && !compact {
				markdown.DefaultCodeBlock(c, b)
				return
			}
			a.previewCodeBlock(c, d, b, search, compact)
		}
		width, _ := c.Size()
		ui.Scroll(c).TrackScroll(&d.Scroll).Grow(1).FillWidth().Padding(24, min(float32(80), width*.07), 80).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(float32(a.preferences.PreviewWidth)).AlignSelf(ui.Center).Children(func() {
				content := markdown.View(c, d.previewSource, &d.MD).FillWidth()
				d.splitScroll.contentY = float64(content.Bounds().Y)
			})
		})
		d.scrolledRevision = d.searchRevision
	})
}

func orderedListDigits(doc *parser.Document) int {
	digits := 0
	for i := 1; i < doc.Len(); i++ {
		list := doc.Block(i)
		if list.Kind != parser.List || !list.Ordered {
			continue
		}
		count := 0
		for item := i + 1; item < list.Next; item = doc.Block(item).Next {
			count++
		}
		digits = max(digits, len(strconv.Itoa(list.Number+max(0, count-1))))
	}
	return digits
}

func fitListMarkers(c *ui.Context, theme *markdown.Theme, digits int) {
	if digits == 0 {
		return
	}
	// MyGo v0.7.4 measures ordered markers without tnum, then renders with
	// tnum. Reserve the actual tabular width plus its 0.4em gap and rounding
	// allowance, including lists that cross a digit boundary (99 -> 100).
	for _, delimiter := range []string{".", ")"} {
		width, _ := c.MeasureText(0, ui.Span{Text: strings.Repeat("9", digits) + delimiter,
			Font: theme.Font, Size: theme.FontSize, Features: "tnum"})
		theme.ListIndent = max(theme.ListIndent, (width+1)/theme.FontSize+0.4)
	}
}

func (a *App) readingTheme(theme *ui.Theme) *markdown.Theme {
	palette := a.palette(theme.Dark)
	reading := markdown.DefaultTheme(theme)
	reading.FontSize = float32(a.preferences.ContentFontSize)
	reading.Font = settings.Font(a.preferences.ReadingFont)
	reading.CodeFont = settings.Font("mono")
	reading.LineHeight = 1.65
	reading.Link = ui.Hex(palette.Link)
	reading.Muted = ui.Hex(palette.Muted)
	reading.CodeBackground = ui.Hex(palette.Code)
	reading.CodeBlockBackground = ui.Hex(palette.Code)
	reading.CodeBlockBorder = ui.Hex(palette.Border).Mix(theme.Background, .35)
	reading.CodeBlockRadius = 6
	reading.Mark = theme.Accent.Alpha(.3)
	for i := 0; i < 3; i++ {
		reading.Headings[i].Color = ui.Hex(palette.Heading)
	}
	return reading
}
func searchSpans(text, query string, theme *ui.Theme) []ui.Span {
	if query == "" {
		return []ui.Span{{Text: text}}
	}
	r, err := regexp.Compile("(?i)" + regexp.QuoteMeta(query))
	if err != nil {
		return []ui.Span{{Text: text}}
	}
	var spans []ui.Span
	at := 0
	for _, m := range r.FindAllStringIndex(text, 2000) {
		if m[0] > at {
			spans = append(spans, ui.Span{Text: text[at:m[0]]})
		}
		spans = append(spans, ui.Span{Text: text[m[0]:m[1]], Background: theme.Accent.Alpha(.25)})
		at = m[1]
	}
	if at < len(text) {
		spans = append(spans, ui.Span{Text: text[at:]})
	}
	return spans
}
