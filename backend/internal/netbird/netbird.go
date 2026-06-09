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
	"strings"
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
		"name":        name,
		"type":        "one-off",
		"expires_in":  86400,
		"usage_limit": 1,
		"auto_groups": []string{}, // assign client groups here later for ACLs
	}
	var out SetupKey
	if err := c.do(http.MethodPost, "/api/setup-keys", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletePeer removes a peer (revokes its access). No-op in mock mode.
// The argument may be a NetBird peer object id OR a NetBird IP; if it looks
// like an IP, it is resolved to the peer id first.
func (c *Client) DeletePeer(peerIDOrIP string) error {
	if c.mock || peerIDOrIP == "" {
		return nil
	}
	id := peerIDOrIP
	if strings.Count(peerIDOrIP, ".") == 3 { // looks like an IPv4 address
		if resolved, err := c.peerIDByIP(peerIDOrIP); err == nil && resolved != "" {
			id = resolved
		} else {
			return fmt.Errorf("peer not found for ip %s", peerIDOrIP)
		}
	}
	return c.do(http.MethodDelete, "/api/peers/"+id, nil, nil)
}

// DeleteSetupKey revokes/deletes a setup key by its NetBird key id. No-op in
// mock mode or when the id is empty (or a mock id). NetBird keeps used one-off
// keys listed until deleted, so we remove ours when a client is deleted to keep
// the NetBird setup-keys page clean.
func (c *Client) DeleteSetupKey(keyID string) error {
	if c.mock || keyID == "" || strings.HasPrefix(keyID, "mock-") {
		return nil
	}
	return c.do(http.MethodDelete, "/api/setup-keys/"+keyID, nil, nil)
}

// peerIDByIP finds a NetBird peer object id by its mesh IP.
// PeerLiveStatus returns a map[netbirdIP]connected for ALL peers, in one call.
// NetBird is the source of truth for reachability — the dashboard uses this to
// show real online/offline instead of relying solely on the agent heartbeat.
// Empty map in mock mode or on error.
func (c *Client) PeerLiveStatus() map[string]bool {
	out := map[string]bool{}
	if c.mock {
		return out
	}
	var peers []struct {
		IP        string `json:"ip"`
		Connected bool   `json:"connected"`
	}
	if err := c.do(http.MethodGet, "/api/peers", nil, &peers); err != nil {
		return out
	}
	for _, p := range peers {
		out[p.IP] = p.Connected
	}
	return out
}

func (c *Client) peerIDByIP(ip string) (string, error) {
	var peers []struct {
		ID string `json:"id"`
		IP string `json:"ip"`
	}
	if err := c.do(http.MethodGet, "/api/peers", nil, &peers); err != nil {
		return "", err
	}
	for _, p := range peers {
		if p.IP == ip {
			return p.ID, nil
		}
	}
	return "", nil
}

// EnableSSH turns on NetBird's built-in SSH server flag for the peer with the
// given NetBird IP. Idempotent. No-op in mock mode or if the peer isn't found.
// This is what makes a freshly-installed client reachable over the mesh with
// zero manual dashboard steps.
func (c *Client) EnableSSH(peerIP string) error {
	if c.mock || peerIP == "" {
		return nil
	}
	// Fetch the peer (need its id + current name; NetBird's PUT /peers/{id}
	// requires the name field or it silently no-ops).
	var peers []struct {
		ID   string `json:"id"`
		IP   string `json:"ip"`
		Name string `json:"name"`
	}
	if err := c.do(http.MethodGet, "/api/peers", nil, &peers); err != nil {
		return err
	}
	var id, name string
	for _, p := range peers {
		if p.IP == peerIP {
			id, name = p.ID, p.Name
			break
		}
	}
	if id == "" {
		return nil // peer not found yet
	}
	body := map[string]any{
		"name":                          name,
		"ssh_enabled":                   true,
		"login_expiration_enabled":      false,
		"inactivity_expiration_enabled": false,
	}
	return c.do(http.MethodPut, "/api/peers/"+id, body, nil)
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
