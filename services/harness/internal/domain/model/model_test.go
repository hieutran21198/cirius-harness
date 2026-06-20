package model_test

import (
	"errors"
	"testing"

	"harness-workspace/services/harness/internal/domain/model"
)

func TestNew(t *testing.T) {
	m, err := model.New("1", "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !m.Enabled {
		t.Fatal("New should enable the model by default")
	}
	if m.ID != "1" || m.Ref() != "openai/gpt-5.5" {
		t.Fatalf("New = %+v, want id=1 ref=openai/gpt-5.5", m)
	}
}

func TestNewInvalid(t *testing.T) {
	cases := map[string]struct{ id, provider, slug string }{
		"empty id":       {"", "openai", "gpt-5.5"},
		"empty provider": {"1", "", "gpt-5.5"},
		"empty slug":     {"1", "openai", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := model.New(c.id, c.provider, c.slug); !errors.Is(err, model.ErrInvalidModel) {
				t.Fatalf("New(%q,%q,%q) err = %v, want ErrInvalidModel", c.id, c.provider, c.slug, err)
			}
		})
	}
}
