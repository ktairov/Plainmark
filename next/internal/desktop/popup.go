package desktop

import (
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"plainmark/internal/icons"
)

type popupItem struct {
	label, icon, key                             string
	disabled, danger, checked, separator, choice bool
	action                                       func()
}
type popupState struct {
	id         string
	x, y       float32
	items      []popupItem
	seen       bool
	inDialog   bool
	focusFirst bool
}

func (a *App) closePopup() {
	if a.popup != nil {
		a.popupReturnID = a.popup.id
		a.popup = nil
	}
}
func (a *App) bindPopup(c *ui.Context, anchor ui.Element, id string, items []popupItem, context bool) bool {
	if a.popupReturnID == id {
		anchor.Focus()
		a.popupReturnID = ""
	}
	opened := false
	if context {
		opened = anchor.RightClicked() || anchor.Shortcut(ui.Shift, ui.KeyF10) || anchor.Clicked() && anchor.ClickModifiers()&ui.Ctrl != 0
	} else {
		opened = anchor.Clicked() || anchor.Shortcut(0, ui.KeyDown)
	}
	if opened && !anchor.IsDisabled() {
		if !context && a.popup != nil && a.popup.id == id {
			a.closePopup()
			return true
		}
		r := anchor.Bounds()
		x, y := r.X, r.Y+r.H+5
		if context {
			px, py, over := anchor.PointerPosition()
			if over {
				x, y = r.X+px, r.Y+py
			}
		}
		a.popup = &popupState{id: id, x: x, y: y, items: items, seen: true, focusFirst: true, inDialog: a.settingsOpen || a.form != nil || a.crypto != nil || a.pending != nil || a.deleting != nil}
	}
	if a.popup != nil && a.popup.id == id {
		a.popup.items = items
		a.popup.seen = true
	}
	return opened
}

func (a *App) drawPopup(c *ui.Context) {
	p := a.popup
	if p == nil {
		return
	}
	if !p.seen || a.busy || p.inDialog != (a.settingsOpen || a.form != nil || a.crypto != nil || a.pending != nil || a.deleting != nil) {
		a.closePopup()
		return
	}
	w, h := c.Size()
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Modal()
		if back.Clicked() || back.RightClicked() || back.OverlayShortcut(0, ui.KeyEscape) {
			a.closePopup()
			return
		}
		back.Children(func() { a.popupPanel(c, p, w, h) })
	})
}

func (a *App) popupPanel(c *ui.Context, p *popupState, w, h float32) {
	items, x, y := p.items, p.x, p.y
	s := a.visual(c)
	width, height := float32(170), float32(12)
	focus := -1
	if p.focusFirst {
		for index, item := range items {
			if !item.disabled && !item.separator && (focus < 0 || item.checked) {
				focus = index
				if item.checked {
					break
				}
			}
		}
	}
	for _, item := range items {
		if item.separator {
			height += 11
			continue
		}
		height += 32
		width = max(width, float32(utf8.RuneCountInString(item.label))*7.6+64)
	}
	width = min(width, min(float32(400), w-16))
	height = min(height, h-16)
	x = max(float32(8), min(x, w-width-8))
	y = max(float32(8), min(y, h-height-8))
	c.Key("popup:" + p.id)
	ui.Column(c).Absolute().Left(x).Top(y).Width(width).Height(height).Padding(5).Radius(8).Background(s.control).Border(1, s.border).Shadow(0, 8, 28, 0, ui.RGBA(0, 0, 0, .2)).Role(ui.RoleMenu).Children(func() {
		ui.Scroll(c).Fill().Children(func() {
			ui.Column(c).FillWidth().FocusGroup(ui.Vertical).Children(func() {
				for index, item := range items {
					if item.separator {
						ui.Box(c).Height(1).FillWidth().Margin(5, 0).Background(s.softBorder)
						continue
					}
					button := ui.ButtonBase(c).Role(ui.RoleMenuItem).Label(item.label).Height(32).FillWidth().Padding(7, 9).Gap(9).Justify(ui.Start).Radius(5).TextColor(s.text).FontSize(14).Disabled(item.disabled)
					if item.key != "" {
						button = button.Key(item.key)
					}
					if item.choice {
						button.Role(ui.RoleMenuItemRadio).Checked(item.checked)
					}
					if item.danger {
						button.TextColor(s.danger)
					}
					if button.Hovered() || button.FocusVisible() {
						button.Background(s.page)
					}
					if p.focusFirst && index == focus {
						button.Focus()
						button.ScrollIntoView()
						p.focusFirst = false
					}
					button.Children(func() {
						if item.icon != "" {
							ui.Icon(c, icons.Get(item.icon)).Size(15, 15)
						}
						ui.Text(c, item.label).SingleLine().MinWidth(0).Grow(1).Ellipsis("…")
						if item.checked {
							ui.Icon(c, icons.Get("check")).Size(14, 14).TextColor(s.accent)
						}
					})
					if button.Clicked() && !item.disabled {
						a.closePopup()
						if item.action != nil {
							item.action()
						}
					}

				}
			})
		})
	})
}

type selectOption struct{ id, label, key string }

func (a *App) selectControl(c *ui.Context, key, label, value string, options []selectOption, onChange func(string)) ui.Element {
	s := a.visual(c)
	display := value
	items := make([]popupItem, 0, len(options))
	for _, option := range options {
		if option.id == value {
			display = option.label
		}
		items = append(items, popupItem{label: option.label, key: option.key, choice: true, checked: option.id == value, action: func() { onChange(option.id) }})
	}
	button := ui.ButtonBase(c).Key(key).Role(ui.RolePopUpButton).Label(a.t(label)).Height(34).Padding(7, 9).Gap(8).Radius(6).Border(1, s.border).Background(s.control).TextColor(s.text).FontSize(14)
	if button.Hovered() {
		button.Background(s.controlHover)
	}
	button.Children(func() {
		ui.Text(c, display).SingleLine().MinWidth(0).Grow(1).Ellipsis("…")
		ui.Icon(c, icons.Get("chevron-down")).Size(13, 13).TextColor(s.muted)
	})
	a.bindPopup(c, button, key, items, false)
	return button
}
