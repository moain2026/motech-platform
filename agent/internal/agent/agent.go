// Package agent implements the Motech client agent logic: registration with
// the backend, joining the NetBird mesh, running as a service, and sending
// periodic heartbeats that apply pending commands (rotate / disable).
package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Agent holds runtime state and on-disk configuration.
type Agent struct {
	Server string
	http   *http.Client
	state  *State
}

// State is persisted locally after registration.
type State struct {
	AgentToken    string `json:"agent_token"`
	NetbirdKey    string `json:"netbird_setupkey"`
	NetbirdAPIURL string `json:"netbird_api_url"`
	HeartbeatSecs int    `json:"heartbeat_secs"`
	Server        string `json:"server"`
}

// New creates an Agent pointed at the given backend base URL.
func New(server string) *Agent {
	a := &Agent{Server: server, http: &http.Client{Timeout: 15 * time.Second}}
	a.state, _ = loadState()
	return a
}

// statePath returns the OS-appropriate config file path.
func statePath() string {
	if runtime.GOOS == "windows" {
		dir := os.Getenv("ProgramData")
		if dir == "" {
			dir = `C:\ProgramData`
		}
		return filepath.Join(dir, "Motech", "agent.json")
	}
	return filepath.Join(os.TempDir(), "motech-agent.json")
}

func loadState() (*State, error) {
	b, err := os.ReadFile(statePath())
	if err != nil {
		return &State{}, err
	}
	var s State
	return &s, json.Unmarshal(b, &s)
}

func (s *State) save() error {
	if err := os.MkdirAll(filepath.Dir(statePath()), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(statePath(), b, 0o600)
}

// Register exchanges a one-time setup token for an agent token + NetBird key.
func (a *Agent) Register(token string) error {
	body, _ := json.Marshal(map[string]string{"token": token})
	resp, err := a.http.Post(a.Server+"/api/agent/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server %d: %s", resp.StatusCode, string(b))
	}
	var s State
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return err
	}
	s.Server = a.Server
	if s.HeartbeatSecs == 0 {
		s.HeartbeatSecs = 20
	}
	a.state = &s
	return s.save()
}

// JoinNetbird installs/connects NetBird using the stored setup key.
// On non-Windows or when the netbird CLI is missing, it logs and continues.
func (a *Agent) JoinNetbird() error {
	if a.state == nil || a.state.NetbirdKey == "" {
		return fmt.Errorf("no netbird setup key (register first)")
	}
	path, err := exec.LookPath("netbird")
	if err != nil {
		log.Printf("netbird CLI not found — install it, then run: netbird up --setup-key %s --management-url %s",
			mask(a.state.NetbirdKey), a.state.NetbirdAPIURL)
		return nil
	}
	args := []string{"up", "--setup-key", a.state.NetbirdKey}
	if a.state.NetbirdAPIURL != "" {
		args = append(args, "--management-url", a.state.NetbirdAPIURL)
	}
	out, err := exec.Command(path, args...).CombinedOutput()
	log.Printf("netbird up: %s", string(out))
	return err
}

// Heartbeat sends one heartbeat and returns the pending-commands response.
func (a *Agent) Heartbeat() (map[string]any, error) {
	if a.state == nil || a.state.AgentToken == "" {
		return nil, fmt.Errorf("not registered")
	}
	req, _ := http.NewRequest(http.MethodPost, a.Server+"/api/agent/heartbeat", nil)
	req.Header.Set("Authorization", "Bearer "+a.state.AgentToken)
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("heartbeat %d: %s", resp.StatusCode, string(b))
	}
	var out map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

// loop runs the heartbeat cycle until stop is closed.
func (a *Agent) loop(stop <-chan struct{}) {
	interval := 20 * time.Second
	if a.state != nil && a.state.HeartbeatSecs > 0 {
		interval = time.Duration(a.state.HeartbeatSecs) * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			cmds, err := a.Heartbeat()
			if err != nil {
				log.Printf("heartbeat error: %v", err)
				continue
			}
			a.applyCommands(cmds)
		}
	}
}

// applyCommands reacts to server-issued commands.
func (a *Agent) applyCommands(cmds map[string]any) {
	if b, _ := cmds["disabled"].(bool); b {
		log.Println("server says DISABLED — removing access & leaving mesh")
		if p, err := exec.LookPath("netbird"); err == nil {
			_ = exec.Command(p, "down").Run()
		}
		return
	}
	if b, _ := cmds["rotate"].(bool); b {
		log.Println("server requests key rotation — applying (placeholder)")
		// Phase 3.1: fetch new key/material and update local SSH config.
	}
}

func mask(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
