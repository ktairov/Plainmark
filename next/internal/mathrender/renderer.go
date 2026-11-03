// Package mathrender adapts TeX layout to MyGo's baseline-aware Markdown API.
package mathrender

import (
	"container/list"
	"fmt"
	"math"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	texmath "github.com/go-tex/math"
)

type Formula struct {
	SVG                  *ui.SVG
	Width, Height, Depth float32
}

type key struct {
	tex     string
	size    int
	display bool
}
type entry struct {
	key     key
	formula Formula
	err     error
}

// Renderer serializes access to the typesetter and keeps a bounded LRU. SVGs
// use currentColor: theme changes repaint the same geometry without re-layout.
type Renderer struct {
	mu       sync.Mutex
	engine   *texmath.Renderer
	cache    map[key]*list.Element
	order    *list.List
	capacity int
}

func New() (*Renderer, error) {
	e, err := texmath.New(texmath.DefaultFont())
	if err != nil {
		return nil, err
	}
	return &Renderer{engine: e, cache: make(map[key]*list.Element), order: list.New(), capacity: 256}, nil
}

func (r *Renderer) Render(tex string, size float32, display bool) (f Formula, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key{tex, max(1, int(math.Round(float64(size)))), display}
	if el := r.cache[k]; el != nil {
		r.order.MoveToFront(el)
		e := el.Value.(entry)
		return e.formula, e.err
	}
	// Malformed user input must never take down the document window, including
	// a panic from a third-party parser. Failures are cached as well.
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("invalid formula: %v", v)
		}
		r.cache[k] = r.order.PushFront(entry{k, f, err})
		if r.order.Len() > r.capacity {
			last := r.order.Back()
			delete(r.cache, last.Value.(entry).key)
			r.order.Remove(last)
		}
	}()
	var svg string
	var m texmath.Metrics
	if display {
		svg, m, err = r.engine.RenderDisplaySVGMetrics(tex, k.size)
	} else {
		svg, m, err = r.engine.RenderSVGMetrics(tex, k.size)
	}
	if err != nil {
		return Formula{}, err
	}
	s, err := ui.ParseSVG([]byte(svg))
	if err != nil {
		return Formula{}, err
	}
	return Formula{s, float32(m.Width), float32(m.Height), float32(m.Depth)}, nil
}

func (r *Renderer) Configure(o *markdown.Options, color ui.Color, size float32) {
	o.Math = func(c *ui.Context, tex string) {
		f, err := r.Render(tex, size, true)
		if err != nil {
			ui.Text(c, tex).Selectable().Tooltip(err.Error())
			return
		}
		ui.ScrollHorizontal(c).FillWidth().Children(func() {
			ui.Icon(c, f.SVG).Width(f.Width).Height(f.Height + f.Depth).TextColor(c.Theme().Text).Label(tex)
		})
	}
	o.InlineMath = func(tex string, size float32) (markdown.InlineObject, bool) {
		f, err := r.Render(tex, size, false)
		if err != nil {
			return markdown.InlineObject{}, false
		}
		return markdown.InlineObject{Width: f.Width, Height: f.Height, Depth: f.Depth,
			Draw: func(p *ui.Painter, box ui.Rect) { p.Icon(f.SVG, box, color) }}, true
	}
}
