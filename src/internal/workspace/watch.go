package workspace

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	Changes <-chan []string
	Errors  <-chan error
	cancel  context.CancelFunc
	done    chan struct{}
}

func (w *Workspace) Watch(ctx context.Context) (*Watcher, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	native, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	addTree := func(path string) error {
		return filepath.WalkDir(path, func(full string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if full != w.root && Excluded(d.Name()) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return native.Add(full)
			}
			return nil
		})
	}
	if err = addTree(w.root); err != nil {
		native.Close()
		return nil, err
	}
	life, cancel := context.WithCancel(ctx)
	changes, errs := make(chan []string), make(chan error, 1)
	watch := &Watcher{changes, errs, cancel, make(chan struct{})}
	go func() {
		defer close(watch.done)
		defer close(changes)
		defer close(errs)
		defer native.Close()
		pending := map[string]bool{}
		var timer *time.Timer
		var tick <-chan time.Time
		var ceiling *time.Timer
		var maxTick <-chan time.Time
		deadlineReached := false
		defer func() {
			if timer != nil {
				timer.Stop()
			}
			if ceiling != nil {
				ceiling.Stop()
			}
		}()
		ready := false
		report := func(err error) {
			select {
			case errs <- err:
			default:
			}
		}
		for {
			var out chan []string
			var batch []string
			if ready && len(pending) > 0 {
				out = changes
				for p := range pending {
					batch = append(batch, p)
				}
				sort.Strings(batch)
			}
			select {
			case <-life.Done():
				return
			case err, ok := <-native.Errors:
				if !ok {
					return
				}
				report(err)
			case e, ok := <-native.Events:
				if !ok {
					return
				}
				if e.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
					continue
				}
				rel, err := filepath.Rel(w.root, e.Name)
				if err != nil {
					report(err)
					continue
				}
				rel = filepath.ToSlash(rel)
				if rel != "." && ValidateRelative(rel, false) != nil {
					continue
				}
				if e.Has(fsnotify.Create) {
					if info, err := os.Lstat(e.Name); err == nil && info.IsDir() {
						if err = addTree(e.Name); err != nil {
							report(err)
						}
					}
				}
				pending[rel] = true
				if ceiling == nil {
					ceiling = time.NewTimer(500 * time.Millisecond)
					maxTick = ceiling.C
				}
				if timer == nil {
					timer = time.NewTimer(100 * time.Millisecond)
				} else {
					timer.Reset(100 * time.Millisecond)
				}
				tick = timer.C
				if !deadlineReached {
					ready = false
				}
			case <-tick:
				tick = nil
				ready = true
			case <-maxTick:
				maxTick = nil
				deadlineReached = true
				ready = true
			case out <- batch:
				pending = map[string]bool{}
				ready = false
				deadlineReached = false
				if ceiling != nil {
					ceiling.Stop()
					ceiling = nil
					maxTick = nil
				}
				if timer != nil {
					timer.Stop()
					tick = nil
				}
			}
		}
	}()
	return watch, nil
}

// Close waits until callbacks/channels can no longer deliver an old workspace.
func (w *Watcher) Close() { w.cancel(); <-w.done }
