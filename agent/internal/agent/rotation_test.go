package agent

import "testing"

// TestRotationStateMachine verifies the confirm/ack cycle and that rotated_ok is
// only sent while a just-applied rotation awaits server acknowledgement.
func TestRotationStateMachine(t *testing.T) {
	a := &Agent{state: &State{AgentToken: "t", SSHPublicKey: "old"}}

	// 1) server says rotate=true -> we apply and enter confirm-pending.
	a.applyCommands(map[string]any{"rotate": true})
	if !a.state.RotateConfirmPending {
		t.Fatal("expected RotateConfirmPending=true after rotate")
	}
	firstKey := a.state.SSHPublicKey
	if firstKey == "old" || firstKey == "" {
		t.Fatalf("expected a freshly generated key, got %q", firstKey)
	}

	// 2) rotate STILL true (server not acked yet) -> must NOT regenerate the key.
	a.applyCommands(map[string]any{"rotate": true})
	if a.state.SSHPublicKey != firstKey {
		t.Fatal("key was regenerated while awaiting ack (should be idempotent)")
	}
	if !a.state.RotateConfirmPending {
		t.Fatal("RotateConfirmPending should remain true until ack")
	}

	// 3) server acks (rotate=false) -> clear confirm-pending.
	a.applyCommands(map[string]any{"rotate": false})
	if a.state.RotateConfirmPending {
		t.Fatal("RotateConfirmPending should be cleared after ack")
	}

	// 4) a SECOND rotation generates a NEW, different key.
	a.applyCommands(map[string]any{"rotate": true})
	if a.state.SSHPublicKey == firstKey {
		t.Fatal("second rotation should produce a different key")
	}
	if !a.state.RotateConfirmPending {
		t.Fatal("expected confirm-pending after second rotation")
	}
}
