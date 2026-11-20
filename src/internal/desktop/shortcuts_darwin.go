package desktop

import "github.com/ebitengine/purego/objc"

// Option plus a digit can type a character on macOS. Handle the mode
// shortcuts before AppKit passes those key presses to the text input system.
func (a *App) installModeShortcuts() func() {
	window := objc.ID(a.win.NativeHandle())
	events := objc.ID(objc.GetClass("NSEvent"))
	eventWindow := objc.RegisterName("window")
	modifiers := objc.RegisterName("modifierFlags")
	keyCode := objc.RegisterName("keyCode")
	attachedSheet := objc.RegisterName("attachedSheet")
	const (
		keyDownMask = 1 << 10
		option      = 1 << 19
		chordMask   = 1<<17 | 1<<18 | option | 1<<20 // Shift, Control, Option, Command.
		digitOne    = 0x12                           // ANSI key codes for 1, 2, 3 are consecutive.
	)
	handler := objc.NewBlock(func(_ objc.Block, event objc.ID) objc.ID {
		if event.Send(eventWindow) != window || window.Send(attachedSheet) != 0 ||
			event.Send(modifiers)&chordMask != option || a.menuBlocked() {
			return event
		}
		digit := int(uint16(event.Send(keyCode))) - digitOne
		d := a.current()
		if digit < 0 || digit >= len(viewModes) || d == nil {
			return event
		}
		a.update(func() {
			a.closePopup()
			d.Tab.Mode = viewModes[digit].mode
		})
		return 0
	})
	monitor := events.Send(objc.RegisterName("addLocalMonitorForEventsMatchingMask:handler:"), uintptr(keyDownMask), handler)
	handler.Release()
	return func() {
		if monitor != 0 {
			events.Send(objc.RegisterName("removeMonitor:"), monitor)
		}
	}
}
