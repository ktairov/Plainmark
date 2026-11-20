package desktop

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
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
	if !tt.HasText("Настройки") || !tt.HasText(a.t("Toggle sidebar")) {
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

func TestSettingsControlsAndThemePreview(t *testing.T) {
	a, _ := workspaceApp(t)
	before := a.preferences
	tt := ui.NewTester(a.view, 1100, 800)
	tt.Frame()
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	menuByID(a.menuTemplate(), "settings").Click(nil, nil)
	tt.Frame()
	if !a.settingsOpen || a.settingsSection != "appearance" {
		t.Fatal("application menu did not open appearance settings")
	}
	click("Night")
	if a.preferences.ColorMode != "dark" {
		t.Fatal("color-mode control did not update preferences")
	}
	click("16 px")
	click("20 px")
	if a.preferences.ContentFontSize != 20 {
		t.Fatal("font-size control did not update preferences")
	}
	click("Theme")
	if dir := os.Getenv("PLAINMARK_SNAPSHOTS"); dir != "" {
		saveFrame(t, tt, dir, "settings-themes-initial.png")
	}
	tt.Scroll(800, 500, 0, 300)
	tt.Frame()
	click("Night theme: Nord")
	if a.preferences.NightTheme != "nord" || a.settingsPreviewMode != "dark" {
		t.Fatalf("night palette did not preview: theme=%s preview=%s", a.preferences.NightTheme, a.settingsPreviewMode)
	}
	tt.Scroll(800, 350, 0, -400)
	tt.Frame()
	click("Day theme: Gruvbox")
	if a.preferences.DayTheme != "gruvbox" || a.settingsPreviewMode != "light" || a.preferences.ColorMode != "dark" {
		t.Fatal("day palette preview changed chosen color mode")
	}
	if dir := os.Getenv("PLAINMARK_SNAPSHOTS"); dir != "" {
		saveFrame(t, tt, dir, "settings-themes.png")
	}
	click("Language")
	click("System")
	click("Русский")
	if a.preferences.Language != "ru" || !tt.HasText("Используется системный язык") {
		t.Fatal("language did not update immediately")
	}
	click(a.t("Cancel"))
	if a.settingsOpen || a.preferences != before || a.settingsPreviewMode != "" {
		t.Fatal("cancel did not restore preferences and clear preview")
	}
	a.openSettings()
	tt.Frame()
	click("Theme")
	click("Day theme: Nord")
	click("Apply")
	if a.settingsOpen || a.preferences.DayTheme != "nord" || a.preferences.ColorMode != before.ColorMode || a.settingsPreviewMode != "" {
		t.Fatal("apply did not preserve color mode or clear palette preview")
	}
}

func TestToolbarColorIsLocalAndPopupEscapeKeepsSettingsOpen(t *testing.T) {
	a, _ := workspaceApp(t)
	a.preferences.ColorMode = "light"
	tt := ui.NewTester(a.view, 1100, 800)
	tt.Frame()
	for _, text := range tt.Texts() {
		if text == "Settings" || text == "More" {
			t.Fatal("settings/more remain in toolbar")
		}
	}
	if err := tt.Click("Toggle color mode"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.appearanceDark || a.preferences.ColorMode != "light" || a.windowColorOverride != "dark" {
		t.Fatal("toolbar changed the configured color mode")
	}
	data, err := a.captureState()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "windowColorOverride") {
		t.Fatal("window override leaked into saved settings")
	}
	a.openSettings()
	tt.Frame()
	if a.appearanceDark {
		t.Fatal("settings should preview the configured light mode")
	}
	if err := tt.Click("Language"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("System"); err != nil {
		t.Fatal(err)
	}
	if a.popup == nil {
		t.Fatal("language popup did not open")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.popup != nil || !a.settingsOpen {
		t.Fatal("Escape should close the popup first")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.settingsOpen || !a.appearanceDark || a.preferences.ColorMode != "light" {
		t.Fatal("cancelling settings lost the local window theme")
	}
	a.openSettings()
	tt.Frame()
	if err := tt.Click("Apply"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.windowColorOverride != "" || a.appearanceDark {
		t.Fatal("applying configured settings did not clear the temporary override")
	}
}

func TestSystemAppearanceResumesAfterLocalToggle(t *testing.T) {
	a, _ := workspaceApp(t)
	a.preferences.ColorMode = "system"
	tt := ui.NewTester(a.view, 1100, 800)
	tt.SetDark(false)
	tt.Frame()
	if a.appearanceDark {
		t.Fatal("automatic mode did not follow system light appearance")
	}
	if err := tt.Click("Toggle color mode"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.appearanceDark || a.windowColorOverride != "dark" {
		t.Fatal("local dark override was not applied")
	}
	tt.SetDark(true)
	if !a.appearanceDark || a.windowColorOverride != "" {
		t.Fatal("system dark event did not resume automatic appearance")
	}
	tt.SetDark(false)
	if a.appearanceDark || a.preferences.ColorMode != "system" {
		t.Fatal("system light event did not update the window")
	}
	// Appearance events also arrive while settings preview a different palette.
	a.toggleColor()
	a.openSettings()
	a.settingsPreviewMode = "light"
	tt.Frame()
	tt.SetDark(true)
	if a.appearanceDark || a.windowColorOverride != "" {
		t.Fatal("system event lost the settings preview or retained the override")
	}
	a.closeSettings(false)
	tt.Frame()
	if !a.appearanceDark {
		t.Fatal("Cancel did not return to the current system appearance")
	}
}

func TestExplicitAppearanceSurvivesSystemChanges(t *testing.T) {
	for _, mode := range []string{"light", "dark"} {
		t.Run(mode, func(t *testing.T) {
			a, _ := workspaceApp(t)
			a.preferences.ColorMode = mode
			tt := ui.NewTester(a.view, 1100, 800)
			tt.SetDark(false)
			tt.Frame()
			tt.SetDark(true)
			if a.appearanceDark != (mode == "dark") {
				t.Fatal("system event replaced an explicit preference")
			}
			a.toggleColor()
			tt.Frame()
			tt.SetDark(false)
			if a.appearanceDark != (mode == "light") || a.windowColorOverride == "" || a.preferences.ColorMode != mode {
				t.Fatal("system event replaced a local override of an explicit preference")
			}
		})
	}
}

func TestCreatePopupKeyboardAndOutsideDismissal(t *testing.T) {
	a, _ := workspaceApp(t)
	tt := ui.NewTester(a.view, 880, 620)
	tt.Frame()
	if err := tt.Click("Create"); err != nil {
		t.Fatal(err)
	}
	if a.popup == nil {
		t.Fatal("create popup did not open")
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.popup != nil || a.form == nil || a.form.operation != "folder" {
		t.Fatal("keyboard did not choose New folder")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.form != nil {
		t.Fatal("Escape did not close the styled dialog")
	}
	if err := tt.Click("Create"); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(700, 500)
	if a.popup != nil {
		t.Fatal("outside click did not dismiss popup")
	}
}
