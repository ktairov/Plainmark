package settings

import (
	"bytes"
	"sync"
	"time"
)

// Writer owns the latest requested and last successfully written snapshots.
// Failed writes retry without requiring another frame or a change of settings.
// Close makes one final attempt to flush the latest snapshot, then stops.
type Writer struct {
	path      string
	wake      chan struct{}
	stop      chan struct{}
	done      chan struct{}
	Results   <-chan error // completed attempts; nil confirms a successful write
	mu        sync.Mutex
	closed    bool
	latest    []byte
	saved     []byte
	hasLatest bool
	hasSaved  bool
	lastError error
}

func NewWriter(path string) *Writer {
	results := make(chan error, 1)
	w := &Writer{path: path, wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{}), Results: results}
	go w.run(results)
	return w
}

func (w *Writer) pending() ([]byte, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.hasLatest || w.hasSaved && bytes.Equal(w.latest, w.saved) {
		w.lastError = nil
		return nil, false
	}
	return w.latest, true // snapshots are immutable after Queue clones them
}

func (w *Writer) write(data []byte, results chan error) error {
	err := Save(w.path, data)
	w.mu.Lock()
	w.lastError = err
	if err == nil {
		w.saved, w.hasSaved = data, true
	}
	w.mu.Unlock()
	// Never block the disk worker on UI notifications; keep the latest result.
	select {
	case results <- err:
	default:
		select {
		case <-results:
		default:
		}
		results <- err
	}
	return err
}

func (w *Writer) run(results chan error) {
	defer close(w.done)
	defer close(results)
	var timer *time.Timer
	var retry <-chan time.Time
	delay := 100 * time.Millisecond
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-w.stop:
			if data, pending := w.pending(); pending {
				w.write(data, results)
			}
			return
		case <-w.wake:
			if timer != nil {
				timer.Stop()
			}
			retry = nil
			delay = 100 * time.Millisecond
		case <-retry:
			retry = nil
		}
		for {
			data, pending := w.pending()
			if !pending {
				break
			}
			if w.write(data, results) != nil {
				if timer == nil {
					timer = time.NewTimer(delay)
				} else {
					timer.Reset(delay)
				}
				retry = timer.C
				delay = min(delay*2, 5*time.Second)
				break
			}
			delay = 100 * time.Millisecond
		}
	}
}

func (w *Writer) Queue(data []byte) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return false
	}
	if w.hasLatest && bytes.Equal(w.latest, data) {
		return true
	}
	w.latest, w.hasLatest = bytes.Clone(data), true
	select {
	case w.wake <- struct{}{}:
	default:
	}
	return true
}

func (w *Writer) Close() error {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		close(w.stop)
	}
	w.mu.Unlock()
	<-w.done
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastError
}
