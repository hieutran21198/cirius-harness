package domain

import (
	"errors"
	"testing"
)

func TestNewTool(t *testing.T) {
	t.Parallel()
	tl, err := NewTool(ToolRead, "read a file")
	if err != nil {
		t.Fatalf("NewTool: %v", err)
	}
	if tl.id == "" || tl.name != ToolRead {
		t.Fatalf("NewTool = %+v", tl)
	}
}

func TestNewToolInvalid(t *testing.T) {
	t.Parallel()
	if _, err := NewTool(ToolName("bogus"), ""); !errors.Is(err, ErrInvalidTool) {
		t.Fatalf("NewTool with bad name err = %v, want ErrInvalidTool", err)
	}
}

func TestRehydrateToolRejectsEmptyID(t *testing.T) {
	t.Parallel()
	if _, err := RehydrateTool("", ToolRead, ""); !errors.Is(err, ErrInvalidTool) {
		t.Fatalf("RehydrateTool with empty id err = %v, want ErrInvalidTool", err)
	}
}
