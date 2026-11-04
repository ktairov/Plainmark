package desktop

import (
	"path/filepath"
	"regexp"
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
		d.previewReady = true
		d.lastQuery, d.lastMatch, d.lastUnlockRevision = query, d.Match, d.unlockRevision
		d.searchRevision++
	}
}
func (a *App) preview(c *ui.Context, d *Document) {
	d.preparePreview()
	ui.Column(c).Grow(1).FillWidth().MinHeight(0).Children(func() {
		ui.FindBar(c, &d.FindOpen, &d.Query, len(d.Matches), &d.Match)
		d.preparePreview()
		a.math.Configure(&d.MD, c.Theme().Text, float32(a.preferences.ContentFontSize)*1.25)
		d.MD.Theme = a.readingTheme(c.Theme())
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
		active := -1
		if d.Tab.Mode == session.Split {
			active = preview.BlockAtLine(d.previewDoc, d.previewSource, d.Code.CursorPos().Line+1)
		}
		selected := -1
		if d.FindOpen && len(d.Matches) > 0 {
			selected = preview.BlockAtLine(d.previewDoc, d.previewSource, d.Matches[d.Match].Line)
		}
		needsScroll := d.FindOpen && d.searchRevision != d.scrolledRevision
		d.MD.Block = func(c *ui.Context, doc *parser.Document, index int) bool {
			if index == active {
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(c.Theme().Accent.Alpha(.08)).BorderStartWidth(2).BorderColor(c.Theme().Accent).PassThrough()
			}
			if index == selected {
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Border(1, c.Theme().Accent).PassThrough()
				if needsScroll {
					ui.Box(c).Absolute().Top(0).Left(0).Size(1, 1).PassThrough().ScrollIntoView()
				}
			}
			return false
		}
		d.MD.Code = map[string]func(*ui.Context, *markdown.CodeBlockView){"encrypted": func(c *ui.Context, b *markdown.CodeBlockView) { a.encryptedBlock(c, d, strings.TrimSpace(b.Code())) }}
		d.MD.CodeBlock = func(c *ui.Context, b *markdown.CodeBlockView) {
			if !d.FindOpen || d.Query == "" {
				markdown.DefaultCodeBlock(c, b)
				return
			}
			ui.Column(c).Background(c.Theme().Surface).Border(1, c.Theme().Border).Radius(6).Padding(8).Children(func() {
				ui.Row(c).Children(func() { ui.Text(c, b.Language).FontSize(12); ui.Box(c).Grow(1); b.CopyButton(c) })
				ui.ScrollHorizontal(c).FillWidth().Children(func() {
					ui.RichText(c, searchSpans(b.Code(), d.Query, c.Theme())...).Font("Menlo").FontSize(14).NoWrap()
				})
			})
		}
		width, _ := c.Size()
		ui.Scroll(c).TrackScroll(&d.Scroll).Grow(1).FillWidth().Padding(24, min(float32(80), width*.07), 80).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(float32(a.preferences.PreviewWidth)).AlignSelf(ui.Center).Children(func() { markdown.View(c, d.previewSource, &d.MD).FillWidth() })
		})
		d.scrolledRevision = d.searchRevision
	})
}

func (a *App) readingTheme(theme *ui.Theme) *markdown.Theme {
	id := a.preferences.DayTheme
	if theme.Dark {
		id = a.preferences.NightTheme
	}
	palette := settings.Themes[id].Light
	if theme.Dark {
		palette = settings.Themes[id].Dark
	}
	reading := markdown.DefaultTheme(theme)
	reading.FontSize = float32(a.preferences.ContentFontSize)
	reading.Font = settings.Font(a.preferences.ReadingFont)
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
