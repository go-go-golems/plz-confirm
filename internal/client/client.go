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

	"github.com/go-go-golems/plz-confirm/internal/types"
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
	Type     types.WidgetType `json:"type"`
	Input    any              `json:"input"`
	TimeoutS int              `json:"timeout,omitempty"`

	// Compatibility with the Node server shape. The Go server ignores sessions,
	// but we keep the field so old clients remain valid.
	SessionID string `json:"sessionId,omitempty"`
}

func (c *Client) CreateRequest(ctx context.Context, p CreateRequestParams) (types.UIRequest, error) {
	u, err := url.Parse(c.BaseURL + "/api/requests")
	if err != nil {
		return types.UIRequest{}, errors.Wrap(err, "parse base url")
	}

	bodyBytes, err := json.Marshal(p)
	if err != nil {
		return types.UIRequest{}, errors.Wrap(err, "marshal create request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return types.UIRequest{}, errors.Wrap(err, "create http request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return types.UIRequest{}, errors.Wrap(err, "post /api/requests")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return types.UIRequest{}, errors.Errorf("create request failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var out types.UIRequest
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return types.UIRequest{}, errors.Wrap(err, "decode create response")
	}
	return out, nil
}

func (c *Client) WaitRequest(ctx context.Context, id string, waitTimeoutS int) (types.UIRequest, error) {
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
			return types.UIRequest{}, errors.Wrap(err, "wait cancelled")
		}

		pollTimeoutS := defaultPollTimeoutS
		if dl, ok := ctx.Deadline(); ok {
			remaining := time.Until(dl)
			if remaining <= 0 {
				return types.UIRequest{}, ErrWaitTimeout
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
		return types.UIRequest{}, err
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

func (c *Client) waitOnce(ctx context.Context, id string, pollTimeoutS int) (types.UIRequest, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/requests/%s/wait", c.BaseURL, url.PathEscape(id)))
	if err != nil {
		return types.UIRequest{}, errors.Wrap(err, "parse wait url")
	}
	q := u.Query()
	q.Set("timeout", fmt.Sprintf("%d", pollTimeoutS))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return types.UIRequest{}, errors.Wrap(err, "create wait request")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return types.UIRequest{}, errors.Wrap(err, "get /wait")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestTimeout {
		return types.UIRequest{}, ErrWaitTimeout
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return types.UIRequest{}, errors.Errorf("wait failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var out types.UIRequest
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return types.UIRequest{}, errors.Wrap(err, "decode wait response")
	}
	return out, nil
}
