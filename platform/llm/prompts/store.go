package prompts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
)

// SHA256Hex is exported so the gateway's cache-key helper can reuse the same
// hash function as the seal verifier.
func SHA256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Store loads sealed prompts from a filesystem (typically embed.FS).
//
// Layout: <root>/<name>/<semver>.md
type Store struct {
	fsys fs.FS
	root string

	mu     sync.RWMutex
	loaded map[string]*Prompt // key = "<name>/<version>"
}

// NewStore constructs a Store rooted at `root` inside fsys (use "." for the
// fsys root).
func NewStore(fsys fs.FS, root string) *Store {
	return &Store{fsys: fsys, root: root, loaded: make(map[string]*Prompt)}
}

// Load returns a parsed + sealed prompt. Repeated loads of the same
// (name, version) are served from an in-memory cache.
func (s *Store) Load(name, version string) (*Prompt, error) {
	key := name + "/" + version
	s.mu.RLock()
	if p, ok := s.loaded[key]; ok {
		s.mu.RUnlock()
		return p, nil
	}
	s.mu.RUnlock()

	rel := filepath.ToSlash(filepath.Join(s.root, name, version+".md"))
	raw, err := fs.ReadFile(s.fsys, rel)
	if err != nil {
		return nil, fmt.Errorf("prompts: read %s: %w", rel, err)
	}
	p, err := Parse(name, version, raw)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.loaded[key] = p
	s.mu.Unlock()
	return p, nil
}
