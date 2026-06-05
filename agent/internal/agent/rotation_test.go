package agent

import "testing"

// TestRotationStateMachine verifies the backend-owned rotation: the agent
// installs the PUSHED public key, enters confirm-pending, is idempotent while
// the same key is pending, and clears confirm-pending on server ack.
//
// installAuthorizedKey + ensureSSHServer are no-ops on non-Windows, so this
// runs cleanly in CI/dev.
func TestRotationStateMachine(t *testing.T) {
	a := &Agent{state: &State{AgentToken: "t", SSHPublicKey: "ssh-ed25519 OLD motech"}}

	key1 := "ssh-ed25519 AAAANEWKEY1 motech"
	key2 := "ssh-ed25519 AAAANEWKEY2 motech"

	// 1) rotate=true + pushed key1 -> install it, enter confirm-pending.
	a.applyCommands(map[string]any{"rotate": true, "install_pubkey": key1})
	if !a.state.RotateConfirmPending {
		t.Fatal("expected RotateConfirmPending=true after rotate")
	}
	if a.state.SSHPublicKey != key1 {
		t.Fatalf("expected installed key1, got %q", a.state.SSHPublicKey)
	}

	// 2) rotate STILL true with the SAME key (server not acked) -> idempotent.
	a.applyCommands(map[string]any{"rotate": true, "install_pubkey": key1})
	if a.state.SSHPublicKey != key1 || !a.state.RotateConfirmPending {
		t.Fatal("re-pushing same key should be idempotent and stay pending")
	}

	// 3) server acks (rotate=false) -> clear confirm-pending.
	a.applyCommands(map[string]any{"rotate": false})
	if a.state.RotateConfirmPending {
		t.Fatal("RotateConfirmPending should clear after ack")
	}

	// 4) a NEW rotation with key2 -> installs the new key, pending again.
	a.applyCommands(map[string]any{"rotate": true, "install_pubkey": key2})
	if a.state.SSHPublicKey != key2 {
		t.Fatalf("expected installed key2, got %q", a.state.SSHPublicKey)
	}
	if !a.state.RotateConfirmPending {
		t.Fatal("expected confirm-pending after second rotation")
	}

	// 5) rotate=true but NO install_pubkey -> must not change anything.
	a.applyCommands(map[string]any{"rotate": false}) // ack key2 first
	a.applyCommands(map[string]any{"rotate": true})
	if a.state.SSHPublicKey != key2 {
		t.Fatal("rotate without install_pubkey must not change the installed key")
	}
}
