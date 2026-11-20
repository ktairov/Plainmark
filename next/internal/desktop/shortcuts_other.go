//go:build !darwin

package desktop

func (a *App) installModeShortcuts() func() { return nil }
