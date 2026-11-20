//go:build e2e

package main

import (
	"encoding/json"
	"os"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/plugins/automation"
)

// The automation plugin injects content events, but cannot move native windows.
// This stdin hook exists only in e2e builds, with automation explicitly enabled.
func init() {
	if os.Getenv("MYGO_AUTOMATION") == "" {
		return
	}
	go func() {
		decoder := json.NewDecoder(os.Stdin)
		for {
			var command struct {
				Bounds    *mygo.Rectangle
				Maximized *bool
			}
			if decoder.Decode(&command) != nil {
				return
			}
			mygo.RunOnMain(func() {
				windows := mygo.Windows()
				if len(windows) == 0 {
					return
				}
				window := windows[0]
				if command.Maximized != nil {
					if *command.Maximized {
						window.Maximize()
					} else {
						window.Unmaximize()
					}
				}
				if command.Bounds != nil {
					window.SetBounds(*command.Bounds)
				}
			})
		}
	}()
}
