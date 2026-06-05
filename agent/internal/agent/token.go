package agent

import (
	"log"
	"sync"
	"time"
)

// TokenManager is a thread-safe cache of the agent auth token read from
// agent.json. It exists to solve a real bug we observed in the field:
//
//	The Scheduled Task launches the `run` process which loads the token ONCE
//	at startup. If that read raced with `register` finalizing agent.json (or if
//	the token is later rotated on disk), the in-memory token went stale and the
//	backend returned 401 "unknown client" forever, even though the token on disk
//	was valid.
//
// TokenManager fixes this by:
//   - caching the token with a 30s TTL (re-reads disk when the cache expires)
//   - exposing ForceReload() so a 401 can trigger an immediate disk re-read
//   - reading the file with a small retry (3 x 100ms) to survive a transient
//     Windows file lock while register/save is writing.
//
// It is also the seam where future server-driven *token* rotation plugs in.
type TokenManager struct {
	mu        sync.RWMutex
	token     string
	loadedAt  time.Time
	ttl       time.Duration
	readState func() (*State, error) // injectable for tests; defaults to loadState
}

// NewTokenManager seeds the manager with an initial token (may be empty) and a
// 30-second freshness TTL.
func NewTokenManager(initial string) *TokenManager {
	return &TokenManager{
		token:     initial,
		loadedAt:  time.Now(),
		ttl:       30 * time.Second,
		readState: loadState,
	}
}

// Get returns the current token. If the cache is older than the TTL it re-reads
// agent.json first (best-effort; keeps the old token if the read fails).
func (m *TokenManager) Get() string {
	m.mu.RLock()
	expired := time.Since(m.loadedAt) > m.ttl
	tok := m.token
	m.mu.RUnlock()
	if expired {
		if v, ok := m.reloadFromDisk(); ok {
			return v
		}
	}
	return tok
}

// ForceReload re-reads the token from disk immediately (used after a 401).
// Returns the (possibly new) token and whether it CHANGED from the cached one.
func (m *TokenManager) ForceReload() (token string, changed bool) {
	m.mu.RLock()
	old := m.token
	m.mu.RUnlock()
	v, ok := m.reloadFromDisk()
	if !ok {
		return old, false
	}
	return v, v != old
}

// reloadFromDisk reads agent.json with a short retry to survive a file lock.
// On success it updates the cache and returns (token, true).
func (m *TokenManager) reloadFromDisk() (string, bool) {
	var st *State
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		st, err = m.readState()
		if err == nil && st != nil && st.AgentToken != "" {
			break
		}
		if attempt < 2 {
			time.Sleep(100 * time.Millisecond) // file-lock backoff
		}
	}
	if err != nil || st == nil || st.AgentToken == "" {
		return "", false
	}
	m.mu.Lock()
	m.token = st.AgentToken
	m.loadedAt = time.Now()
	m.mu.Unlock()
	return st.AgentToken, true
}

// logTokenEvent emits a single structured line for the 401→reload flow so the
// behaviour is easy to verify in agent.log.
func logTokenEvent(stage, detail string) {
	log.Printf("token: %s %s", stage, detail)
}
