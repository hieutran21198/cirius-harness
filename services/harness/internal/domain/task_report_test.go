package domain

import (
	"errors"
	"testing"
	"time"
)

func validEnvelope() TaskReportEnvelope {
	return TaskReportEnvelope{Status: "done", Summary: "did the thing", Confidence: "high"}
}

func TestNewTaskReportValidates(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tests := []struct {
		name    string
		ref     string
		env     TaskReportEnvelope
		wantErr bool
	}{
		{"valid", "T1", validEnvelope(), false},
		{"missing ref", "", validEnvelope(), true},
		{"bad status", "T1", TaskReportEnvelope{Status: "weird", Summary: "x", Confidence: "high"}, true},
		{"bad confidence", "T1", TaskReportEnvelope{Status: "done", Summary: "x", Confidence: "vibes"}, true},
		{"empty summary", "T1", TaskReportEnvelope{Status: "done", Summary: "", Confidence: "low"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewTaskReport(PlanRunID("run-1"), tt.ref, "implementer", tt.env, "raw output", now)
			if tt.wantErr && !errors.Is(err, ErrInvalidTaskReport) {
				t.Fatalf("want ErrInvalidTaskReport, got %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewPlanDecisionValidates(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tests := []struct {
		name    string
		src     CouncilDecision
		wantErr bool
	}{
		{"valid", CouncilDecision{Verdict: "accept", Summary: "all good", Tasks: []TaskVerdict{{Ref: "T1", Verdict: "accept"}}}, false},
		{"bad verdict", CouncilDecision{Verdict: "meh", Summary: "x"}, true},
		{"empty summary", CouncilDecision{Verdict: "accept", Summary: ""}, true},
		{"task missing ref", CouncilDecision{Verdict: "accept", Summary: "x", Tasks: []TaskVerdict{{Ref: "", Verdict: "accept"}}}, true},
		{"task bad verdict", CouncilDecision{Verdict: "accept", Summary: "x", Tasks: []TaskVerdict{{Ref: "T1", Verdict: "nope"}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewPlanDecision(PlanRunID("run-1"), tt.src, now)
			if tt.wantErr && !errors.Is(err, ErrInvalidPlanDecision) {
				t.Fatalf("want ErrInvalidPlanDecision, got %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
