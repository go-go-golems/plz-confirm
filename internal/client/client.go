package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/internal/types"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
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
	if waitTimeoutS <= 0 {
		waitTimeoutS = 60
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/requests/%s/wait", c.BaseURL, url.PathEscape(id)))
	if err != nil {
		return types.UIRequest{}, errors.Wrap(err, "parse wait url")
	}
	q := u.Query()
	q.Set("timeout", fmt.Sprintf("%d", waitTimeoutS))
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
		return types.UIRequest{}, errors.New("timeout waiting for response")
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
