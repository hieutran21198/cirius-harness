package tool_test

import (
	"errors"
	"testing"

	"harness-workspace/services/harness/internal/domain/tool"
)

func TestNew(t *testing.T) {
	tl, err := tool.New("1", tool.NameRead, "read a file")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if tl.ID != "1" || tl.Name != tool.NameRead {
		t.Fatalf("New = %+v, want id=1 name=read", tl)
	}
}

func TestNewInvalid(t *testing.T) {
	cases := map[string]struct {
		id   string
		name tool.Name
	}{
		"empty id": {"", tool.NameRead},
		"bad name": {"1", tool.Name("nope")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := tool.New(c.id, c.name, ""); !errors.Is(err, tool.ErrInvalidTool) {
				t.Fatalf("New err = %v, want ErrInvalidTool", err)
			}
		})
	}
}
