package store

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/internal/types"
)

type requestEntry struct {
	req      types.UIRequest
	done     chan struct{}
	doneOnce sync.Once
}

// Store is an in-memory store for UIRequests (E1).
// It also provides an event-driven wait mechanism (F2) via per-request done channels.
type Store struct {
	mu       sync.RWMutex
	requests map[string]*requestEntry
}

func New() *Store {
	return &Store{
		requests: make(map[string]*requestEntry),
	}
}

type CreateParams struct {
	Type      types.WidgetType
	SessionID string
	Input     any
	TimeoutS  int
}

func (s *Store) Create(_ context.Context, p CreateParams) (types.UIRequest, error) {
	if p.Type == "" {
		return types.UIRequest{}, errors.New("type is required")
	}
	if p.Input == nil {
		return types.UIRequest{}, errors.New("input is required")
	}
	if p.TimeoutS <= 0 {
		p.TimeoutS = 300
	}
	if p.SessionID == "" {
		// Compatibility: React/old server expect a string sessionId field.
		// We intentionally ignore sessions (G=no-session), but keep a non-empty value.
		p.SessionID = "global"
	}

	now := time.Now().UTC()
	id := uuid.NewString()
	req := types.UIRequest{
		ID:        id,
		Type:      p.Type,
		SessionID: p.SessionID,
		Input:     p.Input,
		Status:    types.StatusPending,
		CreatedAt: now.Format(time.RFC3339Nano),
		ExpiresAt: now.Add(time.Duration(p.TimeoutS) * time.Second).Format(time.RFC3339Nano),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests[id] = &requestEntry{
		req:  req,
		done: make(chan struct{}),
	}

	return req, nil
}

func (s *Store) Get(_ context.Context, id string) (types.UIRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.requests[id]
	if !ok {
		return types.UIRequest{}, ErrNotFound
	}
	return e.req, nil
}

func (s *Store) Pending(_ context.Context) []types.UIRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]types.UIRequest, 0, len(s.requests))
	for _, e := range s.requests {
		if e.req.Status == types.StatusPending {
			out = append(out, e.req)
		}
	}
	return out
}

func (s *Store) Complete(_ context.Context, id string, output any) (types.UIRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.requests[id]
	if !ok {
		return types.UIRequest{}, ErrNotFound
	}
	if e.req.Status != types.StatusPending {
		return types.UIRequest{}, ErrAlreadyCompleted
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	e.req.Output = output
	e.req.Status = types.StatusCompleted
	e.req.CompletedAt = &now

	e.doneOnce.Do(func() { close(e.done) })

	return e.req, nil
}

func (s *Store) Wait(ctx context.Context, id string) (types.UIRequest, error) {
	s.mu.RLock()
	e, ok := s.requests[id]
	s.mu.RUnlock()

	if !ok {
		return types.UIRequest{}, ErrNotFound
	}
	if e.req.Status == types.StatusCompleted {
		return e.req, nil
	}

	select {
	case <-e.done:
		// Return latest value (may have been updated)
		return s.Get(ctx, id)
	case <-ctx.Done():
		return types.UIRequest{}, ErrWaitTimeout
	}
}
