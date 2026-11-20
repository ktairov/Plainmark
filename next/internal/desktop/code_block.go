package desktop

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

func textDiagram(language, code string) bool {
	switch strings.ToLower(language) {
	case "", "text", "txt", "plaintext", "ascii":
	default:
		return false
	}
	return strings.ContainsRune(code, '\n') && strings.ContainsAny(code, "│┃┌┐└┘├┤┬┴┼╔╗╚╝╠╣╦╩╬║╭╮╯╰")
}

func (a *App) previewCodeBlock(c *ui.Context, d *Document, b *markdown.CodeBlockView, search, compact bool) {
	th := d.MD.Theme
	em := th.FontSize
	pad := em * th.CodeBlockPadding
	height := th.CodeLineHeight
	if compact {
		// Box-drawing glyphs must occupy adjacent font rows. The usual
		// code leading leaves gaps in vertical strokes in fonts like Menlo.
		height = 1
	}
	label := "Code"
	if b.Language != "" {
		label += ", " + b.Language
	}
	ui.Column(c).Direction(ui.LeftToRight).Background(th.CodeBlockBackground).
		Border(1, th.CodeBlockBorder).Radius(th.CodeBlockRadius).FillWidth().Clip().Role(ui.RoleGroup).Label(label).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Padding(em*.35, em*.4, 0, pad).Children(func() {
			ui.Text(c, b.Language).FontSize(em * .8).TextColor(th.Muted).Unselectable()
			ui.Box(c).Grow(1)
			b.CopyButton(c)
		})
		if !search {
			b.Body(c).LineHeight(height)
			return
		}
		ui.ScrollHorizontal(c).FillWidth().Font(th.CodeFont).FontSize(em * th.CodeSize).LineHeight(height).Children(func() {
			ui.Column(c).Shrink(0).Padding(em*.25, pad, pad, pad).Children(func() {
				// Code() appends a newline to every source line; Body omits
				// that final synthetic newline, so keep search geometry equal.
				text := strings.TrimSuffix(b.Code(), "\n")
				if text == "" {
					text = " "
				}
				ui.RichText(c, searchSpans(text, d.Query, c.Theme())...).NoWrap()
			})
		})
	})
}
