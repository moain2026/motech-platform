package agent

import (
	"errors"
	"testing"
	"time"
)

func TestTokenManager_ForceReloadDetectsChange(t *testing.T) {
	disk := "OLD"
	m := NewTokenManager("OLD")
	m.readState = func() (*State, error) { return &State{AgentToken: disk}, nil }

	// unchanged
	if tok, changed := m.ForceReload(); changed || tok != "OLD" {
		t.Fatalf("expected unchanged OLD, got tok=%q changed=%v", tok, changed)
	}
	// disk updates -> reload should detect change
	disk = "NEW"
	if tok, changed := m.ForceReload(); !changed || tok != "NEW" {
		t.Fatalf("expected changed NEW, got tok=%q changed=%v", tok, changed)
	}
}

func TestTokenManager_GetTTLReread(t *testing.T) {
	disk := "T1"
	m := NewTokenManager("T1")
	m.ttl = 10 * time.Millisecond
	m.readState = func() (*State, error) { return &State{AgentToken: disk}, nil }

	if got := m.Get(); got != "T1" {
		t.Fatalf("want T1 got %q", got)
	}
	disk = "T2"
	time.Sleep(15 * time.Millisecond) // expire TTL
	if got := m.Get(); got != "T2" {
		t.Fatalf("after TTL expiry want T2 got %q", got)
	}
}

func TestTokenManager_ReloadKeepsOldOnError(t *testing.T) {
	m := NewTokenManager("KEEP")
	m.readState = func() (*State, error) { return nil, errors.New("locked") }
	if tok, changed := m.ForceReload(); changed || tok != "KEEP" {
		t.Fatalf("on read error expected KEEP/unchanged, got tok=%q changed=%v", tok, changed)
	}
}

func TestTokenManager_RetriesOnTransientEmpty(t *testing.T) {
	calls := 0
	m := NewTokenManager("")
	m.readState = func() (*State, error) {
		calls++
		if calls < 2 {
			return &State{AgentToken: ""}, nil // transient empty (file mid-write)
		}
		return &State{AgentToken: "FINAL"}, nil
	}
	if tok, _ := m.ForceReload(); tok != "FINAL" {
		t.Fatalf("expected FINAL after retry, got %q (calls=%d)", tok, calls)
	}
	if calls < 2 {
		t.Fatalf("expected >=2 read attempts, got %d", calls)
	}
}
