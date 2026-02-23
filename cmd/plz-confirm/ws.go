package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newWSCmd(ctx context.Context) *cobra.Command {
	var baseURL string
	var sessionID string
	var pretty bool
	var maxMessages int
	var timeoutS int

	cmd := &cobra.Command{
		Use:   "ws",
		Short: "Connect to the WebSocket and print events",
		RunE: func(cmd *cobra.Command, args []string) error {
			wsURL, err := buildWSURL(baseURL, sessionID)
			if err != nil {
				return err
			}

			cctx := ctx
			if timeoutS > 0 {
				var cancel context.CancelFunc
				cctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutS)*time.Second)
				defer cancel()
			}

			d := websocket.Dialer{}
			conn, _, err := d.DialContext(cctx, wsURL, nil)
			if err != nil {
				return errors.Wrap(err, "dial websocket")
			}
			defer func() { _ = conn.Close() }()

			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "connected: %s\n", wsURL)

			seen := 0
			for {
				select {
				case <-cctx.Done():
					return nil
				default:
				}

				_, msg, err := conn.ReadMessage()
				if err != nil {
					return errors.Wrap(err, "read websocket message")
				}

				if pretty {
					var v any
					if err := json.Unmarshal(msg, &v); err == nil {
						b, _ := json.MarshalIndent(v, "", "  ")
						_, _ = fmt.Fprintln(w, string(b))
						seen++
					} else {
						_, _ = fmt.Fprintln(w, string(msg))
						seen++
					}
				} else {
					_, _ = fmt.Fprintln(w, string(msg))
					seen++
				}

				if maxMessages > 0 && seen >= maxMessages {
					return nil
				}
			}
		},
	}

	cmd.Flags().StringVar(&baseURL, "base-url", "http://localhost:3000", "Base URL (http/https) to derive the WebSocket URL from")
	cmd.Flags().StringVar(&sessionID, "session-id", "global", "Session ID to subscribe to")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "Pretty-print JSON messages")
	cmd.Flags().IntVar(&maxMessages, "count", 0, "Exit after N messages (0 = run until canceled)")
	cmd.Flags().IntVar(&timeoutS, "timeout", 0, "Overall timeout in seconds (0 = no timeout)")
	return cmd
}

func buildWSURL(baseURL, sessionID string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.Wrap(err, "parse base url")
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", errors.Errorf("unsupported base url scheme: %s", u.Scheme)
	}
	basePath := strings.TrimSuffix(u.Path, "/")
	switch {
	case basePath == "":
		u.Path = "/ws"
	case strings.HasSuffix(basePath, "/ws"):
		u.Path = basePath
	default:
		u.Path = basePath + "/ws"
	}
	q := u.Query()
	q.Set("sessionId", sessionID)
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String(), nil
}
