package desktop

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/markdown/parser"

	"plainmark/internal/mathrender"
	"plainmark/internal/preview"
	"plainmark/internal/workspace"
)

type App struct {
	win            *mygo.Window
	code           *editor.State
	md             markdown.Options
	math           *mathrender.Renderer
	path           string
	store          *workspace.Workspace
	note           workspace.Note
	mode           int
	wrap           bool
	busy           bool
	status         string
	scroll         ui.ScrollState
	ratio          float32
	previewBuffer  *editor.Buffer
	previewVersion uint64
	previewSource  string
	// run dispatches slow work off the UI thread; tests can replace it.
	run func(func())
}

func New() (*App, error) {
	m, err := mathrender.New()
	if err != nil {
		return nil, err
	}
	a := &App{math: m, mode: 2, wrap: true, ratio: 50, status: "Open a Markdown file to begin"}
	a.setDocument("", nil)
	a.run = func(fn func()) { go fn() }
	return a, nil
}

func (a *App) setDocument(path string, data []byte) {
	a.path = path
	b := editor.NewBuffer(string(data))
	b.SetLanguage(editor.LanguageFor("note.md"))
	b.MarkSaved()
	a.code = &editor.State{Buffer: b, SoftWrap: true}
	a.md = markdown.Options{Extensions: parser.Default | parser.ExtDollarMath}
	a.scroll = ui.ScrollState{}
}

func (a *App) update(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
	} else {
		fn()
	}
}

func (a *App) start(fn func() error) {
	if a.busy {
		return
	}
	a.busy = true
	a.run(func() {
		err := fn()
		a.update(func() {
			a.busy = false
			if err != nil {
				a.status = err.Error()
			}
		})
	})
}

func (a *App) confirmDiscard() bool {
	var dirty bool
	a.update(func() { dirty = a.code.Buffer.Modified() })
	if !dirty {
		return true
	}
	r, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: a.win, Type: mygo.MessageWarning,
		Message: "Discard unsaved changes?", Buttons: []string{"Cancel", "Discard"}, CancelButton: 0, DefaultButton: 0, DestructiveButtons: []int{1}})
	return err == nil && r.Button == 1
}

func (a *App) chooseFile() {
	a.start(func() error {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Filters: []mygo.FileFilter{{Name: "Markdown", Extensions: []string{"md", "mdx"}}}})
		if err != nil || len(paths) == 0 {
			return err
		}
		if !a.confirmDiscard() {
			return nil
		}
		return a.load(paths[0])
	})
}

func (a *App) load(path string) error {
	if !workspace.IsMarkdown(path) {
		return errors.New("Only Markdown files can be opened")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	store, err := workspace.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	note, err := store.Read(filepath.Base(path))
	if err != nil {
		return err
	}
	a.update(func() {
		a.setDocument(path, []byte(note.Markdown))
		a.store, a.note = store, note
		a.status = "Opened " + path
	})
	return nil
}

func (a *App) save() {
	if a.path == "" || a.busy || a.store == nil {
		return
	}
	store, note := a.store, a.note
	submitted := a.code.Buffer.String()
	buf := a.code.Buffer
	a.start(func() error {
		saved, err := store.Save(note.Path, submitted, note.Revision, false)
		if err != nil {
			if workspace.IsCode(err, "conflict") {
				return fmt.Errorf("Conflict: %w", err)
			}
			return err
		}
		a.update(func() {
			a.note = saved
			if a.code.Buffer == buf && buf.String() == submitted {
				buf.MarkSaved()
			}
			a.status = "Saved " + a.path
		})
		return nil
	})
}

func (a *App) preview(c *ui.Context) {
	a.math.Configure(&a.md, c.Theme().Text, 20)
	if a.previewBuffer != a.code.Buffer || a.previewVersion != a.code.Buffer.Version() {
		a.previewBuffer, a.previewVersion = a.code.Buffer, a.code.Buffer.Version()
		a.previewSource = preview.Normalize(a.code.Buffer.String())
	}
	ui.Scroll(c).TrackScroll(&a.scroll).Grow(1).FillWidth().Padding(20).Children(func() {
		markdown.View(c, a.previewSource, &a.md).FillWidth()
	})
}

func (a *App) edit(c *ui.Context) {
	a.code.SoftWrap = a.wrap
	editor.View(c, a.code).Key("editor").Label("Markdown editor").Grow(1).FillWidth()
}

func (a *App) view(c *ui.Context) {
	t := c.Theme()
	root := ui.Column(c).Fill()
	if root.Shortcut(ui.Cmd, ui.KeyS) {
		a.save()
	}
	if root.Shortcut(ui.Cmd, ui.KeyO) {
		a.chooseFile()
	}
	root.Children(func() {
		ui.Row(c).Padding(8).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			if ui.Button(c, "Open file").Key("open").Disabled(a.busy).Clicked() {
				a.chooseFile()
			}
			if ui.PrimaryButton(c, "Save").Key("save").Disabled(a.path == "" || a.busy || !a.code.Buffer.Modified()).Clicked() {
				a.save()
			}
			ui.Segmented(c, &a.mode, "Preview", "Edit", "Split").Label("View mode")
			ui.Box(c).Grow(1)
			ui.Checkbox(c, &a.wrap, "Wrap")
		})
		ui.Row(c).Padding(4, 12).Children(func() {
			label := a.path
			if label == "" {
				label = "Untitled"
			}
			if a.code.Buffer.Modified() {
				label += " •"
			}
			ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
		})
		switch a.mode {
		case 0:
			a.preview(c)
		case 1:
			a.edit(c)
		default:
			ui.SplitPercent(c, &a.ratio, func() { a.edit(c) }, func() { a.preview(c) }).Grow(1).FillWidth()
		}
		ui.Row(c).Padding(4, 12).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, a.status).Key("status").FontSize(12).TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			p := a.code.CursorPos()
			ui.Text(c, fmt.Sprintf("Ln %d, Col %d", p.Line+1, p.Col+1)).FontSize(12)
		})
	})
}

func Run(path string) error {
	a, err := New()
	if err != nil {
		return err
	}
	if path != "" {
		if err = a.load(path); err != nil {
			return err
		}
	}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "Plainmark — Go", Width: 1220, Height: 820, MinWidth: 880, MinHeight: 620, Content: ui.View(a.view)})
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Label: "File", Submenu: []*mygo.MenuItem{
				{Label: "Open file…", Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { a.chooseFile() }},
				{Label: "Save", Accelerator: "CmdOrCtrl+S", Click: func(*mygo.MenuItem, *mygo.Window) { a.save() }},
			}},
			{Role: mygo.RoleEditMenu}, {Role: mygo.RoleWindowMenu},
		}))
		a.win.OnClose(func(e *mygo.CloseEvent) {
			if a.busy {
				e.PreventDefault()
				return
			}
			if a.code.Buffer.Modified() {
				e.PreventDefault()
				a.start(func() error {
					if a.confirmDiscard() {
						a.update(func() { a.code.Buffer.MarkSaved(); a.busy = false; a.win.Close() })
					}
					return nil
				})
			}
		})
	})
	return mygo.App.Run()
}
