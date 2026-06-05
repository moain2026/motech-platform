// Package handlers implements all HTTP endpoints for the Motech Platform API.
package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"motech-platform/backend/internal/auth"
	"motech-platform/backend/internal/config"
	"motech-platform/backend/internal/models"
	"motech-platform/backend/internal/netbird"
)

// Handler bundles dependencies shared by all endpoints.
type Handler struct {
	DB  *sqlx.DB
	Cfg *config.Config
	Auth *auth.Manager
	NB  *netbird.Client
}

// New builds a Handler.
func New(db *sqlx.DB, cfg *config.Config, am *auth.Manager, nb *netbird.Client) *Handler {
	return &Handler{DB: db, Cfg: cfg, Auth: am, NB: nb}
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// randToken returns a URL-safe random token and its sha256 hex hash.
func randToken() (plain, hash string) {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	plain = hex.EncodeToString(b)
	// format as XXXX-XXXX-XXXX for readability
	formatted := plain[0:4] + "-" + plain[4:8] + "-" + plain[8:12]
	sum := sha256.Sum256([]byte(formatted))
	return formatted, hex.EncodeToString(sum[:])
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func (h *Handler) logActivity(actor, action string, clientID *string, meta map[string]any) {
	b, _ := json.Marshal(meta)
	_, _ = h.DB.Exec(
		`INSERT INTO activity_log (actor, client_id, action, metadata) VALUES ($1,$2,$3,$4)`,
		actor, clientID, action, b,
	)
}

// ---- health ----

// Health reports server + DB status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := h.DB.Ping(); err != nil {
		dbStatus = "error"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"db":           dbStatus,
		"netbird_mode": map[bool]string{true: "mock", false: "live"}[h.NB.IsMock()],
	})
}

// ---- auth ----

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates an admin and returns a JWT.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var a models.Admin
	err := h.DB.Get(&a, `SELECT * FROM admins WHERE email=$1`, req.Email)
	if err != nil || !auth.CheckPassword(a.PasswordHash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	tok, err := h.Auth.Issue(a.ID, "admin", 12*time.Hour)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "email": a.Email})
}

// Me returns the currently authenticated admin's profile.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, _ := r.Context().Value(auth.CtxSubject).(string)
	var a models.Admin
	if err := h.DB.Get(&a, `SELECT * FROM admins WHERE id=$1`, id); err != nil {
		writeErr(w, http.StatusNotFound, "admin not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": a.ID, "email": a.Email, "role": a.Role})
}

type updateMeReq struct {
	CurrentPassword string `json:"current_password"`
	Email           string `json:"email"`
	NewPassword     string `json:"new_password"`
}

// UpdateMe lets the logged-in admin change their email and/or password.
// The CURRENT password is required (re-auth) before any change is applied.
// On success a fresh token is returned (so the session stays valid).
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	id, _ := r.Context().Value(auth.CtxSubject).(string)
	var req updateMeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var a models.Admin
	if err := h.DB.Get(&a, `SELECT * FROM admins WHERE id=$1`, id); err != nil {
		writeErr(w, http.StatusNotFound, "admin not found")
		return
	}
	// Re-authenticate with the current password before allowing changes.
	if !auth.CheckPassword(a.PasswordHash, req.CurrentPassword) {
		writeErr(w, http.StatusUnauthorized, "كلمة المرور الحالية غير صحيحة")
		return
	}

	newEmail := strings.TrimSpace(req.Email)
	if newEmail != "" && newEmail != a.Email {
		// basic email sanity + uniqueness
		if !strings.Contains(newEmail, "@") {
			writeErr(w, http.StatusBadRequest, "بريد غير صالح")
			return
		}
		var n int
		_ = h.DB.Get(&n, `SELECT COUNT(*) FROM admins WHERE email=$1 AND id<>$2`, newEmail, id)
		if n > 0 {
			writeErr(w, http.StatusConflict, "البريد مستخدم بالفعل")
			return
		}
		if _, err := h.DB.Exec(`UPDATE admins SET email=$1 WHERE id=$2`, newEmail, id); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.Email = newEmail
	}

	if req.NewPassword != "" {
		if len(req.NewPassword) < 8 {
			writeErr(w, http.StatusBadRequest, "كلمة المرور الجديدة قصيرة (8 أحرف على الأقل)")
			return
		}
		hash, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "hash error")
			return
		}
		if _, err := h.DB.Exec(`UPDATE admins SET password_hash=$1 WHERE id=$2`, hash, id); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	h.logActivity("admin", "admin.update_profile", nil, nil)
	// Issue a fresh token so the current session keeps working after the change.
	tok, _ := h.Auth.Issue(a.ID, "admin", 12*time.Hour)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "email": a.Email, "token": tok})
}

// ---- clients ----

type createClientReq struct {
	Name         string `json:"name"`
	Branch       string `json:"branch"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
}

// ListClients returns all clients (newest first).
func (h *Handler) ListClients(w http.ResponseWriter, r *http.Request) {
	var cs []models.Client
	if err := h.DB.Select(&cs, `SELECT * FROM clients ORDER BY created_at DESC`); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cs == nil {
		cs = []models.Client{}
	}
	writeJSON(w, http.StatusOK, cs)
}

// CreateClient creates a client + one-time setup token + ssh key placeholder +
// NetBird setup key, and returns the plaintext setup token ONCE.
func (h *Handler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var req createClientReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	tx, err := h.DB.Beginx()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var clientID string
	err = tx.Get(&clientID,
		`INSERT INTO clients (name, branch, contact_name, contact_phone, status)
		 VALUES ($1,$2,$3,$4,'pending') RETURNING id`,
		req.Name, nullable(req.Branch), nullable(req.ContactName), nullable(req.ContactPhone))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	plain, hash := randToken()
	_, err = tx.Exec(
		`INSERT INTO setup_tokens (client_id, token_hash, expires_at)
		 VALUES ($1,$2,$3)`, clientID, hash, time.Now().Add(24*time.Hour))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Backend-generated SSH keypair: private key encrypted at rest (AES-256-GCM),
	// public key handed to the agent to install in authorized_keys.
	pubLine, privPEM, kerr := generateKeypair("motech-" + req.Name)
	if kerr != nil {
		writeErr(w, http.StatusInternalServerError, "keygen: "+kerr.Error())
		return
	}
	privEnc, kerr := auth.Encrypt(h.Cfg.MasterKey, []byte(privPEM))
	if kerr != nil {
		writeErr(w, http.StatusInternalServerError, "encrypt key: "+kerr.Error())
		return
	}
	// rotated_at set NOW: this key was generated server-side and is the current
	// active key (the agent just needs to install the public part; it does not
	// need to generate its own anymore).
	_, err = tx.Exec(
		`INSERT INTO ssh_keys (client_id, active, public_key, private_key_enc, rotated_at)
		 VALUES ($1,true,$2,$3,now())`, clientID, pubLine, privEnc)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// NetBird setup key (mock or live)
	sk, err := h.NB.CreateSetupKey("motech-" + req.Name)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "netbird: "+err.Error())
		return
	}
	_, err = tx.Exec(
		`INSERT INTO netbird_links (client_id, setup_key_ref, setup_key_full) VALUES ($1,$2,$3)`,
		clientID, sk.ID, sk.Key)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logActivity(actorOf(r), "client.create", &clientID, map[string]any{"name": req.Name})

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":             clientID,
		"name":           req.Name,
		"setup_token":    plain, // shown ONCE
		"setup_url":      "/setup/" + plain, // shareable one-link install page
		"netbird_setup":  sk.Key,
		"netbird_mode":   map[bool]string{true: "mock", false: "live"}[h.NB.IsMock()],
		"installer":      "motech-connect.exe",
		"note":           "أعطِ العميل رابط التثبيت (setup_url) — يفتحه ويتبع الخطوات. الرابط يعمل مرة واحدة.",
	})
}

// GetClient returns one client's details.
func (h *Handler) GetClient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var c models.Client
	if err := h.DB.Get(&c, `SELECT * FROM clients WHERE id=$1`, id); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

type updateClientReq struct {
	Name         *string `json:"name"`
	Branch       *string `json:"branch"`
	ContactName  *string `json:"contact_name"`
	ContactPhone *string `json:"contact_phone"`
}

// UpdateClient edits a client's editable metadata (name/branch/contact). Only
// provided fields are changed. Status/keys/tokens are NOT touched here.
func (h *Handler) UpdateClient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateClientReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name cannot be empty")
		return
	}
	res, err := h.DB.Exec(`
		UPDATE clients SET
		  name          = COALESCE($2, name),
		  branch        = COALESCE($3, branch),
		  contact_name  = COALESCE($4, contact_name),
		  contact_phone = COALESCE($5, contact_phone),
		  updated_at    = now()
		WHERE id=$1`,
		id, req.Name, req.Branch, req.ContactName, req.ContactPhone)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	h.logActivity(actorOf(r), "client.update", &id, nil)
	var c models.Client
	_ = h.DB.Get(&c, `SELECT * FROM clients WHERE id=$1`, id)
	writeJSON(w, http.StatusOK, c)
}

// Connection returns the copy-able connection info for AI agents.
func (h *Handler) Connection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var c models.Client
	if err := h.DB.Get(&c, `SELECT * FROM clients WHERE id=$1`, id); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	var nb models.NetbirdLink
	_ = h.DB.Get(&nb, `SELECT * FROM netbird_links WHERE client_id=$1`, id)
	ip := "<netbird-ip-pending>"
	if nb.PeerID != nil && *nb.PeerID != "" {
		ip = *nb.PeerID
	}
	var pubKey string
	_ = h.DB.Get(&pubKey, `SELECT COALESCE(public_key,'') FROM ssh_keys WHERE client_id=$1 AND active=true LIMIT 1`, id)
	var privEnc []byte
	_ = h.DB.Get(&privEnc, `SELECT private_key_enc FROM ssh_keys WHERE client_id=$1 AND active=true LIMIT 1`, id)
	privKey := ""
	if len(privEnc) > 0 {
		if dec, err := auth.Decrypt(h.Cfg.MasterKey, privEnc); err == nil {
			privKey = string(dec)
		}
	}
	h.logActivity(actorOf(r), "client.connection_copied", &id, nil)

	// The agent installs its public key into administrators_authorized_keys, so
	// SSH as Administrator is the reliable path on Windows OpenSSH.
	user := "Administrator"
	keyFile := "motech_" + shortID(id) + ".key"

	// A single copy-paste block an AI agent can run as-is: write the private key
	// to a 0600 file, then ssh in over the NetBird mesh.
	ready := strings.Join([]string{
		"# 1) save the private key",
		"cat > " + keyFile + " <<'MOTECH_KEY'",
		strings.TrimRight(privKey, "\n"),
		"MOTECH_KEY",
		"chmod 600 " + keyFile,
		"# 2) connect (over NetBird mesh)",
		"ssh -i " + keyFile + " -o StrictHostKeyChecking=no " + user + "@" + ip,
	}, "\n")

	writeJSON(w, http.StatusOK, map[string]any{
		"ip":          ip,
		"user":        user,
		"public_key":  pubKey,
		"private_key": privKey,
		"key_file":    keyFile,
		"ssh":         "ssh -i " + keyFile + " " + user + "@" + ip,
		"ready":       ready,
		"note":        "NetBird mesh: الوكيل يحتاج المفتاح الخاص + الاتصال من جهاز منضمّ لنفس شبكة NetBird.",
	})
}

// shortID returns the first 8 chars of a UUID for friendly file names.
func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// RotateKey generates a fresh keypair SERVER-SIDE, stores it encrypted, and
// PERMANENTLY DELETES the previous key(s). The new public key is pushed to the
// agent on its next heartbeat (rotate=true + public_key), which installs it and
// removes the old one. The agent never generates keys itself.
func (h *Handler) RotateKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// fetch client name for the key comment
	var name string
	if err := h.DB.Get(&name, `SELECT name FROM clients WHERE id=$1`, id); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	pubLine, privPEM, kerr := generateKeypair("motech-" + name)
	if kerr != nil {
		writeErr(w, http.StatusInternalServerError, "keygen: "+kerr.Error())
		return
	}
	privEnc, kerr := auth.Encrypt(h.Cfg.MasterKey, []byte(privPEM))
	if kerr != nil {
		writeErr(w, http.StatusInternalServerError, "encrypt: "+kerr.Error())
		return
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()
	// Permanently delete old key(s) — no history kept (per design).
	if _, err := tx.Exec(`DELETE FROM ssh_keys WHERE client_id=$1`, id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Insert the new key as active but NOT yet acknowledged by the agent
	// (rotated_at NULL => heartbeat reports rotate=true until the agent installs).
	if _, err := tx.Exec(
		`INSERT INTO ssh_keys (client_id, active, public_key, private_key_enc, rotated_at)
		 VALUES ($1,true,$2,$3,NULL)`, id, pubLine, privEnc); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logActivity(actorOf(r), "client.rotate_key", &id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pending_rotation": true})
}

// PrivateKey returns the decrypted SSH private key for a client (admin only).
// Every access is recorded in the activity log.
func (h *Handler) PrivateKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var privEnc []byte
	if err := h.DB.Get(&privEnc, `SELECT private_key_enc FROM ssh_keys WHERE client_id=$1 AND active=true LIMIT 1`, id); err != nil || len(privEnc) == 0 {
		writeErr(w, http.StatusNotFound, "no active private key for this client")
		return
	}
	dec, err := auth.Decrypt(h.Cfg.MasterKey, privEnc)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "decrypt failed")
		return
	}
	h.logActivity(actorOf(r), "client.private_key_accessed", &id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"private_key": string(dec)})
}

// DisableClient disables a client and revokes NetBird access.
func (h *Handler) DisableClient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var nb models.NetbirdLink
	_ = h.DB.Get(&nb, `SELECT * FROM netbird_links WHERE client_id=$1`, id)
	if nb.PeerID != nil {
		_ = h.NB.DeletePeer(*nb.PeerID)
	}
	_, err := h.DB.Exec(`UPDATE clients SET status='disabled', updated_at=now() WHERE id=$1`, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logActivity(actorOf(r), "client.disable", &id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "disabled"})
}

// DeleteClient removes a client entirely.
func (h *Handler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var nb models.NetbirdLink
	_ = h.DB.Get(&nb, `SELECT * FROM netbird_links WHERE client_id=$1`, id)
	// Revoke the peer (kills access) AND delete the setup key (keeps the NetBird
	// setup-keys list clean — one-off keys linger there otherwise).
	if nb.PeerID != nil {
		_ = h.NB.DeletePeer(*nb.PeerID)
	}
	if nb.SetupKeyRef != nil {
		_ = h.NB.DeleteSetupKey(*nb.SetupKeyRef)
	}
	if _, err := h.DB.Exec(`DELETE FROM clients WHERE id=$1`, id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logActivity(actorOf(r), "client.delete", &id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- activity ----

// Activity returns recent activity-log entries enriched with the affected
// client's name (so the UI can show "rotated key — <client>" not just the action).
func (h *Handler) Activity(w http.ResponseWriter, r *http.Request) {
	type activityRow struct {
		ID         string    `db:"id" json:"id"`
		Actor      string    `db:"actor" json:"actor"`
		ClientID   *string   `db:"client_id" json:"client_id,omitempty"`
		ClientName *string   `db:"client_name" json:"client_name,omitempty"`
		Action     string    `db:"action" json:"action"`
		CreatedAt  time.Time `db:"created_at" json:"created_at"`
	}
	var rows []activityRow
	if err := h.DB.Select(&rows, `
		SELECT a.id, a.actor, a.client_id, c.name AS client_name, a.action, a.created_at
		FROM activity_log a
		LEFT JOIN clients c ON c.id = a.client_id
		ORDER BY a.created_at DESC LIMIT 200`); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []activityRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

// ---- agent ----

type agentRegisterReq struct {
	Token string `json:"token"`
}

// AgentRegister validates a one-time setup token and onboards the agent.
func (h *Handler) AgentRegister(w http.ResponseWriter, r *http.Request) {
	var req agentRegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeErr(w, http.StatusBadRequest, "token required")
		return
	}
	var st models.SetupToken
	err := h.DB.Get(&st, `SELECT * FROM setup_tokens WHERE token_hash=$1`, hashToken(req.Token))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid token")
		return
	}
	if st.UsedAt != nil {
		writeErr(w, http.StatusConflict, "token already used")
		return
	}
	if time.Now().After(st.ExpiresAt) {
		writeErr(w, http.StatusGone, "token expired")
		return
	}
	now := time.Now()
	_, _ = h.DB.Exec(`UPDATE setup_tokens SET used_at=$1 WHERE id=$2`, now, st.ID)
	_, _ = h.DB.Exec(`UPDATE clients SET status='online', last_seen=$1, updated_at=now() WHERE id=$2`, now, st.ClientID)

	var setupKey string
	_ = h.DB.Get(&setupKey, `SELECT COALESCE(setup_key_full,'') FROM netbird_links WHERE client_id=$1`, st.ClientID)

	// Backend-owned key model: hand the agent the public key to install. The
	// agent does NOT generate keys; the private key stays encrypted in the DB.
	var pubKey string
	_ = h.DB.Get(&pubKey, `SELECT COALESCE(public_key,'') FROM ssh_keys WHERE client_id=$1 AND active=true LIMIT 1`, st.ClientID)

	agentTok, _ := h.Auth.Issue(st.ClientID, "agent", 365*24*time.Hour)
	h.logActivity("agent", "agent.register", &st.ClientID, nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_token":      agentTok,
		"netbird_setupkey": setupKey,
		"netbird_api_url":  h.Cfg.NetbirdAPIURL,
		"heartbeat_secs":   20,
		"install_pubkey":   pubKey,
	})
}

type heartbeatReq struct {
	PeerID    string `json:"peer_id"`
	RotatedOK bool   `json:"rotated_ok"` // agent confirms it installed the latest public key
}

// AgentHeartbeat updates last_seen + the agent's NetBird peer_id, and returns
// pending commands. In the backend-owned key model the server NEVER ingests
// keys from the agent; instead, when a rotation is pending it PUSHES the new
// public key (install_pubkey) for the agent to install. The agent confirms with
// rotated_ok=true, which flips rotated_at and stops the push.
func (h *Handler) AgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	clientID, _ := r.Context().Value(auth.CtxSubject).(string)
	var status string
	err := h.DB.Get(&status, `SELECT status FROM clients WHERE id=$1`, clientID)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unknown client")
		return
	}

	var req heartbeatReq
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.PeerID != "" {
		_, _ = h.DB.Exec(`UPDATE netbird_links SET peer_id=$1 WHERE client_id=$2`, req.PeerID, clientID)
	}
	if req.RotatedOK {
		res, _ := h.DB.Exec(`UPDATE ssh_keys SET rotated_at=now() WHERE client_id=$1 AND active=true AND rotated_at IS NULL`, clientID)
		if res != nil {
			if n, _ := res.RowsAffected(); n > 0 {
				h.logActivity("agent", "agent.rotated_applied", &clientID, nil)
			}
		}
	}

	if status != "disabled" {
		_, _ = h.DB.Exec(`UPDATE clients SET last_seen=now(), status='online', updated_at=now() WHERE id=$1`, clientID)
	}

	// Pending rotation = active key the agent hasn't confirmed installing yet.
	// While pending, push the public key so the agent installs exactly it.
	var pendingRotate bool
	var installPub string
	_ = h.DB.Get(&pendingRotate,
		`SELECT EXISTS(SELECT 1 FROM ssh_keys WHERE client_id=$1 AND active=true AND rotated_at IS NULL)`,
		clientID)
	if pendingRotate {
		_ = h.DB.Get(&installPub, `SELECT COALESCE(public_key,'') FROM ssh_keys WHERE client_id=$1 AND active=true LIMIT 1`, clientID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"disabled":       status == "disabled",
		"rotate":         pendingRotate,
		"install_pubkey": installPub,
	})
}

// ---- small utils ----

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func actorOf(r *http.Request) string {
	if v, ok := r.Context().Value(auth.CtxSubject).(string); ok && v != "" {
		return v
	}
	return "system"
}
