// Package netbird is a thin client for the NetBird REST API. The base URL is
// fully switchable via NETBIRD_API_URL so we can move from NetBird Cloud
// (https://api.netbird.io) to a self-hosted instance by changing one env var.
//
// When no API token is configured the client runs in MOCK mode: it returns
// clearly-marked fake values so the rest of the system works during dev
// without a real NetBird account.
package netbird

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Client talks to a NetBird management API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
	mock    bool
}

// New builds a NetBird client. mock=true (empty token) => no real calls.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 15 * time.Second},
		mock:    token == "",
	}
}

// IsMock reports whether the client is in mock mode.
func (c *Client) IsMock() bool { return c.mock }

// SetupKey is the result of creating a NetBird setup key.
type SetupKey struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

// CreateSetupKey creates a reusable=false ephemeral setup key for one peer.
// In mock mode it returns a fake key prefixed with MOCK-.
func (c *Client) CreateSetupKey(name string) (*SetupKey, error) {
	if c.mock {
		id := uuid.NewString()
		return &SetupKey{ID: "mock-" + id[:8], Key: "MOCK-SETUP-KEY-" + id}, nil
	}
	body := map[string]any{
		"name":       name,
		"type":       "one-off",
		"expires_in": 86400,
		"usage_limit": 1,
	}
	var out SetupKey
	if err := c.do(http.MethodPost, "/api/setup-keys", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletePeer removes a peer (revokes its access). No-op in mock mode.
func (c *Client) DeletePeer(peerID string) error {
	if c.mock || peerID == "" {
		return nil
	}
	return c.do(http.MethodDelete, "/api/peers/"+peerID, nil, nil)
}

func (c *Client) do(method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("netbird %s %s: %d %s", method, path, resp.StatusCode, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
