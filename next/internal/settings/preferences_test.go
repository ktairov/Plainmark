package settings

import "testing"

func TestParseDefaultsClampsAndSystemTheme(t *testing.T) {
	p := Parse([]byte(`{"language":"ru","colorMode":"system","contentFontSize":99,"previewWidth":3,"dayTheme":"missing"}`))
	if p.Language != "ru" || p.ContentFontSize != 26 || p.PreviewWidth != 560 || p.DayTheme != "ayu" || p.NightTheme != "mariana" || p.TreeFontSize != 13 {
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

func TestRemovedThemeAndPortableFontPreferences(t *testing.T) {
	p := Parse([]byte(`{"dayTheme":"graphite","nightTheme":"graphite","uiFont":"go-mono","readingFont":"family:Unavailable Test Font"}`))
	if len(ThemeOrder) != 6 || len(Themes) != 6 || p.DayTheme != "ayu" || p.NightTheme != "mariana" {
		t.Fatalf("removed theme did not fall back: %+v", p)
	}
	if p.UIFont != "go-mono" || Font(p.UIFont) != "Go Mono" {
		t.Fatalf("bundled font was not preserved: %q", p.UIFont)
	}
	if p.ReadingFont != "family:Unavailable Test Font" || Font(p.ReadingFont) != "Go Sans" {
		t.Fatalf("missing font did not fall back: %q", p.ReadingFont)
	}
}
