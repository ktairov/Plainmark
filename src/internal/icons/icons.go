// Package icons embeds the Lucide vectors shared with the original UI.
// Regenerate with node scripts/legacy_icons.mjs; see LICENSE for attribution.
package icons

import (
	"embed"
	"github.com/egoist/mygo/ui"
)

//go:embed svg/*.svg
var files embed.FS

var vectors = func() map[string]*ui.SVG {
	entries, err := files.ReadDir("svg")
	if err != nil {
		panic(err)
	}
	result := make(map[string]*ui.SVG, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		data, err := files.ReadFile("svg/" + name)
		if err != nil {
			panic(err)
		}
		result[name[:len(name)-4]] = ui.MustParseSVG(data)
	}
	return result
}()

func Get(name string) *ui.SVG {
	icon := vectors[name]
	if icon == nil {
		panic("unknown icon: " + name)
	}
	return icon
}
