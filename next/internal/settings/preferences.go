package settings

import (
	_ "embed"
	"encoding/json"
	"slices"
)

type Preferences struct {
	Language        string `json:"language"`
	ColorMode       string `json:"colorMode"`
	DayTheme        string `json:"dayTheme"`
	NightTheme      string `json:"nightTheme"`
	UIFont          string `json:"uiFont"`
	ReadingFont     string `json:"readingFont"`
	TreeFontSize    int    `json:"treeFontSize"`
	ContentFontSize int    `json:"contentFontSize"`
	PreviewWidth    int    `json:"previewWidth"`
}

func Defaults() Preferences {
	return Preferences{"system", "system", "ayu", "mariana", "system", "system", 13, 16, 860}
}
func Parse(data []byte) Preferences {
	p := Defaults()
	if json.Unmarshal(data, &p) != nil {
		return Defaults()
	}
	p.Normalize()
	return p
}
func (p *Preferences) Normalize() {
	d := Defaults()
	if !slices.Contains([]string{"system", "en", "ru"}, p.Language) {
		p.Language = d.Language
	}
	if !slices.Contains([]string{"system", "light", "dark"}, p.ColorMode) {
		p.ColorMode = d.ColorMode
	}
	if _, ok := Themes[p.DayTheme]; !ok {
		p.DayTheme = d.DayTheme
	}
	if _, ok := Themes[p.NightTheme]; !ok {
		p.NightTheme = d.NightTheme
	}
	for _, value := range []*string{&p.UIFont, &p.ReadingFont} {
		if !validFont(*value) {
			*value = "system"
		}
	}
	p.TreeFontSize = min(max(p.TreeFontSize, 13), 21)
	p.ContentFontSize = min(max(p.ContentFontSize, 13), 26)
	p.PreviewWidth = min(max(p.PreviewWidth, 560), 1400)
}
func (p Preferences) Dark(system bool) bool {
	if p.ColorMode == "dark" {
		return true
	}
	if p.ColorMode == "light" {
		return false
	}
	return system
}

type Palette struct{ Page, Control, Code, Text, TextSecondary, Muted, Border, Accent, AccentSoft, AccentContrast, Selected, Selection, Heading, Link, Success, Warning, Danger string }
type Theme struct {
	ID, Label   string
	Light, Dark Palette
}

//go:embed themes.json
var themeData []byte
var Themes = func() map[string]Theme {
	var t map[string]Theme
	if err := json.Unmarshal(themeData, &t); err != nil {
		panic(err)
	}
	return t
}()
var ThemeOrder = []string{"solarized", "gruvbox", "catppuccin", "ayu", "nord", "mariana"}
