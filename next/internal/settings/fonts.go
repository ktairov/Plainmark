package settings

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/go-fonts/latin-modern/lmroman10bold"
	"github.com/go-fonts/latin-modern/lmroman10bolditalic"
	"github.com/go-fonts/latin-modern/lmroman10italic"
	"github.com/go-fonts/latin-modern/lmroman10regular"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
)

type FontChoice struct{ ID, Name string }

const familyPrefix = "family:"

var bundledFonts = [...]struct {
	FontChoice
	faces [4][]byte
}{
	{FontChoice{"go-sans", "Go Sans"}, [4][]byte{goregular.TTF, gobold.TTF, goitalic.TTF, gobolditalic.TTF}},
	{FontChoice{"go-mono", "Go Mono"}, [4][]byte{gomono.TTF, gomonobold.TTF, gomonoitalic.TTF, gomonobolditalic.TTF}},
	{FontChoice{"latin-modern", "Latin Modern Roman"}, [4][]byte{lmroman10regular.TTF, lmroman10bold.TTF, lmroman10italic.TTF, lmroman10bolditalic.TTF}},
}

var registerFonts = sync.OnceValue(func() error {
	for _, font := range bundledFonts {
		for _, face := range font.faces {
			if err := ui.RegisterFont(face, font.Name); err != nil {
				return fmt.Errorf("register %s: %w", font.Name, err)
			}
		}
	}
	return nil
})

func RegisterFonts() error { return registerFonts() }

var systemFamilies = sync.OnceValues(func() ([]FontChoice, map[string]string) {
	var choices []FontChoice
	names := make(map[string]string)
	for _, family := range ui.FontFamilies() {
		names[strings.ToLower(family.Name)] = family.Name
		if !family.Registered {
			choices = append(choices, FontChoice{familyPrefix + family.Name, family.Name})
		}
	}
	return choices, names
})

func FontChoices() []FontChoice {
	system, _ := systemFamilies()
	choices := make([]FontChoice, 0, len(bundledFonts)+len(system))
	for _, font := range bundledFonts {
		choices = append(choices, font.FontChoice)
	}
	return append(choices, system...)
}

func validFont(id string) bool {
	switch id {
	case "system", "humanist", "serif", "mono":
		return true
	}
	for _, font := range bundledFonts {
		if font.ID == id {
			return true
		}
	}
	return strings.HasPrefix(id, familyPrefix) && strings.TrimSpace(strings.TrimPrefix(id, familyPrefix)) != ""
}

func Font(id string) string {
	for _, font := range bundledFonts {
		if font.ID == id {
			return font.Name
		}
	}
	if id == "system" {
		return ""
	}
	_, names := systemFamilies()
	var preferred, fallback string
	switch id {
	case "humanist":
		preferred, fallback = "Segoe UI", "Go Sans"
		if runtime.GOOS == "darwin" {
			preferred = "Avenir Next"
		}
	case "serif":
		preferred, fallback = "Georgia", "Latin Modern Roman"
		if runtime.GOOS == "darwin" {
			preferred = "Charter"
		}
	case "mono":
		preferred, fallback = "Consolas", "Go Mono"
		if runtime.GOOS == "darwin" {
			preferred = "Menlo"
		}
	default:
		if !strings.HasPrefix(id, familyPrefix) {
			return ""
		}
		preferred, fallback = strings.TrimPrefix(id, familyPrefix), "Go Sans"
	}
	if name := names[strings.ToLower(preferred)]; name != "" {
		return name
	}
	return fallback
}
