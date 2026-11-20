// Mathprobe compares the two candidates on the migration formula corpus.
// It is deliberately separate from the application binary.
package main

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"
	texmath "github.com/go-tex/math"
	"github.com/luo-studio/go-tex/tex/layout"
	"github.com/luo-studio/go-tex/tex/parser"
	"github.com/luo-studio/go-tex/tex/render"
)

var corpus = []string{
	`x_i^2 + \frac{1}{2}`,
	`\int_{-\infty}^{\infty} e^{-x^2}\,dx = \sqrt{\pi}`,
	`\begin{pmatrix}a & b \\ c & d\end{pmatrix}`,
	`f(x)=\begin{cases}x^2 & x>0 \\ 0 & x\leq 0\end{cases}`,
	`\begin{aligned}a &= b+c \\ d &= e-f\end{aligned}`,
	`500\text{ GB} \times \$0.015 = \$7.50`,
	`\unknownPlainmarkCommand{x}`,
}

func main() {
	dir := filepath.Join(os.TempDir(), "plainmark-mathprobe")
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		panic(err)
	}
	r, err := texmath.New(texmath.DefaultFont())
	if err != nil {
		panic(err)
	}
	for i, src := range corpus {
		svg, m, err := r.RenderSVGMetrics(src, 32)
		if err == nil {
			_, err = ui.ParseSVG([]byte(svg))
		}
		if err == nil {
			err = os.WriteFile(filepath.Join(dir, fmt.Sprintf("math-%d.svg", i)), []byte(svg), 0600)
		}
		fmt.Printf("go-tex/math [%d] width=%.2f height=%.2f depth=%.2f error=%v\n", i, m.Width, m.Height, m.Depth, err)
		compareLuo(i, src, dir)
	}
}

func compareLuo(i int, src, dir string) {
	defer func() {
		if v := recover(); v != nil {
			fmt.Printf("Luo [%d] panic=%v\n", i, v)
		}
	}()
	nodes, err := parser.Parse(src)
	if err != nil {
		fmt.Printf("Luo [%d] error=%v\n", i, err)
		return
	}
	dl := layout.ToDisplayList(layout.Layout(nodes, layout.DefaultOptions()))
	data, err := render.PNG(dl, render.WithFontSize(32), render.WithScale(2))
	if err == nil {
		_, err = png.Decode(bytes.NewReader(data))
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(dir, fmt.Sprintf("luo-%d.png", i)), data, 0600)
	}
	fmt.Printf("Luo [%d] width=%.2f height=%.2f depth=%.2f error=%v\n", i, dl.Width, dl.Height, dl.Depth, err)
}
