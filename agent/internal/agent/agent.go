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
	"strings"
	"time"
)

// Agent holds runtime state and on-disk configuration.
type Agent struct {
	Server string
	http   *http.Client
	state  *State
	tokens *TokenManager
}

// State is persisted locally after registration.
type State struct {
	AgentToken    string `json:"agent_token"`
	NetbirdKey    string `json:"netbird_setupkey"`
	NetbirdAPIURL string `json:"netbird_api_url"`
	HeartbeatSecs int    `json:"heartbeat_secs"`
	Server        string `json:"server"`
	SSHPublicKey  string `json:"ssh_public_key"`
	SSHPrivateKey string `json:"ssh_private_key"`
	RotatePending bool   `json:"rotate_pending"`
	RotateApplied bool   `json:"rotate_applied"`
	// RotateConfirmPending is true after we apply a rotation locally and need to
	// tell the server (rotated_ok=true) until it acknowledges (rotate=false).
	// This replaces the old "always send rotated_ok" behaviour that spammed the
	// server every heartbeat and could prematurely confirm a NEW rotation.
	RotateConfirmPending bool `json:"rotate_confirm_pending"`
}

// New creates an Agent pointed at the given backend base URL. If a previously
// saved state has a server, it takes precedence (so the Scheduled Task uses the
// real server, not the CLI default).
func New(server string) *Agent {
	a := &Agent{Server: server, http: &http.Client{Timeout: 15 * time.Second}}
	a.state, _ = loadState()
	if a.state != nil && a.state.Server != "" {
		a.Server = a.state.Server
	}
	initTok := ""
	if a.state != nil {
		initTok = a.state.AgentToken
	}
	a.tokens = NewTokenManager(initTok)
	return a
}

// authToken returns the freshest known agent token (TTL-cached, disk-backed).
func (a *Agent) authToken() string {
	if a.tokens != nil {
		if t := a.tokens.Get(); t != "" {
			return t
		}
	}
	if a.state != nil {
		return a.state.AgentToken
	}
	return ""
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
	path, err := a.EnsureNetbirdInstalled()
	if err != nil {
		return fmt.Errorf("تعذّر تثبيت NetBird تلقائياً: %w", err)
	}
	// Ensure the NetBird background service is installed & running first.
	_ = exec.Command(path, "service", "install").Run()
	_ = exec.Command(path, "service", "start").Run()
	time.Sleep(2 * time.Second)

	args := []string{"up", "--setup-key", a.state.NetbirdKey}
	if a.state.NetbirdAPIURL != "" {
		args = append(args, "--management-url", a.state.NetbirdAPIURL)
	}
	out, err := exec.Command(path, args...).CombinedOutput()
	log.Printf("netbird up: %s", string(out))
	if err != nil {
		return fmt.Errorf("netbird up: %w (%s)", err, string(out))
	}
	return nil
}

// netbirdPeerIP returns this machine's NetBird IP via `netbird status`, or "".
func netbirdPeerIP() string {
	p, err := exec.LookPath("netbird")
	if err != nil {
		return ""
	}
	out, err := exec.Command(p, "status", "--json").Output()
	if err != nil {
		return ""
	}
	var st struct {
		NetbirdIP string `json:"netbirdIp"`
	}
	if json.Unmarshal(out, &st) == nil {
		// strip CIDR suffix (e.g. 100.95.255.69/16 -> 100.95.255.69)
		if i := strings.IndexByte(st.NetbirdIP, '/'); i > 0 {
			return st.NetbirdIP[:i]
		}
		return st.NetbirdIP
	}
	return ""
}

// SetupAccess generates the initial SSH keypair and installs the public key
// into the OS authorized-keys file (idempotent; safe to call on first install).
func (a *Agent) SetupAccess() error {
	if a.state == nil {
		return fmt.Errorf("not registered")
	}
	if err := a.applyKeyRotation(); err != nil {
		return err
	}
	_ = a.state.save()
	// Install + start the OS SSH server, open firewall, fix ACLs (Windows).
	return ensureSSHServer()
}

// Heartbeat sends one heartbeat (reporting peer_id + public key) and returns
// the pending-commands response.
//
// On a 401 ("unknown client") it re-reads the token from disk EXACTLY ONCE and
// retries — this recovers from a stale in-memory token (e.g. the run-loop loaded
// the token before register finished writing agent.json). If the reloaded token
// is unchanged, it does NOT retry (avoids a pointless second call / loop).
func (a *Agent) Heartbeat() (map[string]any, error) {
	if a.authToken() == "" {
		return nil, fmt.Errorf("not registered")
	}

	tok := a.authToken()
	out, status, err := a.sendHeartbeat(tok)
	if err == nil {
		return out, nil
	}
	logTokenEvent("hb", fmt.Sprintf("status=%d tokTail=%s err=%v", status, tail8(tok), err))
	if status != http.StatusUnauthorized {
		return nil, err // network or non-401 error: surface as-is
	}

	// 401 path: reload token from disk once and decide whether to retry.
	newTok, changed := a.tokens.ForceReload()
	if !changed {
		logTokenEvent("401→reload", "token unchanged → no retry")
		return nil, err
	}
	logTokenEvent("401→reload", "token changed → retrying once")
	if a.state != nil {
		a.state.AgentToken = newTok // keep state in sync for other callers
	}
	out, status, err = a.sendHeartbeat(newTok)
	if err == nil {
		logTokenEvent("401→reload→retry", "result=OK")
		return out, nil
	}
	logTokenEvent("401→reload→retry", fmt.Sprintf("result=FAIL status=%d", status))
	return nil, err
}

// sendHeartbeat performs one heartbeat POST with the given token. It returns the
// decoded body, the HTTP status (0 on transport error), and an error.
func (a *Agent) sendHeartbeat(token string) (map[string]any, int, error) {
	var pub, priv string
	var rotated bool
	if a.state != nil {
		pub, priv = a.state.SSHPublicKey, a.state.SSHPrivateKey
		rotated = a.state.RotateConfirmPending // only while a just-applied rotation awaits server ack
	}
	payload := map[string]any{
		"peer_id":     netbirdPeerIP(),
		"public_key":  pub,
		"private_key": priv,
		"rotated_ok":  rotated,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, a.Server+"/api/agent/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, resp.StatusCode, fmt.Errorf("heartbeat %d: %s", resp.StatusCode, string(b))
	}
	var out map[string]any
	return out, resp.StatusCode, json.NewDecoder(resp.Body).Decode(&out)
}

// tail8 returns the last 8 chars of s (for safe token identification in logs).
func tail8(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[len(s)-8:]
}

// loop runs the heartbeat cycle until stop is closed.
func (a *Agent) loop(stop <-chan struct{}) {
	// Persistent file log (helps diagnose service/task runs).
	if lf, err := os.OpenFile(filepath.Join(filepath.Dir(statePath()), "agent.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(lf)
	}
	// Reload state in case the process started fresh (e.g. as SYSTEM).
	if a.state == nil || a.state.AgentToken == "" {
		if s, err := loadState(); err == nil {
			a.state = s
		}
	}
	if a.state != nil && a.state.Server != "" {
		a.Server = a.state.Server
	}
	log.Printf("loop start: server=%s hasToken=%v BUILD=tokenmgr-v3 tokTail=%s", a.Server, a.state != nil && a.state.AgentToken != "", tail8(a.authToken()))
	interval := 20 * time.Second
	if a.state != nil && a.state.HeartbeatSecs > 0 {
		interval = time.Duration(a.state.HeartbeatSecs) * time.Second
	}
	// Fire one heartbeat immediately so the dashboard sees us right away.
	if cmds, err := a.Heartbeat(); err == nil {
		a.applyCommands(cmds)
	} else {
		log.Printf("initial heartbeat error: %v", err)
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
//
// Rotation state machine:
//   - server sets rotate=true (active key has rotated_at IS NULL)
//   - we generate+install a fresh keypair, set RotateConfirmPending=true; the
//     NEXT heartbeat sends rotated_ok=true + the new public/private key.
//   - server flips rotated_at=now() and then reports rotate=false; we see that
//     and clear RotateConfirmPending so we stop sending rotated_ok.
// This is idempotent: if rotate stays true (server hasn't acked yet) we keep
// sending rotated_ok without re-generating the key.
func (a *Agent) applyCommands(cmds map[string]any) {
	if b, _ := cmds["disabled"].(bool); b {
		log.Println("server says DISABLED — removing access & leaving mesh")
		if p, err := exec.LookPath("netbird"); err == nil {
			_ = exec.Command(p, "down").Run()
		}
		return
	}
	rotate, _ := cmds["rotate"].(bool)
	if rotate {
		if a.state.RotateConfirmPending {
			// Already applied this rotation; just keep confirming until ack.
			return
		}
		log.Println("server requests key rotation — applying")
		if err := a.applyKeyRotation(); err != nil {
			log.Printf("rotation failed: %v", err)
			return
		}
		a.state.RotatePending = true
		a.state.RotateApplied = true
		a.state.RotateConfirmPending = true
		_ = a.state.save()
		log.Println("key rotation applied — will confirm on next heartbeat")
		return
	}
	// rotate=false: if we were waiting for an ack, the server has now confirmed.
	if a.state.RotateConfirmPending {
		a.state.RotateConfirmPending = false
		_ = a.state.save()
		log.Println("server acknowledged key rotation — confirm cycle complete")
	}
}

