package settings

import "testing"

func TestParseDefaultsClampsAndSystemTheme(t *testing.T) {
	p := Parse([]byte(`{"language":"ru","colorMode":"system","contentFontSize":99,"previewWidth":3,"dayTheme":"missing"}`))
	if p.Language != "ru" || p.ContentFontSize != 26 || p.PreviewWidth != 560 || p.DayTheme != "ayu" || p.TreeFontSize != 13 {
		t.Fatalf("%+v", p)
	}
	if !p.Dark(true) || p.Dark(false) {
		t.Fatal("system appearance ignored")
	}
	if Parse([]byte(`broken`)) != Defaults() {
		t.Fatal("broken settings lost defaults")
	}
	for _, id := range ThemeOrder {
		theme, ok := Themes[id]
		if !ok || theme.Light.Page == "" || theme.Dark.Text == "" {
			t.Fatal("palette missing", id)
		}
	}
}
