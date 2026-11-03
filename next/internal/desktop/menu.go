package desktop

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"plainmark/internal/session"
)

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
	state := fmt.Sprintf("%t/%t/%s/%t/%d/%d/%t/%s/%t/%t/%t", a.busy, a.store != nil, active, dirty, len(a.session.Tabs), mode, a.sidebarVisible, strings.Join(a.recent, "\x00"), a.pending != nil, a.form != nil, a.deleting != nil || a.crypto != nil)
	state += fmt.Sprintf("/%t/%t", a.crypto != nil, selection)
	if state == a.menuState {
		return
	}
	a.menuState = state
	a.installMenu()
}
func (a *App) installMenu() {
	if a.win == nil {
		return
	}
	d := a.current()
	hasNote := d != nil
	dirty := hasNote && d.Tab.Dirty()
	blocked := a.busy || a.pending != nil || a.form != nil || a.deleting != nil || a.crypto != nil
	item := func(label, key string, disabled bool, fn func()) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Accelerator: key, Disabled: disabled, Click: func(*mygo.MenuItem, *mygo.Window) {
			if !a.busy {
				fn()
			}
		}}
	}
	newForm := func(op string) func() { return func() { a.form = &pathForm{operation: op, path: a.activeFolder} } }
	var recent []*mygo.MenuItem
	for _, p := range a.recent {
		recent = append(recent, item(p, "", blocked, func() { a.switchTo(p) }))
	}
	file := []*mygo.MenuItem{
		item("Open folder…", "CmdOrCtrl+Shift+O", blocked, a.chooseFolder), item("Open file…", "CmdOrCtrl+O", blocked, a.chooseFile),
		{Label: "Recent folders", Submenu: recent},
		item("New note…", "CmdOrCtrl+N", blocked || a.store == nil, newForm("note")), item("New folder…", "", blocked || a.store == nil, newForm("folder")),
		item("Save", "CmdOrCtrl+S", blocked || !dirty, a.save),
		item("Close tab", "CmdOrCtrl+W", blocked || !hasNote, func() {
			if d := a.current(); d != nil {
				a.closeTabs([]string{d.Tab.Note.Path})
			}
		}),
		item("Close all tabs", "", blocked || len(a.session.Tabs) == 0, func() { a.closeTabs(a.allPaths()) }),
	}
	view := []*mygo.MenuItem{}
	for _, choice := range []struct {
		name string
		key  string
		mode session.Mode
	}{{"Preview", "CmdOrCtrl+1", session.Preview}, {"Edit", "CmdOrCtrl+2", session.Edit}, {"Split", "CmdOrCtrl+3", session.Split}} {
		view = append(view, item(choice.name, choice.key, blocked || !hasNote, func() {
			if d := a.current(); d != nil {
				d.Tab.Mode = choice.mode
			}
		}))
	}
	view = append(view, item("Find", "CmdOrCtrl+F", blocked || !hasNote, a.find), item("Encrypt selection", "CmdOrCtrl+Shift+E", blocked || !hasNote || d.Tab.Mode == session.Preview || len(d.Code.Selections()) == 0 || d.Code.Selections()[0].Empty(), a.beginEncrypt))
	view = append(view, item("Toggle sidebar", "CmdOrCtrl+B", blocked, func() { a.sidebarVisible = !a.sidebarVisible }), item("Refresh", "CmdOrCtrl+R", blocked || a.store == nil, a.refresh))
	view = append(view, item("Previous tab", "Ctrl+Shift+Tab", blocked || !hasNote, func() { a.cycleTab(-1) }), item("Next tab", "Ctrl+Tab", blocked || !hasNote, func() { a.cycleTab(1) }))
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleAppMenu}, {Label: "File", Submenu: file}, {Role: mygo.RoleEditMenu}, {Label: "View", Submenu: view}, {Role: mygo.RoleWindowMenu}}))
	title := "Plainmark — Go"
	if d != nil {
		title = d.Tab.Note.Title + " — Plainmark"
		if dirty {
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
