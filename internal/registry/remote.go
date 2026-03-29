// Package registry provides an HTTP client for push/pull against a remote registry.
package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"hop.top/ben/internal/run"
)

// RemoteClient pushes and pulls runs to/from a remote registry server.
type RemoteClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRemoteClient creates a RemoteClient targeting baseURL.
func NewRemoteClient(baseURL string) *RemoteClient {
	return &RemoteClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// SetHTTPClient replaces the underlying HTTP client (test hook).
func (c *RemoteClient) SetHTTPClient(hc *http.Client) {
	c.httpClient = hc
}

type pushResponse struct {
	ID string `json:"id"`
}

// Push POSTs a run to baseURL/runs and returns the remoteID assigned by the server.
func (c *RemoteClient) Push(ctx context.Context, r *run.Run) (string, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("marshal run: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/runs", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("push request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("push: server returned %d", resp.StatusCode)
	}

	var pr pushResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return "", fmt.Errorf("decode push response: %w", err)
	}
	if pr.ID == "" {
		return "", fmt.Errorf("push: server returned empty id")
	}
	return pr.ID, nil
}

type pullResponse struct {
	Runs []*run.Run `json:"runs"`
}

// Pull GETs runs from baseURL/runs filtered by suite and limited to limit entries.
func (c *RemoteClient) Pull(ctx context.Context, suite string, limit int) ([]*run.Run, error) {
	u, err := url.Parse(c.baseURL + "/runs")
	if err != nil {
		return nil, fmt.Errorf("parse pull url: %w", err)
	}
	q := u.Query()
	if suite != "" {
		q.Set("suite", suite)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build pull request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pull request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("pull: server returned %d", resp.StatusCode)
	}

	var pr pullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("decode pull response: %w", err)
	}
	return pr.Runs, nil
}
