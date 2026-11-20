package desktop

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"plainmark/internal/locales"
	"plainmark/internal/session"
)

var viewModes = [...]struct {
	id, label, accelerator string
	key                    ui.Key
	mode                   session.Mode
}{
	{"preview", "preview", "Alt+1", ui.Key1, session.Preview},
	{"editor", "editor", "Alt+2", ui.Key2, session.Edit},
	{"split", "split", "Alt+3", ui.Key3, session.Split},
}

func (a *App) syncMenu() {
	if a.win == nil {
		return
	}
	d := a.current()
	active := ""
	dirty := false
	mode := session.Preview
	if d != nil {
		active = d.Tab.Note.Path
		dirty = d.Tab.Dirty()
		mode = d.Tab.Mode
	}
	selection := false
	if d != nil {
		for _, s := range d.Code.Selections() {
			if !s.Empty() {
				selection = true
				break
			}
		}
	}
	state := fmt.Sprintf("%t/%t/%s/%t/%d/%d/%t/%s/%t/%t/%t", a.busy, a.store != nil, active, dirty, len(a.session.Tabs), mode, a.sidebarVisible, strings.Join(a.recent, "\x00"), a.pending != nil, a.form != nil, a.deleting != nil || a.crypto != nil || a.settingsOpen)
	state += fmt.Sprintf("/%t/%t", a.crypto != nil || a.settingsOpen, selection)
	state += fmt.Sprintf("/%t", a.appearanceDark)
	state += "/" + a.language() + "/" + a.preferences.ColorMode
	if d != nil {
		state += fmt.Sprintf("/%t", a.pinned()[d.Tab.Note.Path])
	}
	if state == a.menuState {
		return
	}
	a.menuState = state
	a.installMenu()
}
func (a *App) menuText(key string) string {
	return locales.Catalog.Format(a.language(), "menu."+key, nil)
}
func (a *App) contextText(key string) string {
	return locales.Catalog.Format(a.language(), "context."+key, nil)
}
func (a *App) menuBlocked() bool {
	return a.busy || a.pending != nil || a.form != nil || a.deleting != nil || a.crypto != nil || a.settingsOpen
}
func (a *App) menuTemplate() []*mygo.MenuItem {
	return a.menuTemplateForPlatform(runtime.GOOS)
}

func (a *App) menuTemplateForPlatform(goos string) []*mygo.MenuItem {
	d := a.current()
	hasNote := d != nil
	dirty := hasNote && d.Tab.Dirty()
	blocked := a.menuBlocked()
	item := func(id, key, shortcut string, disabled bool, fn func()) *mygo.MenuItem {
		return &mygo.MenuItem{ID: "plainmark." + id, Label: a.menuText(key), Accelerator: shortcut, Disabled: disabled, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.update(func() {
				if !a.menuBlocked() {
					a.closePopup()
					fn()
					a.menuState = "" // Restore checkmarks even when reselecting the current mode.
				}
			})
		}}
	}
	newForm := func(op string) func() { return func() { a.form = &pathForm{operation: op, path: a.activeFolder} } }
	recent := []*mygo.MenuItem{}
	for _, path := range a.recent {
		target := path
		entry := item("recent-folder:"+path, "recentFolders", "", blocked, func() { a.switchTo(target) })
		entry.Label = path
		recent = append(recent, entry)
	}
	if len(recent) == 0 {
		recent = append(recent, &mygo.MenuItem{Label: a.menuText("noRecentFolders"), Disabled: true})
	}
	file := []*mygo.MenuItem{
		item("new-note", "newNote", "CmdOrCtrl+N", blocked || a.store == nil, newForm("note")),
		item("new-folder", "newFolder", "CmdOrCtrl+Shift+N", blocked || a.store == nil, newForm("folder")),
		mygo.Separator(),
		item("open-file", "openFile", "CmdOrCtrl+O", blocked, a.chooseFile),
		item("open-folder", "openFolder", "CmdOrCtrl+Shift+O", blocked, a.chooseFolder),
		{ID: "plainmark.recent-folders", Label: a.menuText("recentFolders"), Submenu: recent, Disabled: blocked},
		mygo.Separator(),
		item("save", "save", "CmdOrCtrl+S", blocked || !dirty, a.save),
		mygo.Separator(),
		item("close-tab", "closeTab", "CmdOrCtrl+W", blocked || !hasNote, func() {
			if d := a.current(); d != nil {
				a.closeTabs([]string{d.Tab.Note.Path})
			}
		}),
	}
	edit := []*mygo.MenuItem{
		{Role: mygo.RoleUndo}, {Role: mygo.RoleRedo}, mygo.Separator(),
		{Role: mygo.RoleCut}, {Role: mygo.RoleCopy}, {Role: mygo.RolePaste}, {Role: mygo.RoleSelectAll}, mygo.Separator(),
		item("find", "find", "CmdOrCtrl+F", blocked || !hasNote, a.find),
		item("search-notes", "searchNotes", "CmdOrCtrl+Shift+F", blocked || a.store == nil, func() { a.sidebarVisible = true; a.focusSearch = true }),
	}
	view := []*mygo.MenuItem{}
	for _, choice := range viewModes {
		entry := item("view-"+choice.id, choice.label, choice.accelerator, blocked || !hasNote, func() {
			if d := a.current(); d != nil {
				d.Tab.Mode = choice.mode
			}
		})
		entry.Type = mygo.MenuItemCheckbox
		entry.Checked = hasNote && d.Tab.Mode == choice.mode
		view = append(view, entry)
	}
	sidebarKey, colorKey := "showSidebar", "nightMode"
	if a.sidebarVisible {
		sidebarKey = "hideSidebar"
	}
	if a.appearanceDark {
		colorKey = "dayMode"
	}
	view = append(view, mygo.Separator(),
		item("toggle-sidebar", sidebarKey, "CmdOrCtrl+B", blocked, func() { a.sidebarVisible = !a.sidebarVisible }),
		item("refresh", "refresh", "CmdOrCtrl+R", blocked || a.store == nil, a.refresh),
		item("toggle-color-mode", colorKey, "", blocked, a.toggleColor),
		mygo.Separator(), &mygo.MenuItem{Role: mygo.RoleToggleFullScreen},
	)
	path, pinKey := "", "pin"
	canEncrypt := false
	if hasNote {
		path = d.Tab.Note.Path
		if a.pinned()[path] {
			pinKey = "unpin"
		}
		if d.Tab.Mode != session.Preview {
			selections := d.Code.Selections()
			canEncrypt = len(selections) > 0 && !selections[0].Empty()
		}
	}
	note := []*mygo.MenuItem{
		item("toggle-pin", pinKey, "", blocked || !hasNote, func() { a.pinned()[path] = !a.pinned()[path] }),
		item("encrypt-selection", "encryptSelection", "", blocked || !canEncrypt, a.beginEncrypt),
		mygo.Separator(),
		item("copy-path", "copyPath", "", blocked || !hasNote, func() { a.pathAction(path, false) }),
		item("copy-relative-path", "copyRelativePath", "", blocked || !hasNote, func() { mygo.Clipboard.WriteText(path) }),
		item("reveal-note", "reveal", "", blocked || !hasNote, func() { a.pathAction(path, true) }),
		mygo.Separator(),
		item("rename", "rename", "", blocked || !hasNote, func() { a.form = &pathForm{operation: "rename", path: path, value: filepath.Base(path)} }),
		item("move", "move", "", blocked || !hasNote, func() { a.form = &pathForm{operation: "move", path: path} }),
		mygo.Separator(),
		item("delete", "delete", "", blocked || !hasNote, func() { a.askDelete(path) }),
	}
	window := []*mygo.MenuItem{
		item("previous-tab", "previousTab", "Ctrl+Shift+Tab", blocked || len(a.session.Tabs) < 2, func() { a.cycleTab(-1) }),
		item("next-tab", "nextTab", "Ctrl+Tab", blocked || len(a.session.Tabs) < 2, func() { a.cycleTab(1) }),
		mygo.Separator(),
		item("close-other-tabs", "closeOtherTabs", "", blocked || !hasNote || len(a.session.Tabs) < 2, a.closeOtherTabs),
		item("close-all-tabs", "closeAllTabs", "", blocked || len(a.session.Tabs) == 0, func() { a.closeTabs(a.allPaths()) }),
		mygo.Separator(), {Role: mygo.RoleMinimize}, {Role: mygo.RoleZoom},
	}
	settings := item("settings", "settings", "CmdOrCtrl+,", blocked, a.openSettings)
	quit := &mygo.MenuItem{ID: "plainmark.quit", Role: mygo.RoleQuit, Label: a.menuText("quit")}
	app := []*mygo.MenuItem{
		{Role: mygo.RoleAbout}, mygo.Separator(),
		settings, mygo.Separator(),
		{Role: mygo.RoleServices}, mygo.Separator(),
		{Role: mygo.RoleHide}, {Role: mygo.RoleHideOthers}, {Role: mygo.RoleUnhide}, mygo.Separator(),
		quit,
	}
	if goos != "darwin" {
		edit = append(edit, mygo.Separator(), settings)
		file = append(file, mygo.Separator(), quit)
	}
	menus := []*mygo.MenuItem{
		{Label: a.menuText("file"), Submenu: file},
		{Label: a.menuText("edit"), Submenu: edit},
		{Label: a.menuText("view"), Submenu: view},
		{Label: a.menuText("note"), Submenu: note},
		{Label: a.menuText("window"), Submenu: window},
	}
	if goos == "darwin" {
		menus = append([]*mygo.MenuItem{{Role: mygo.RoleAppMenu, Submenu: app}}, menus...)
	}
	return menus
}
func (a *App) closeOtherTabs() {
	var paths []string
	for _, tab := range a.session.Tabs {
		if tab.Note.Path != a.session.Active {
			paths = append(paths, tab.Note.Path)
		}
	}
	a.closeTabs(paths)
}
func (a *App) installMenu() {
	if a.win == nil {
		return
	}
	mygo.App.SetMenu(mygo.NewMenu(a.menuTemplate()))
	title := "Plainmark — Go"
	if d := a.current(); d != nil {
		title = d.Tab.Note.Title + " — Plainmark"
		if d.Tab.Dirty() {
			title = "• " + title
		}
	}
	a.win.SetTitle(title)
}
func (a *App) cycleTab(delta int) {
	if len(a.session.Tabs) == 0 {
		return
	}
	index := 0
	for i, t := range a.session.Tabs {
		if t.Note.Path == a.session.Active {
			index = i
			break
		}
	}
	index = (index + delta + len(a.session.Tabs)) % len(a.session.Tabs)
	a.activate(a.session.Tabs[index].Note.Path)
}
