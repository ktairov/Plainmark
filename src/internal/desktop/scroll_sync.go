package desktop

import (
	"math"

	"github.com/egoist/mygo/ui"
)

type scrollBlock struct {
	line int
	y    float64
}

type splitScrollState struct {
	ready             bool
	editorY, previewY float64
	contentY          float64
	blocks            []scrollBlock
}

// syncSplitScroll follows whichever pane moved. Remembering the applied
// positions prevents programmatic scrolling from bouncing back as user input.
// The first frame preserves the independently restored positions of the tab.
func (d *Document) syncSplitScroll(c *ui.Context) {
	s := &d.splitScroll
	editorY, previewY := d.Code.ScrollY(), float64(d.Scroll.Y)
	if !s.ready || len(s.blocks) == 0 {
		s.ready = d.pendingEditorY == nil
		s.editorY, s.previewY = editorY, previewY
		return
	}
	if math.Abs(editorY-s.editorY) > .5 {
		d.Scroll.Y = float32(d.mapSplitScroll(editorY, false))
		c.After(0)
	} else if math.Abs(previewY-s.previewY) > .5 {
		d.Code.SetScrollY(d.mapSplitScroll(previewY, true))
		c.After(0)
	}
	s.editorY, s.previewY = d.Code.ScrollY(), float64(d.Scroll.Y)
}

// Map between rendered block starts and the same source lines in the editor.
// Interpolating within blocks also covers long paragraphs, lists and code.
func (d *Document) mapSplitScroll(y float64, fromPreview bool) float64 {
	s := &d.splitScroll
	editorEnd := d.Code.LineTop(max(0, d.Code.Buffer.LineCount()-1))
	previewEnd := float64(d.Scroll.MaxY)
	x0, y0 := 0.0, 0.0
	for _, block := range s.blocks {
		editor := min(editorEnd, d.Code.LineTop(block.line))
		// The preview scroll container has 24 DIPs of top padding.
		preview := max(0, min(previewEnd, block.y-s.contentY+24))
		x1, y1 := editor, preview
		if fromPreview {
			x1, y1 = preview, editor
		}
		if x1 <= x0 || y1 < y0 {
			continue
		}
		if y <= x1 {
			return interpolateScroll(y, x0, x1, y0, y1)
		}
		x0, y0 = x1, y1
	}
	x1, y1 := editorEnd, previewEnd
	if fromPreview {
		x1, y1 = previewEnd, editorEnd
	}
	return interpolateScroll(y, x0, x1, y0, y1)
}

func interpolateScroll(y, x0, x1, y0, y1 float64) float64 {
	if x1 <= x0 {
		return y1
	}
	return y0 + max(0, min(1, (y-x0)/(x1-x0)))*(y1-y0)
}
