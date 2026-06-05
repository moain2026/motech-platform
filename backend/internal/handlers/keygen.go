package handlers

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// generateKeypair creates a fresh ed25519 SSH keypair and returns the OpenSSH
// authorized_keys public line and the OpenSSH PEM private key.
//
// The backend now owns key generation (per security design): the private key is
// created here, stored AES-256-GCM-encrypted at rest, and the public key is
// handed to the agent to install in administrators_authorized_keys. The private
// key is only ever returned to an authenticated admin over HTTPS.
func generateKeypair(comment string) (pubLine, privPEM string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ed25519: %w", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", "", fmt.Errorf("ssh public key: %w", err)
	}
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	if comment != "" {
		line += " " + comment
	}
	pemBlock, err := ssh.MarshalPrivateKey(priv, comment)
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}
	return line, string(pem.EncodeToMemory(pemBlock)), nil
}
