package store

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
)

type requestEntry struct {
	req      *v1.UIRequest
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

// Create creates a new UIRequest from a protobuf UIRequest.
// The request should have Input oneof populated, Type set, and SessionId set.
// ID, Status, CreatedAt, and ExpiresAt will be set automatically.
func (s *Store) Create(_ context.Context, req *v1.UIRequest) (*v1.UIRequest, error) {
	if req.Type == v1.WidgetType_widget_type_unspecified {
		return nil, errors.New("type is required")
	}
	if req.Input == nil {
		return nil, errors.New("input is required")
	}
	if req.SessionId == "" {
		// Compatibility: clients expect a string sessionId field.
		req.SessionId = "global"
	}

	now := time.Now().UTC()
	id := uuid.NewString()

	// Clone the request and set required fields
	reqCopy := &v1.UIRequest{
		Id:        id,
		Type:      req.Type,
		SessionId: req.SessionId,
		Input:     req.Input, // Copy the oneof field
		Metadata:  req.Metadata,
		Status:    v1.RequestStatus_pending,
		CreatedAt: now.Format(time.RFC3339Nano),
		ExpiresAt: now.Format(time.RFC3339Nano), // Will be set below
	}

	// Parse expiresAt if provided, otherwise use default timeout
	var timeoutS int64 = 300
	if req.ExpiresAt != "" {
		if expTime, err := time.Parse(time.RFC3339Nano, req.ExpiresAt); err == nil {
			timeoutS = int64(time.Until(expTime).Seconds())
		}
	}
	if timeoutS <= 0 {
		timeoutS = 300
	}
	reqCopy.ExpiresAt = now.Add(time.Duration(timeoutS) * time.Second).Format(time.RFC3339Nano)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests[id] = &requestEntry{
		req:  reqCopy,
		done: make(chan struct{}),
	}

	return reqCopy, nil
}

func (s *Store) Get(_ context.Context, id string) (*v1.UIRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.requests[id]
	if !ok {
		return nil, ErrNotFound
	}
	return e.req, nil
}

func (s *Store) Pending(_ context.Context) []*v1.UIRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*v1.UIRequest, 0, len(s.requests))
	for _, e := range s.requests {
		if e.req.Status == v1.RequestStatus_pending {
			out = append(out, e.req)
		}
	}
	return out
}

func (s *Store) PendingForSession(_ context.Context, sessionID string) []*v1.UIRequest {
	if sessionID == "" {
		sessionID = "global"
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*v1.UIRequest, 0, len(s.requests))
	for _, e := range s.requests {
		if e.req.Status == v1.RequestStatus_pending && e.req.SessionId == sessionID {
			out = append(out, e.req)
		}
	}
	return out
}

func (s *Store) Expire(now time.Time) []*v1.UIRequest {
	now = now.UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	var expired []*v1.UIRequest
	for _, e := range s.requests {
		if e.req.Status != v1.RequestStatus_pending {
			continue
		}
		expAt, err := time.Parse(time.RFC3339Nano, e.req.ExpiresAt)
		if err != nil {
			continue
		}
		if now.Before(expAt) {
			continue
		}

		e.req.Status = v1.RequestStatus_timeout
		completedAt := now.Format(time.RFC3339Nano)
		e.req.CompletedAt = &completedAt
		errMsg := "request timed out"
		e.req.Error = &errMsg

		e.doneOnce.Do(func() { close(e.done) })
		expired = append(expired, e.req)
	}

	return expired
}

func (s *Store) Complete(_ context.Context, id string, output *v1.UIRequest) (*v1.UIRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.requests[id]
	if !ok {
		return nil, ErrNotFound
	}
	if e.req.Status != v1.RequestStatus_pending {
		return nil, ErrAlreadyCompleted
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	e.req.Output = output.Output // Copy the output oneof field
	e.req.Status = v1.RequestStatus_completed
	completedAt := now
	e.req.CompletedAt = &completedAt

	e.doneOnce.Do(func() { close(e.done) })

	return e.req, nil
}

func (s *Store) Wait(ctx context.Context, id string) (*v1.UIRequest, error) {
	s.mu.RLock()
	e, ok := s.requests[id]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}
	if e.req.Status == v1.RequestStatus_completed || e.req.Status == v1.RequestStatus_timeout || e.req.Status == v1.RequestStatus_error {
		return e.req, nil
	}

	select {
	case <-e.done:
		// Return latest value (may have been updated)
		return s.Get(ctx, id)
	case <-ctx.Done():
		return nil, ErrWaitTimeout
	}
}
