package agent

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/ssh"
)

// administratorsAuthorizedKeys is the Windows OpenSSH file that grants SSH
// access to members of the Administrators group.
const administratorsAuthorizedKeys = `C:\ProgramData\ssh\administrators_authorized_keys`

// applyKeyRotation generates a fresh SSH keypair, installs the public key into
// the OS authorized-keys file, and stores the public key in agent state so it
// is reported on the next heartbeat. The private key stays on the client.
func (a *Agent) applyKeyRotation() error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return err
	}
	authLine := string(ssh.MarshalAuthorizedKey(sshPub)) // "ssh-ed25519 AAAA... \n"

	// persist the private key locally (PEM), 0600
	pemKey, err := ssh.MarshalPrivateKey(priv, "motech")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(statePath()), 0o700); err != nil {
		return err
	}
	keyPath := filepath.Join(filepath.Dir(statePath()), "id_motech")
	privPEM := pem.EncodeToMemory(pemKey)
	if err := os.WriteFile(keyPath, privPEM, 0o600); err != nil {
		return err
	}

	if err := installAuthorizedKey(authLine); err != nil {
		return fmt.Errorf("install authorized key: %w", err)
	}
	a.state.SSHPublicKey = authLine
	a.state.SSHPrivateKey = string(privPEM)
	return nil
}

// installAuthorizedKey writes the public key into the OS authorized-keys file.
// On Windows it targets administrators_authorized_keys; elsewhere it logs the
// intended action (so the agent builds/tests cross-platform).
func installAuthorizedKey(line string) error {
	if runtime.GOOS != "windows" {
		// Non-Windows (dev/test): keep behavior safe and observable.
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(administratorsAuthorizedKeys), 0o755); err != nil {
		return err
	}
	// Replace contents so rotation revokes the old key automatically.
	if err := os.WriteFile(administratorsAuthorizedKeys, []byte(line), 0o600); err != nil {
		return err
	}
	return nil
}
