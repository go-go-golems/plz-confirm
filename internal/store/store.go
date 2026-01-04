package store

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/types/known/structpb"
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

		autoComment := "AUTO_TIMEOUT"
		setDefaultOutputFor(e.req, now, &autoComment)
		e.req.Status = v1.RequestStatus_completed
		completedAt := now.Format(time.RFC3339Nano)
		e.req.CompletedAt = &completedAt
		e.req.Error = nil

		e.doneOnce.Do(func() { close(e.done) })
		expired = append(expired, e.req)
	}

	return expired
}

func setDefaultOutputFor(req *v1.UIRequest, now time.Time, comment *string) {
	switch req.Type {
	case v1.WidgetType_widget_type_unspecified:
		req.Output = nil
		return
	case v1.WidgetType_confirm:
		req.Output = &v1.UIRequest_ConfirmOutput{
			ConfirmOutput: &v1.ConfirmOutput{
				Approved:  false,
				Timestamp: now.Format(time.RFC3339Nano),
				Comment:   comment,
			},
		}
		return
	case v1.WidgetType_select:
		in := req.GetSelectInput()
		multi := in != nil && in.Multi != nil && *in.Multi
		if multi {
			req.Output = &v1.UIRequest_SelectOutput{
				SelectOutput: &v1.SelectOutput{
					Selected: &v1.SelectOutput_SelectedMulti{
						SelectedMulti: &v1.SelectOutputMulti{Values: []string{}},
					},
					Comment: comment,
				},
			}
			return
		}
		first := ""
		if in != nil && len(in.Options) > 0 {
			first = in.Options[0]
		}
		req.Output = &v1.UIRequest_SelectOutput{
			SelectOutput: &v1.SelectOutput{
				Selected: &v1.SelectOutput_SelectedSingle{SelectedSingle: first},
				Comment:  comment,
			},
		}
		return
	case v1.WidgetType_form:
		st, _ := structpb.NewStruct(map[string]any{})
		req.Output = &v1.UIRequest_FormOutput{
			FormOutput: &v1.FormOutput{
				Data:    st,
				Comment: comment,
			},
		}
		return
	case v1.WidgetType_upload:
		req.Output = &v1.UIRequest_UploadOutput{
			UploadOutput: &v1.UploadOutput{
				Files:   []*v1.UploadedFile{},
				Comment: comment,
			},
		}
		return
	case v1.WidgetType_table:
		in := req.GetTableInput()
		multi := in != nil && in.MultiSelect != nil && *in.MultiSelect
		if multi {
			req.Output = &v1.UIRequest_TableOutput{
				TableOutput: &v1.TableOutput{
					Selected: &v1.TableOutput_SelectedMulti{
						SelectedMulti: &v1.TableOutputMulti{Values: []*structpb.Struct{}},
					},
					Comment: comment,
				},
			}
			return
		}
		st, _ := structpb.NewStruct(map[string]any{})
		req.Output = &v1.UIRequest_TableOutput{
			TableOutput: &v1.TableOutput{
				Selected: &v1.TableOutput_SelectedSingle{SelectedSingle: st},
				Comment:  comment,
			},
		}
		return
	case v1.WidgetType_image:
		in := req.GetImageInput()
		isConfirm := in != nil && in.Mode == "confirm"
		hasOptions := in != nil && len(in.Options) > 0
		multi := in != nil && in.Multi != nil && *in.Multi
		if isConfirm {
			req.Output = &v1.UIRequest_ImageOutput{
				ImageOutput: &v1.ImageOutput{
					Selected:  &v1.ImageOutput_SelectedBool{SelectedBool: false},
					Timestamp: now.Format(time.RFC3339Nano),
					Comment:   comment,
				},
			}
			return
		}
		if hasOptions {
			if multi {
				req.Output = &v1.UIRequest_ImageOutput{
					ImageOutput: &v1.ImageOutput{
						Selected: &v1.ImageOutput_SelectedStrings{
							SelectedStrings: &v1.ImageOutputStrings{Values: []string{}},
						},
						Timestamp: now.Format(time.RFC3339Nano),
						Comment:   comment,
					},
				}
				return
			}
			first := ""
			if len(in.Options) > 0 {
				first = in.Options[0]
			}
			req.Output = &v1.UIRequest_ImageOutput{
				ImageOutput: &v1.ImageOutput{
					Selected:  &v1.ImageOutput_SelectedString{SelectedString: first},
					Timestamp: now.Format(time.RFC3339Nano),
					Comment:   comment,
				},
			}
			return
		}
		if multi {
			req.Output = &v1.UIRequest_ImageOutput{
				ImageOutput: &v1.ImageOutput{
					Selected: &v1.ImageOutput_SelectedNumbers{
						SelectedNumbers: &v1.ImageOutputNumbers{Values: []int64{}},
					},
					Timestamp: now.Format(time.RFC3339Nano),
					Comment:   comment,
				},
			}
			return
		}
		req.Output = &v1.UIRequest_ImageOutput{
			ImageOutput: &v1.ImageOutput{
				Selected:  &v1.ImageOutput_SelectedNumber{SelectedNumber: 0},
				Timestamp: now.Format(time.RFC3339Nano),
				Comment:   comment,
			},
		}
		return
	default:
		req.Output = nil
		return
	}
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
