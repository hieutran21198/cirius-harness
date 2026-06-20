package session_test

import (
	"errors"
	"testing"
	"time"

	"harness-workspace/services/harness/internal/domain/session"
)

func TestNew(t *testing.T) {
	created := time.Unix(1_700_000_000, 0).UTC()
	s, err := session.New("1", "p1", "fix the bug", created)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Status != session.StatusPending {
		t.Fatalf("New status = %q, want pending", s.Status)
	}
	if s.EnvType != session.EnvUnset || s.EnvID != "" {
		t.Fatalf("New env = %q/%q, want unset/empty", s.EnvType, s.EnvID)
	}
	if !s.CreatedAt.Equal(created) {
		t.Fatalf("New createdAt = %v, want %v", s.CreatedAt, created)
	}
}

func TestNewInvalid(t *testing.T) {
	cases := map[string]struct{ id, projectID string }{
		"empty id":         {"", "p1"},
		"empty project id": {"1", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := session.New(c.id, c.projectID, "", time.Time{}); !errors.Is(err, session.ErrInvalidSession) {
				t.Fatalf("New err = %v, want ErrInvalidSession", err)
			}
		})
	}
}

func TestNewMember(t *testing.T) {
	m, err := session.NewMember("1", "a1", "m1")
	if err != nil {
		t.Fatalf("NewMember: %v", err)
	}
	if m.AgentID != "a1" || m.ModelID != "m1" {
		t.Fatalf("NewMember = %+v, want agent=a1 model=m1", m)
	}

	cases := map[string]struct{ id, agentID string }{
		"empty id":       {"", "a1"},
		"empty agent id": {"1", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := session.NewMember(c.id, c.agentID, ""); !errors.Is(err, session.ErrInvalidSession) {
				t.Fatalf("NewMember err = %v, want ErrInvalidSession", err)
			}
		})
	}
}
