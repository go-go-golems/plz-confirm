package client

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/internal/metadata"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

var ErrWaitTimeout = stderrors.New("timeout waiting for response")

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			// Intentionally no global timeout:
			// - wait requests are long-poll based and may legitimately block
			// - we rely on per-request contexts / server-side long-poll timeouts
			Timeout: 0,
		},
	}
}

type CreateRequestParams struct {
	Type     v1.WidgetType
	Input    proto.Message
	TimeoutS int

	SessionID string

	Metadata *v1.RequestMetadata
}

func (c *Client) CreateRequest(ctx context.Context, p CreateRequestParams) (*v1.UIRequest, error) {
	u, err := url.Parse(c.BaseURL + "/api/requests")
	if err != nil {
		return nil, errors.Wrap(err, "parse base url")
	}

	if p.Input == nil {
		return nil, errors.New("input is required")
	}

	reqProto := &v1.UIRequest{
		Type:      p.Type,
		SessionId: p.SessionID,
	}
	if p.Metadata != nil {
		reqProto.Metadata = p.Metadata
	} else {
		reqProto.Metadata = metadata.Collect()
	}
	if p.TimeoutS > 0 {
		reqProto.ExpiresAt = time.Now().UTC().Add(time.Duration(p.TimeoutS) * time.Second).Format(time.RFC3339Nano)
	}

	switch p.Type {
	case v1.WidgetType_widget_type_unspecified:
		return nil, errors.New("invalid widget type")
	case v1.WidgetType_confirm:
		in, ok := p.Input.(*v1.ConfirmInput)
		if !ok {
			return nil, errors.New("input must be *v1.ConfirmInput for type=confirm")
		}
		reqProto.Input = &v1.UIRequest_ConfirmInput{ConfirmInput: in}
	case v1.WidgetType_select:
		in, ok := p.Input.(*v1.SelectInput)
		if !ok {
			return nil, errors.New("input must be *v1.SelectInput for type=select")
		}
		reqProto.Input = &v1.UIRequest_SelectInput{SelectInput: in}
	case v1.WidgetType_form:
		in, ok := p.Input.(*v1.FormInput)
		if !ok {
			return nil, errors.New("input must be *v1.FormInput for type=form")
		}
		reqProto.Input = &v1.UIRequest_FormInput{FormInput: in}
	case v1.WidgetType_upload:
		in, ok := p.Input.(*v1.UploadInput)
		if !ok {
			return nil, errors.New("input must be *v1.UploadInput for type=upload")
		}
		reqProto.Input = &v1.UIRequest_UploadInput{UploadInput: in}
	case v1.WidgetType_table:
		in, ok := p.Input.(*v1.TableInput)
		if !ok {
			return nil, errors.New("input must be *v1.TableInput for type=table")
		}
		reqProto.Input = &v1.UIRequest_TableInput{TableInput: in}
	case v1.WidgetType_image:
		in, ok := p.Input.(*v1.ImageInput)
		if !ok {
			return nil, errors.New("input must be *v1.ImageInput for type=image")
		}
		reqProto.Input = &v1.UIRequest_ImageInput{ImageInput: in}
	default:
		return nil, errors.New("invalid widget type")
	}

	bodyBytes, err := protojson.MarshalOptions{
		UseProtoNames: false,
	}.Marshal(reqProto)
	if err != nil {
		return nil, errors.Wrap(err, "marshal protojson UIRequest")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errors.Wrap(err, "create http request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "post /api/requests")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, errors.Errorf("create request failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errors.Wrap(err, "read create response")
	}

	out := &v1.UIRequest{}
	if err := protojson.Unmarshal(respBytes, out); err != nil {
		return nil, errors.Wrap(err, "protojson unmarshal create response")
	}
	return out, nil
}

func (c *Client) WaitRequest(ctx context.Context, id string, waitTimeoutS int) (*v1.UIRequest, error) {
	// Long-poll loop:
	// - waitTimeoutS > 0 is an overall deadline (seconds)
	// - waitTimeoutS <= 0 waits forever (until ctx is cancelled)
	if waitTimeoutS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(waitTimeoutS)*time.Second)
		defer cancel()
	}

	const defaultPollTimeoutS = 25

	for {
		// If the caller cancelled (or overall deadline elapsed), stop.
		if err := ctx.Err(); err != nil {
			return nil, errors.Wrap(err, "wait cancelled")
		}

		pollTimeoutS := defaultPollTimeoutS
		if dl, ok := ctx.Deadline(); ok {
			remaining := time.Until(dl)
			if remaining <= 0 {
				return nil, ErrWaitTimeout
			}
			// Clamp poll timeout to remaining time; always >= 1s.
			remS := int(remaining.Seconds())
			if remS < 1 {
				remS = 1
			}
			if remS < pollTimeoutS {
				pollTimeoutS = remS
			}
		}

		// Give the HTTP request a little headroom over the server-side poll timeout
		// (network jitter, scheduling).
		reqCtx, cancel := context.WithTimeout(ctx, time.Duration(pollTimeoutS+5)*time.Second)
		out, err := c.waitOnce(reqCtx, id, pollTimeoutS)
		cancel()
		if err == nil {
			return out, nil
		}
		if stderrors.Is(err, ErrWaitTimeout) {
			continue
		}
		return nil, err
	}
}

type UploadImageResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

func (c *Client) UploadImage(ctx context.Context, filePath string, ttlSeconds int) (UploadImageResponse, error) {
	u, err := url.Parse(c.BaseURL + "/api/images")
	if err != nil {
		return UploadImageResponse{}, errors.Wrap(err, "parse base url")
	}

	// Stream multipart body so we don't buffer large files in memory.
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer func() {
			_ = pw.Close()
		}()
		defer func() {
			_ = mw.Close()
		}()

		f, err := os.Open(filePath)
		if err != nil {
			_ = pw.CloseWithError(errors.Wrap(err, "open image file"))
			return
		}
		defer func() {
			_ = f.Close()
		}()

		if ttlSeconds > 0 {
			if err := mw.WriteField("ttlSeconds", strconv.Itoa(ttlSeconds)); err != nil {
				_ = pw.CloseWithError(errors.Wrap(err, "write ttlSeconds field"))
				return
			}
		}

		part, err := mw.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			_ = pw.CloseWithError(errors.Wrap(err, "create multipart file field"))
			return
		}

		if _, err := io.Copy(part, f); err != nil {
			_ = pw.CloseWithError(errors.Wrap(err, "copy file into multipart"))
			return
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), pr)
	if err != nil {
		return UploadImageResponse{}, errors.Wrap(err, "create http request")
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return UploadImageResponse{}, errors.Wrap(err, "post /api/images")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return UploadImageResponse{}, errors.Errorf("upload image failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var out UploadImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return UploadImageResponse{}, errors.Wrap(err, "decode upload image response")
	}
	return out, nil
}

func (c *Client) waitOnce(ctx context.Context, id string, pollTimeoutS int) (*v1.UIRequest, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/requests/%s/wait", c.BaseURL, url.PathEscape(id)))
	if err != nil {
		return nil, errors.Wrap(err, "parse wait url")
	}
	q := u.Query()
	q.Set("timeout", fmt.Sprintf("%d", pollTimeoutS))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "create wait request")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "get /wait")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestTimeout {
		return nil, ErrWaitTimeout
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, errors.Errorf("wait failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errors.Wrap(err, "read wait response")
	}

	out := &v1.UIRequest{}
	if err := protojson.Unmarshal(respBytes, out); err != nil {
		return nil, errors.Wrap(err, "protojson unmarshal wait response")
	}
	return out, nil
}
