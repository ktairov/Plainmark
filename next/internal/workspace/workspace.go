// Package workspace owns the file-based note store, independent of the GUI.
package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string           { return e.Message }
func fail(code, msg string) error        { return &Error{code, msg} }
func IsCode(err error, code string) bool { var e *Error; return errors.As(err, &e) && e.Code == code }

type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// Revision includes content identity so a same-timestamp external write is
// detected too. The millisecond timestamp is retained for old behavior/UI.
type Revision struct {
	ModifiedMS int64  `json:"modified_ms"`
	Hash       string `json:"hash"`
}
type Note struct {
	Path     string    `json:"path"`
	Title    string    `json:"title"`
	Markdown string    `json:"markdown"`
	Headings []Heading `json:"headings"`
	Revision Revision  `json:"revision"`
	Size     int64     `json:"size"`
}
type Entry struct {
	Path       string
	Title      string
	Folder     string
	Headings   []Heading
	ModifiedMS int64
	Size       int64
}
type Folder struct {
	Path string
	Name string
}
type Snapshot struct {
	Notes   []Entry
	Folders []Folder
}
type Result struct {
	Path string
	Kind string
}
type TrashInfo struct {
	Path        string
	Name        string
	NestedFiles int
	IsDirectory bool
}

type Workspace struct {
	root string
	mu   sync.Mutex
}

func Open(root string) (*Workspace, error) {
	full, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	full, err = filepath.EvalSymlinks(full)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fail("notAFolder", "The workspace must be a folder.")
	}
	return &Workspace{root: full}, nil
}
func (w *Workspace) Root() string { return w.root }
func IsMarkdown(path string) bool {
	e := strings.ToLower(filepath.Ext(path))
	return e == ".md" || e == ".mdx"
}
func Excluded(name string) bool {
	return strings.HasPrefix(name, ".") || name == "node_modules" || name == "target" || name == "dist"
}

func ValidateRelative(path string, empty bool) error {
	if path == "" && empty {
		return nil
	}
	if strings.TrimSpace(path) == "" || filepath.IsAbs(path) || strings.ContainsAny(path, "\\\x00") || filepath.ToSlash(filepath.Clean(path)) != path {
		return fail("invalidRelativePath", "Path must be a normalized relative path inside the workspace.")
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." || part == "." || part == "" {
			return fail("invalidRelativePath", "Path must remain inside the workspace.")
		}
		if Excluded(part) {
			return fail("excludedPath", "Hidden and excluded paths cannot be modified.")
		}
	}
	return nil
}
func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", fail("invalidName", "Name cannot be empty, '.' or '..'.")
	}
	if strings.ContainsAny(name, "/\\\x00") {
		return "", fail("nameHasSeparator", "Name cannot contain path separators.")
	}
	if Excluded(name) {
		return "", fail("excludedName", "Hidden and excluded names are not allowed.")
	}
	return name, nil
}
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
func (w *Workspace) existing(path string) (string, error) {
	if err := ValidateRelative(path, false); err != nil {
		return "", err
	}
	full, err := filepath.EvalSymlinks(filepath.Join(w.root, filepath.FromSlash(path)))
	if err != nil {
		return "", err
	}
	if !within(w.root, full) {
		return "", fail("pathOutsideWorkspace", "Path is outside the workspace folder.")
	}
	return full, nil
}
func (w *Workspace) Absolute(path string) (string, error) { return w.existing(path) }
func (w *Workspace) directory(path string) (string, error) {
	if path == "" {
		return w.root, nil
	}
	full, err := w.existing(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(full)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fail("notAFolder", "Destination is not a folder.")
	}
	return full, nil
}
func (w *Workspace) child(parent, name string) (string, error) {
	dir, err := w.directory(parent)
	if err != nil {
		return "", err
	}
	name, err = validName(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}
func (w *Workspace) relative(path string) string {
	rel, _ := filepath.Rel(w.root, path)
	return filepath.ToSlash(rel)
}
func revision(data []byte, info fs.FileInfo) Revision {
	h := sha256.Sum256(data)
	return Revision{info.ModTime().UnixMilli(), hex.EncodeToString(h[:])}
}
func (w *Workspace) read(path string) (Note, error) {
	full, err := w.existing(path)
	if err != nil {
		return Note{}, err
	}
	if !IsMarkdown(full) {
		return Note{}, fail("notMarkdown", "Only .md and .mdx files can be read.")
	}
	f, err := os.Open(full)
	if err != nil {
		return Note{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return Note{}, err
	}
	if !utf8.Valid(data) {
		return Note{}, fail("invalidText", "The note is not valid UTF-8.")
	}
	info, err := f.Stat()
	if err != nil {
		return Note{}, err
	}
	src := string(data)
	heads := ParseHeadings(src)
	return Note{Path: path, Title: TitleFor(path, src, heads), Markdown: src, Headings: heads, Revision: revision(data, info), Size: int64(len(data))}, nil
}
func (w *Workspace) Read(path string) (Note, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.read(path)
}
func (w *Workspace) Scan() (Snapshot, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := Snapshot{Notes: []Entry{}, Folders: []Folder{}}
	err := filepath.WalkDir(w.root, func(full string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if full == w.root {
			return nil
		}
		if Excluded(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		path := w.relative(full)
		if d.IsDir() {
			s.Folders = append(s.Folders, Folder{path, d.Name()})
			return nil
		}
		if !d.Type().IsRegular() || !IsMarkdown(path) {
			return nil
		}
		n, err := w.read(path)
		if err != nil {
			return fmt.Errorf("scan %s: %w", path, err)
		}
		folder := filepath.ToSlash(filepath.Dir(path))
		if folder == "." {
			folder = ""
		}
		s.Notes = append(s.Notes, Entry{path, n.Title, folder, n.Headings, n.Revision.ModifiedMS, n.Size})
		return nil
	})
	return s, err
}
