// Package models defines the database entities (mapped via sqlx struct tags).
package models

import "time"

// Admin is a dashboard user authenticated with email + bcrypt password.
type Admin struct {
	ID           string    `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// Client is a managed remote machine (branch/company).
type Client struct {
	ID           string     `db:"id" json:"id"`
	Name         string     `db:"name" json:"name"`
	Branch       *string    `db:"branch" json:"branch,omitempty"`
	ContactName  *string    `db:"contact_name" json:"contact_name,omitempty"`
	ContactPhone *string    `db:"contact_phone" json:"contact_phone,omitempty"`
	Status       string     `db:"status" json:"status"` // pending|online|offline|disabled
	LastSeen     *time.Time `db:"last_seen" json:"last_seen,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

// SetupToken is a one-time activation token (stored hashed).
type SetupToken struct {
	ID        string     `db:"id" json:"id"`
	ClientID  string     `db:"client_id" json:"client_id"`
	TokenHash string     `db:"token_hash" json:"-"`
	UsedAt    *time.Time `db:"used_at" json:"used_at,omitempty"`
	ExpiresAt time.Time  `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// SSHKey holds a per-client SSH key; the private key is encrypted at rest.
type SSHKey struct {
	ID            string     `db:"id" json:"id"`
	ClientID      string     `db:"client_id" json:"client_id"`
	PublicKey     *string    `db:"public_key" json:"public_key,omitempty"`
	PrivateKeyEnc []byte     `db:"private_key_enc" json:"-"`
	Fingerprint   *string    `db:"fingerprint" json:"fingerprint,omitempty"`
	Active        bool       `db:"active" json:"active"`
	RotatedAt     *time.Time `db:"rotated_at" json:"rotated_at,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

// NetbirdLink maps a client to its NetBird peer/group/setup-key.
type NetbirdLink struct {
	ClientID    string    `db:"client_id" json:"client_id"`
	PeerID      *string   `db:"peer_id" json:"peer_id,omitempty"`
	SetupKeyRef *string   `db:"setup_key_ref" json:"setup_key_ref,omitempty"`
	SetupKeyFull *string  `db:"setup_key_full" json:"-"`
	GroupID     *string   `db:"group_id" json:"group_id,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// ActivityLog is one audit-trail entry.
type ActivityLog struct {
	ID        string    `db:"id" json:"id"`
	Actor     string    `db:"actor" json:"actor"`
	ClientID  *string   `db:"client_id" json:"client_id,omitempty"`
	Action    string    `db:"action" json:"action"`
	Metadata  []byte    `db:"metadata" json:"metadata"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
