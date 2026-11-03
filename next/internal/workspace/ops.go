package workspace

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (w *Workspace) CreateNote(parent, name string, title *string) (Result, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	name, err := validName(name)
	if err != nil {
		return Result{}, err
	}
	if filepath.Ext(name) == "" {
		name += ".md"
	}
	if !IsMarkdown(name) {
		return Result{}, fail("notMarkdown", "A note must use .md or .mdx.")
	}
	full, err := w.child(parent, name)
	if err != nil {
		return Result{}, err
	}
	f, err := os.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return Result{}, err
	}
	head := strings.NewReplacer("-", " ", "_", " ").Replace(strings.TrimSuffix(name, filepath.Ext(name)))
	if title != nil {
		head = *title
	}
	if strings.TrimSpace(head) != "" {
		_, err = f.WriteString("# " + strings.TrimSpace(head) + "\n")
	}
	closeErr := f.Close()
	if err != nil {
		return Result{}, err
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	return Result{w.relative(full), "note"}, nil
}
func (w *Workspace) CreateFolder(parent, name string) (Result, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	full, err := w.child(parent, name)
	if err != nil {
		return Result{}, err
	}
	if err = os.Mkdir(full, 0755); err != nil {
		return Result{}, err
	}
	return Result{w.relative(full), "folder"}, nil
}
func (w *Workspace) Rename(path, name string) (Result, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	source, err := w.existing(path)
	if err != nil {
		return Result{}, err
	}
	name, err = validName(name)
	if err != nil {
		return Result{}, err
	}
	info, err := os.Stat(source)
	if err != nil {
		return Result{}, err
	}
	if !info.IsDir() && IsMarkdown(source) {
		if filepath.Ext(name) == "" {
			name += filepath.Ext(source)
		}
		if !IsMarkdown(name) {
			return Result{}, fail("notMarkdown", "A note must keep .md or .mdx.")
		}
	}
	return w.moveTo(source, filepath.Join(filepath.Dir(source), name), info.IsDir())
}
func (w *Workspace) Move(path, destination string) (Result, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	source, err := w.existing(path)
	if err != nil {
		return Result{}, err
	}
	dir, err := w.directory(destination)
	if err != nil {
		return Result{}, err
	}
	info, err := os.Stat(source)
	if err != nil {
		return Result{}, err
	}
	if info.IsDir() && within(source, dir) {
		return Result{}, fail("moveIntoDescendant", "A folder cannot move into itself or its descendant.")
	}
	return w.moveTo(source, filepath.Join(dir, filepath.Base(source)), info.IsDir())
}
func (w *Workspace) moveTo(source, target string, dir bool) (Result, error) {
	if _, err := os.Lstat(target); err == nil {
		return Result{}, fail("itemExists", "An item with this name already exists.")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Result{}, err
	}
	if err := os.Rename(source, target); err != nil {
		return Result{}, err
	}
	kind := "note"
	if dir {
		kind = "folder"
	}
	return Result{w.relative(target), kind}, nil
}

// Save uses a sibling temporary file, fsync, and rename. It checks both the
// timestamp and content hash before writing, and again just before replacement.
func (w *Workspace) Save(path, markdown string, expected Revision, force bool) (Note, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	target, err := w.existing(path)
	if err != nil {
		return Note{}, err
	}
	if !IsMarkdown(target) {
		return Note{}, fail("notMarkdown", "Only Markdown notes can be saved.")
	}
	before, err := w.read(path)
	if err != nil {
		return Note{}, err
	}
	if !force && before.Revision != expected {
		return Note{}, fail("conflict", "The file changed on disk. Your draft has been kept.")
	}
	info, err := os.Stat(target)
	if err != nil {
		return Note{}, err
	}
	f, err := os.CreateTemp(filepath.Dir(target), ".notes-save-*")
	if err != nil {
		return Note{}, err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = f.Chmod(info.Mode().Perm()); err != nil {
		return Note{}, err
	}
	if _, err = f.WriteString(markdown); err != nil {
		return Note{}, err
	}
	if err = f.Sync(); err != nil {
		return Note{}, err
	}
	if err = f.Close(); err != nil {
		return Note{}, err
	}
	if !force {
		latest, err := w.read(path)
		if err != nil {
			return Note{}, err
		}
		if latest.Revision != expected {
			return Note{}, fail("conflict", "The file changed during save. Your draft has been kept.")
		}
	}
	if err = os.Rename(f.Name(), target); err != nil {
		return Note{}, err
	}
	info, err = os.Stat(target)
	if err != nil {
		return Note{}, err
	}
	if info.ModTime().UnixMilli() <= expected.ModifiedMS {
		next := time.UnixMilli(expected.ModifiedMS + 1)
		if err = os.Chtimes(target, next, next); err != nil {
			return Note{}, err
		}
	}
	return w.read(path)
}

func (w *Workspace) TrashInfo(path string) (TrashInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	full, err := w.existing(path)
	if err != nil {
		return TrashInfo{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return TrashInfo{}, err
	}
	t := TrashInfo{Path: path, Name: filepath.Base(full), IsDirectory: info.IsDir()}
	if info.IsDir() {
		err = filepath.WalkDir(full, func(_ string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type().IsRegular() {
				t.NestedFiles++
			}
			return nil
		})
	}
	return t, err
}

// Trash delegates to the desktop's real trash operation; no permanent-delete
// fallback is provided. Tests inject a temporary-directory mover.
func (w *Workspace) Trash(path string, trash func(string) error) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	full, err := w.existing(path)
	if err != nil {
		return err
	}
	return trash(full)
}
