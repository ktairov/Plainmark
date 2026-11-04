package settings

import (
	"bytes"
	"sync"
)

// Writer serializes state writes off the UI thread and coalesces queued frames.
// Close flushes the most recent accepted state before returning.
type Writer struct {
	path      string
	requests  chan []byte
	done      chan struct{}
	Errors    <-chan error
	mu        sync.Mutex
	closed    bool
	lastError error
}

func NewWriter(path string) *Writer {
	errors := make(chan error, 1)
	w := &Writer{path: path, requests: make(chan []byte, 1), done: make(chan struct{}), Errors: errors}
	go func() {
		defer close(w.done)
		defer close(errors)
		for data := range w.requests {
		drain:
			for {
				select {
				case next, ok := <-w.requests:
					if !ok {
						break drain
					}
					data = next
				default:
					break drain
				}
			}
			err := Save(w.path, data)
			w.mu.Lock()
			w.lastError = err
			w.mu.Unlock()
			if err != nil {
				select {
				case errors <- err:
				default:
				}
			}
		}
	}()
	return w
}
func (w *Writer) Queue(data []byte) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return false
	}
	data = bytes.Clone(data)
	select {
	case w.requests <- data:
	default:
		select {
		case <-w.requests:
		default:
		}
		w.requests <- data
	}
	return true
}
func (w *Writer) Close() error {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		close(w.requests)
	}
	w.mu.Unlock()
	<-w.done
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastError
}
