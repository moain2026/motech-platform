// Package agent implements the Motech client agent logic: registration with
// the backend, joining the NetBird mesh, running as a service, and sending
// periodic heartbeats that apply pending commands (rotate / disable).
package agent

import (
	"bytes"
	"context"
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
//
// Backend-owned key model: the server generates the keypair and tells us which
// PUBLIC key to install via install_pubkey (at register and on rotation). We
// never generate or hold the private key.
type State struct {
	AgentToken    string `json:"agent_token"`
	NetbirdKey    string `json:"netbird_setupkey"`
	NetbirdAPIURL string `json:"netbird_api_url"`
	HeartbeatSecs int    `json:"heartbeat_secs"`
	Server        string `json:"server"`
	InstallPubKey string `json:"install_pubkey"` // public key the backend told us to install
	SSHPublicKey  string `json:"ssh_public_key"`  // the public key currently installed
	SSHPrivateKey string `json:"ssh_private_key"` // legacy/unused in backend-owned model
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
	// Retry with backoff so a weak/intermittent connection doesn't fail the
	// install on the first network blip. Up to 5 attempts: 2s,4s,8s,16s.
	var resp *http.Response
	var err error
	for attempt := 1; attempt <= 5; attempt++ {
		resp, err = a.http.Post(a.Server+"/api/agent/register", "application/json", bytes.NewReader(body))
		if err == nil && resp.StatusCode < 500 {
			break // success or a definitive client error (e.g. bad token) — don't retry
		}
		if resp != nil {
			resp.Body.Close()
		}
		if attempt < 5 {
			wait := time.Duration(1<<attempt) * time.Second
			log.Printf("register attempt %d failed (%v), retrying in %s", attempt, err, wait)
			time.Sleep(wait)
		}
	}
	if err != nil {
		return fmt.Errorf("فشل الاتصال بالخادم بعد 5 محاولات: %w", err)
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
	if err := s.save(); err != nil {
		return err
	}
	// Backend-owned key: install the public key the server generated for us.
	if s.InstallPubKey != "" {
		if err := a.installServerPublicKey(s.InstallPubKey); err != nil {
			log.Printf("warn: install server public key: %v", err)
		}
	}
	return nil
}

// installServerPublicKey installs a backend-provided public key into the OS
// authorized-keys file and ensures the SSH server is running. It records the
// installed key in state. Used at register and on rotation.
func (a *Agent) installServerPublicKey(pubLine string) error {
	if err := installAuthorizedKey(pubLine); err != nil {
		return err
	}
	a.state.SSHPublicKey = pubLine
	a.state.InstallPubKey = pubLine
	_ = a.state.save()
	return ensureSSHServer()
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
	// Each command is bounded by a timeout so a hung CLI can't stall install.
	installCtx, cancelInstall := context.WithTimeout(context.Background(), 30*time.Second)
	_ = silentCmdCtx(installCtx, path, "service", "install").Run()
	_ = silentCmdCtx(installCtx, path, "service", "start").Run()
	cancelInstall()
	time.Sleep(2 * time.Second)

	// Enable NetBird's BUILT-IN SSH server (--allow-server-ssh): SSH arrives on
	// port 22 and is re-routed internally to 22022 over the mesh. This avoids
	// installing Windows OpenSSH + administrators_authorized_keys ACL + firewall
	// (the classic failure source). --disable-ssh-auth uses machine-identity
	// (NetBird ACLs) instead of interactive JWT/OIDC, so install stays headless.
	args := []string{"up", "--setup-key", a.state.NetbirdKey,
		"--allow-server-ssh", "--disable-ssh-auth"}
	if a.state.NetbirdAPIURL != "" {
		args = append(args, "--management-url", a.state.NetbirdAPIURL)
	}
	// CRITICAL: `netbird up` can block indefinitely (e.g. waiting on interactive
	// SSO login when a setup-key is rejected). Bound it so the install always
	// proceeds; a join failure is reported but does not freeze the installer.
	upCtx, cancelUp := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelUp()
	out, err := silentCmdCtx(upCtx, path, args...).CombinedOutput()
	log.Printf("netbird up: %s", string(out))
	if upCtx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("netbird up تجاوز المهلة (45s) — تحقق من setup-key/الاتصال")
	}
	if err != nil {
		// Older clients may not know --allow-server-ssh/--disable-ssh-auth.
		// Retry once without them so the join still succeeds (OpenSSH fallback).
		if strings.Contains(string(out), "unknown flag") || strings.Contains(string(out), "unknown shorthand") {
			log.Printf("netbird: ssh flags unsupported, retrying basic up")
			basic := []string{"up", "--setup-key", a.state.NetbirdKey}
			if a.state.NetbirdAPIURL != "" {
				basic = append(basic, "--management-url", a.state.NetbirdAPIURL)
			}
			ctx2, c2 := context.WithTimeout(context.Background(), 45*time.Second)
			defer c2()
			out2, err2 := silentCmdCtx(ctx2, path, basic...).CombinedOutput()
			log.Printf("netbird up (basic): %s", string(out2))
			if err2 != nil {
				return fmt.Errorf("netbird up: %w (%s)", err2, string(out2))
			}
			return nil
		}
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
	out, err := silentCmd(p, "status", "--json").Output()
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

// InstalledPubKey returns the public key the backend handed us at register
// (used to verify the key actually landed in authorized_keys).
func (a *Agent) InstalledPubKey() string {
	if a.state == nil {
		return ""
	}
	if a.state.InstallPubKey != "" {
		return a.state.InstallPubKey
	}
	return a.state.SSHPublicKey
}

// SetupAccess installs the backend-provided public key (idempotent) and ensures
// the OS SSH server is running. In the backend-owned key model the key is
// generated by the server and delivered via install_pubkey at register; the
// agent does NOT generate keys.
func (a *Agent) SetupAccess() error {
	if a.state == nil {
		return fmt.Errorf("not registered")
	}
	pub := a.state.InstallPubKey
	if pub == "" {
		pub = a.state.SSHPublicKey
	}
	if pub == "" {
		// Nothing to install yet; just ensure the SSH server is up.
		return ensureSSHServer()
	}
	return a.installServerPublicKey(pub)
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
	var rotated bool
	if a.state != nil {
		rotated = a.state.RotateConfirmPending // only while a just-installed rotation awaits server ack
	}
	// Backend-owned key model: we report only peer_id + rotation ack. We never
	// send keys to the server (it generates and holds them).
	payload := map[string]any{
		"peer_id":    netbirdPeerIP(),
		"rotated_ok": rotated,
		"login_user": currentLoginUser(),
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
			_ = silentCmd(p, "down").Run()
		}
		return
	}
	rotate, _ := cmds["rotate"].(bool)
	if rotate {
		// Backend-owned model: install the PUBLIC key the server pushed.
		newPub, _ := cmds["install_pubkey"].(string)
		if newPub == "" {
			log.Println("rotate requested but no install_pubkey provided — skipping")
			return
		}
		if a.state.RotateConfirmPending && a.state.SSHPublicKey == newPub {
			// Already installed this key; keep confirming until server acks.
			return
		}
		log.Println("server requests key rotation — installing pushed public key")
		if err := a.installServerPublicKey(newPub); err != nil {
			log.Printf("rotation install failed: %v", err)
			return
		}
		a.state.RotateConfirmPending = true
		_ = a.state.save()
		log.Println("new public key installed — will confirm on next heartbeat")
		return
	}
	// rotate=false: if we were waiting for an ack, the server has now confirmed.
	if a.state.RotateConfirmPending {
		a.state.RotateConfirmPending = false
		_ = a.state.save()
		log.Println("server acknowledged key rotation — confirm cycle complete")
	}
}

