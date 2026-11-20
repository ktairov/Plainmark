package locales

import "testing"

func TestAppAndNativeRussianTranslations(t *testing.T) {
	for _, text := range []string{"Open file", "Save", "Settings", "Password", "Encrypt selection", "Color mode"} {
		got := Text("ru", text)
		if got == text || got == "" {
			t.Errorf("missing translation: %q", text)
		}
	}
	if got := Text("ru", "Opened {path}", "path", "note.md"); got != "Открыто: note.md" && got != "Открыто note.md" {
		t.Fatal(got)
	}
	if got := Catalog.Format("ru", "mygo.copy", nil); got != "Копировать" {
		t.Fatal(got)
	}
	if got := Text("en", "Open file"); got != "Open file" {
		t.Fatal(got)
	}
}
