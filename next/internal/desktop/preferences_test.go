package desktop

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"plainmark/internal/session"
	"plainmark/internal/settings"
)

func TestSettingsPreviewCancelAndApply(t *testing.T) {
	a, _ := workspaceApp(t)
	statePath := filepath.Join(t.TempDir(), "state.json")
	a.writer = settings.NewWriter(statePath)
	defer a.writer.Close()
	before := a.preferences
	tt := ui.NewTester(a.view, 1220, 820)
	tt.Frame()
	a.openSettings()
	tt.Frame()
	a.preferences.ColorMode = "dark"
	a.preferences.Language = "ru"
	a.preferences.ContentFontSize = 21
	tt.SetLocale("ru")
	tt.Frame()
	if !tt.HasText("Настройки") || !tt.HasText(a.t("Open file")) {
		t.Fatal("Russian settings or app UI missing", tt.Texts())
	}
	if err := tt.Click(a.t("Cancel")); err != nil {
		t.Fatal(err)
	}
	if a.preferences != before || a.settingsOpen {
		t.Fatal("cancel did not restore preferences")
	}
	a.openSettings()
	a.preferences.ColorMode = "light"
	a.preferences.ReadingFont = "serif"
	tt.Frame()
	if err := tt.Click(a.t("Apply")); err != nil {
		t.Fatal(err)
	}
	if a.settingsOpen || a.preferences.ReadingFont != "serif" || a.preferences.ColorMode != "light" {
		t.Fatal("apply discarded preferences")
	}
	if err := a.writer.Close(); err != nil {
		t.Fatal(err)
	}
	loaded, err := settings.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Preferences.ReadingFont != "serif" || loaded.Preferences.ColorMode != "light" || loaded.Preferences.Language != before.Language {
		t.Fatal("Apply/Cancel was not persisted correctly")
	}
}

func TestRussianEditorFindAndSettingsSnapshot(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err = a.load("../../../testdata/workspace/Math.md"); err != nil {
		t.Fatal(err)
	}
	a.preferences.Language = "ru"
	a.preferences.ColorMode = "dark"
	a.current().Tab.Mode = session.Edit
	tt := ui.NewTester(a.view, 1220, 820)
	tt.SetLocale("ru")
	tt.Frame()
	a.find()
	tt.Frame()
	if !tt.HasText("Найти") || !tt.HasText("Заменить") {
		t.Fatal("native editor find bar not translated", tt.Texts())
	}
	a.current().Tab.Mode = session.Preview
	a.openSettings()
	tt.Frame()
	if dir := os.Getenv("PLAINMARK_SNAPSHOTS"); dir != "" {
		saveFrame(t, tt, dir, "settings-ru.png")
	}
}
func saveFrame(t *testing.T, tt *ui.Tester, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, tt.Image())
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
