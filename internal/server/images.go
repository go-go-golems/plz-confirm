package server

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// StoredImage is a server-side stored blob (currently: file on disk + in-memory index).
// We keep it intentionally small: just enough metadata to serve the file back.
type StoredImage struct {
	ID        string
	Path      string
	MimeType  string
	Size      int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

type ImageStore struct {
	mu sync.RWMutex

	dir            string
	maxUploadBytes int64
	images         map[string]StoredImage
}

type ImageStoreOptions struct {
	Dir            string
	MaxUploadBytes int64
}

func NewImageStore(opts ImageStoreOptions) (*ImageStore, error) {
	dir := opts.Dir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "plz-confirm-images")
	}
	if opts.MaxUploadBytes <= 0 {
		opts.MaxUploadBytes = 50 << 20 // 50MB default
	}

	// Ensure directory exists (no-op if already present).
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Wrap(err, "create image store dir")
	}

	return &ImageStore{
		dir:            dir,
		maxUploadBytes: opts.MaxUploadBytes,
		images:         make(map[string]StoredImage),
	}, nil
}

func (s *ImageStore) MaxUploadBytes() int64 {
	return s.maxUploadBytes
}

func (s *ImageStore) Put(
	_ context.Context,
	r io.Reader,
	mimeType string,
	expiresAt time.Time,
) (StoredImage, error) {
	now := time.Now().UTC()
	id := uuid.NewString()
	dstPath := filepath.Join(s.dir, id)

	f, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return StoredImage{}, errors.Wrap(err, "open destination file")
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		_ = os.Remove(dstPath)
		return StoredImage{}, errors.Wrap(err, "write destination file")
	}

	img := StoredImage{
		ID:        id,
		Path:      dstPath,
		MimeType:  mimeType,
		Size:      n,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	s.mu.Lock()
	s.images[id] = img
	s.mu.Unlock()

	return img, nil
}

func (s *ImageStore) Get(_ context.Context, id string) (StoredImage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	img, ok := s.images[id]
	return img, ok
}

func (s *ImageStore) Delete(_ context.Context, id string) {
	s.mu.Lock()
	img, ok := s.images[id]
	if ok {
		delete(s.images, id)
	}
	s.mu.Unlock()

	if ok {
		_ = os.Remove(img.Path)
	}
}

func (s *ImageStore) Cleanup(_ context.Context, now time.Time) (deleted int) {
	// Snapshot keys to avoid holding lock while deleting files.
	toDelete := make([]string, 0)

	s.mu.RLock()
	for id, img := range s.images {
		if !img.ExpiresAt.IsZero() && now.After(img.ExpiresAt) {
			toDelete = append(toDelete, id)
		}
	}
	s.mu.RUnlock()

	for _, id := range toDelete {
		s.Delete(context.Background(), id)
		deleted++
	}

	return deleted
}
