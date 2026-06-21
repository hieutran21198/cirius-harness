package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	t.Parallel()
	s, err := NewSession("p1", "fix the bug", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if s.id == "" {
		t.Fatal("NewSession should mint a non-empty id")
	}
	if s.status != SessionPending {
		t.Fatalf("NewSession status = %q, want pending", s.status)
	}
	if s.envType != EnvUnset {
		t.Fatalf("NewSession envType = %q, want unset", s.envType)
	}
}

func TestNewSessionInvalid(t *testing.T) {
	t.Parallel()
	if _, err := NewSession("", "", time.Unix(0, 0).UTC()); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("NewSession with empty project id err = %v, want ErrInvalidSession", err)
	}
}

func TestRehydrateSessionEnvConsistency(t *testing.T) {
	t.Parallel()
	// A non-unset env type requires an env id.
	if _, err := RehydrateSession("1", "p1", EnvContainer, "", "", SessionRunning, time.Unix(0, 0).UTC(), nil, nil, nil); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("RehydrateSession err = %v, want ErrInvalidSession (env id required)", err)
	}
	// Unset env type forbids an env id.
	if _, err := RehydrateSession("1", "p1", EnvUnset, "c1", "", SessionPending, time.Unix(0, 0).UTC(), nil, nil, nil); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("RehydrateSession err = %v, want ErrInvalidSession (env id must be empty)", err)
	}
}

func TestSessionValidatesMembers(t *testing.T) {
	t.Parallel()
	bad := []Member{{id: "", agentID: "a1"}} // empty member id
	if _, err := RehydrateSession("1", "p1", EnvUnset, "", "", SessionPending, time.Unix(0, 0).UTC(), nil, nil, bad); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("RehydrateSession err = %v, want ErrInvalidSession (bad member)", err)
	}
}

func TestNewMember(t *testing.T) {
	t.Parallel()
	m, err := NewMember("a1", "m1")
	if err != nil {
		t.Fatalf("NewMember: %v", err)
	}
	if m.id == "" {
		t.Fatal("NewMember should mint a non-empty id")
	}
}

func TestNewMemberInvalid(t *testing.T) {
	t.Parallel()
	if _, err := NewMember("", "m1"); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("NewMember with empty agent id err = %v, want ErrInvalidSession", err)
	}
}
