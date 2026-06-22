package pilink_test

import (
	"bufio"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"harness-workspace/services/harness/internal/delivery/pilink"
)

// stubHandler records the inbound requests and returns canned replies.
type stubHandler struct {
	gotModels pilink.ModelsReq
	gotAgent  pilink.ResolveAgentReq
}

func (s *stubHandler) Hello(_ context.Context, req pilink.HelloReq) (pilink.ReadyResp, error) {
	return pilink.ReadyResp{SchemaVersion: 42, DBPath: "x", PID: 1}, nil
}

func (s *stubHandler) SyncModels(_ context.Context, req pilink.ModelsReq) (pilink.ModelsSyncedResp, error) {
	s.gotModels = req
	return pilink.ModelsSyncedResp{Added: len(req.Models), Total: len(req.Models)}, nil
}

func (s *stubHandler) ResolveAgent(_ context.Context, req pilink.ResolveAgentReq) (pilink.AgentResolvedResp, error) {
	s.gotAgent = req
	return pilink.AgentResolvedResp{Name: req.Agent, Persona: "weigh and plan"}, nil
}

// decodeLines reads NDJSON frames from out into generic maps.
func decodeLines(t *testing.T, out string) []map[string]any {
	t.Helper()
	var frames []map[string]any
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("decode %q: %v", sc.Text(), err)
		}
		frames = append(frames, m)
	}
	return frames
}

func TestServeRoutesModels(t *testing.T) {
	h := &stubHandler{}
	in := strings.NewReader(
		`{"type":"hello","id":"h1","cwd":"/p"}` + "\n" +
			`{"type":"models","id":"m1","client":"pi","models":[{"provider":"anthropic","slug":"claude-opus-4-8"},{"provider":"openai","slug":"gpt-5.5"}]}` + "\n",
	)
	var out strings.Builder
	if err := pilink.Serve(context.Background(), in, &out, h, nil); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	frames := decodeLines(t, out.String())
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2: %v", len(frames), frames)
	}

	if frames[0]["type"] != "ready" || frames[0]["id"] != "h1" {
		t.Fatalf("frame 0 = %v, want ready/h1", frames[0])
	}

	ms := frames[1]
	if ms["type"] != "models_synced" || ms["id"] != "m1" {
		t.Fatalf("frame 1 = %v, want models_synced/m1", ms)
	}
	if ms["added"] != float64(2) || ms["total"] != float64(2) {
		t.Fatalf("models_synced counts = %v, want added=2 total=2", ms)
	}

	// The handler received the parsed refs.
	if len(h.gotModels.Models) != 2 || h.gotModels.Client != "pi" {
		t.Fatalf("handler got %+v, want 2 models from client pi", h.gotModels)
	}
	if h.gotModels.Models[0].Provider != "anthropic" || h.gotModels.Models[0].Slug != "claude-opus-4-8" {
		t.Fatalf("first ref = %+v", h.gotModels.Models[0])
	}
}

func TestServeRoutesResolveAgent(t *testing.T) {
	h := &stubHandler{}
	in := strings.NewReader(
		`{"type":"resolve_agent","id":"a1","agent":"council","client":"pi"}` + "\n",
	)
	var out strings.Builder
	if err := pilink.Serve(context.Background(), in, &out, h, nil); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	frames := decodeLines(t, out.String())
	if len(frames) != 1 {
		t.Fatalf("got %d frames, want 1: %v", len(frames), frames)
	}
	ar := frames[0]
	if ar["type"] != "agent_resolved" || ar["id"] != "a1" {
		t.Fatalf("frame = %v, want agent_resolved/a1", ar)
	}
	if ar["name"] != "council" || ar["persona"] != "weigh and plan" {
		t.Fatalf("agent_resolved = %v, want council persona", ar)
	}
	if h.gotAgent.Agent != "council" || h.gotAgent.Client != "pi" {
		t.Fatalf("handler got %+v, want agent=council client=pi", h.gotAgent)
	}
}

func TestServeUnknownType(t *testing.T) {
	in := strings.NewReader(`{"type":"nope","id":"x"}` + "\n")
	var out strings.Builder
	if err := pilink.Serve(context.Background(), in, &out, &stubHandler{}, nil); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	frames := decodeLines(t, out.String())
	if len(frames) != 1 || frames[0]["type"] != "error" || frames[0]["id"] != "x" {
		t.Fatalf("got %v, want one error frame with id x", frames)
	}
}
