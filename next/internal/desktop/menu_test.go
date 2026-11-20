package desktop

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"plainmark/internal/session"
)

func menuEntries(items []*mygo.MenuItem) []string {
	var out []string
	for _, item := range items {
		switch {
		case item.Type == mygo.MenuItemSeparator:
			out = append(out, "-")
		case item.ID != "":
			out = append(out, strings.TrimPrefix(item.ID, "plainmark."))
		default:
			out = append(out, string(item.Role))
		}
	}
	return out
}

func menuByID(items []*mygo.MenuItem, id string) *mygo.MenuItem {
	for _, item := range items {
		if item.ID == "plainmark."+id {
			return item
		}
		if found := menuByID(item.Submenu, id); found != nil {
			return found
		}
	}
	return nil
}

func TestNativeMenuMatchesOriginalStructureAndState(t *testing.T) {
	a, _ := workspaceApp(t)
	// Order and separators from preserved src-tauri/src/menu.rs.
	want := [][]string{
		{"about", "-", "settings", "-", "services", "-", "hide", "hideOthers", "unhide", "-", "quit"},
		{"new-note", "new-folder", "-", "open-file", "open-folder", "recent-folders", "-", "save", "-", "close-tab"},
		{"undo", "redo", "-", "cut", "copy", "paste", "selectAll", "-", "find", "search-notes"},
		{"view-preview", "view-editor", "view-split", "-", "toggle-sidebar", "refresh", "toggle-color-mode", "-", "togglefullscreen"},
		{"toggle-pin", "encrypt-selection", "-", "copy-path", "copy-relative-path", "reveal-note", "-", "rename", "move", "-", "delete"},
		{"previous-tab", "next-tab", "-", "close-other-tabs", "close-all-tabs", "-", "minimize", "zoom"},
	}
	template := a.menuTemplate()
	if len(template) != len(want) {
		t.Fatal("wrong top-level menu count")
	}
	for i, group := range template {
		if got := menuEntries(group.Submenu); !slices.Equal(got, want[i]) {
			t.Fatalf("menu %d: %v; want %v", i, got, want[i])
		}
	}
	if !menuByID(template, "save").Disabled || !menuByID(template, "next-tab").Disabled || !menuByID(template, "encrypt-selection").Disabled {
		t.Fatal("clean/single/preview states are not disabled")
	}
	if !menuByID(template, "view-preview").Checked {
		t.Fatal("current mode not checked")
	}
	if menuByID(template, "find").Accelerator != "CmdOrCtrl+F" || menuByID(template, "settings").Accelerator != "CmdOrCtrl+," {
		t.Fatal("shortcuts changed")
	}
	a.pinned()["a.md"] = true
	a.sidebarVisible = false
	a.appearanceDark = true
	a.preferences.Language = "ru"
	template = a.menuTemplate()
	for id, label := range map[string]string{"toggle-pin": "Открепить", "toggle-sidebar": "Показать боковую панель", "toggle-color-mode": "Дневной режим", "settings": "Настройки…"} {
		if item := menuByID(template, id); item.Label != label {
			t.Fatalf("%s: %q", id, item.Label)
		}
	}
	a.preferences.Language = "en"
	a.openNote("b.md", "")
	a.current().Tab.Mode = session.Edit
	a.current().Code.Select(0, 3)
	template = a.menuTemplate()
	if menuByID(template, "next-tab").Disabled || menuByID(template, "encrypt-selection").Disabled {
		t.Fatal("multi-tab/selection actions remained disabled")
	}
	a.settingsOpen = true
	template = a.menuTemplate()
	if !menuByID(template, "rename").Disabled || !menuByID(template, "find").Disabled {
		t.Fatal("document commands remained enabled under a modal")
	}
}

func TestViewModeShortcuts(t *testing.T) {
	a, _ := workspaceApp(t)
	tt := ui.NewTester(a.view, 1220, 820)
	tt.Frame()
	for _, step := range []struct {
		key  ui.Key
		mode session.Mode
	}{
		{ui.Key2, session.Edit},
		{ui.Key3, session.Split},
		{ui.Key1, session.Preview},
	} {
		tt.Key(ui.Alt, step.key)
		if got := a.current().Tab.Mode; got != step.mode {
			t.Fatalf("shortcut %v set mode %v, want %v", step.key, got, step.mode)
		}
	}
	a.openSettings()
	tt.Frame()
	tt.Key(ui.Alt, ui.Key2)
	if a.current().Tab.Mode != session.Preview {
		t.Fatal("mode shortcut ran behind settings")
	}
}

func TestContextMenusMatchOriginalAndOpenNewTab(t *testing.T) {
	a, _ := workspaceApp(t)
	path, folder := "b.md", false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(200, 40).Label("target").ContextMenu(func(m *ui.Menu) { a.pathMenu(m, path, folder) })
	}, 400, 300)
	tt.Frame()
	if err := tt.RightClick("target"); err != nil {
		t.Fatal(err)
	}
	want := []string{"Copy path", "Open in new tab", "Pin", "Rename", "Move", "Move to Trash…"}
	if got := tt.Menu(); !slices.Equal(got, want) {
		t.Fatalf("note context: %q", got)
	}
	if err := tt.ChooseMenuItem("Open in new tab"); err != nil {
		t.Fatal(err)
	}
	if len(a.session.Tabs) != 2 || a.session.Active != "b.md" {
		t.Fatal("context navigation replaced the current tab")
	}
	if err := tt.RightClick("target"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Pin"); err != nil {
		t.Fatal(err)
	}
	if !a.pinned()["b.md"] {
		t.Fatal("pin action failed")
	}
	if err := tt.RightClick("target"); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(tt.Menu(), "Unpin") {
		t.Fatal("pin label did not change")
	}
	tt.CloseMenu()
	path, folder = "folder", true
	tt.Frame()
	if err := tt.RightClick("target"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Menu(); !slices.Equal(got, []string{"New note", "New folder", "Rename", "Move", "Move to Trash…"}) {
		t.Fatalf("folder context: %q", got)
	}
	tt.CloseMenu()
	path, folder = "", true
	tt.Frame()
	if err := tt.RightClick("target"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Menu(); !slices.Equal(got, []string{"New note", "New folder"}) {
		t.Fatalf("root context: %q", got)
	}
}
