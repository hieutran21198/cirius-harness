package query_test

import (
	"context"
	"testing"
	"time"

	"harness-workspace/services/harness/internal/app/query"
	"harness-workspace/services/harness/internal/domain"
)

// fakeRunByPlan is a domain.PlanRunReader returning one canned run (or not-found).
type fakeRunByPlan struct {
	run   domain.PlanRun
	found bool
}

func (r fakeRunByPlan) FindByPlan(_ context.Context, _ domain.PlanID) (domain.PlanRun, error) {
	if !r.found {
		return domain.PlanRun{}, domain.ErrPlanRunNotFound
	}
	return r.run, nil
}

// fakeReportsByRun is a domain.TaskReportReader returning canned reports.
type fakeReportsByRun struct{ reports []domain.TaskReport }

func (r fakeReportsByRun) FindByPlanRun(_ context.Context, _ domain.PlanRunID) ([]domain.TaskReport, error) {
	return r.reports, nil
}

func TestGetReportsReturnsNormalizedEnvelopes(t *testing.T) {
	now := time.Now()
	run, err := domain.NewPlanRun("plan-1", "sess-1", []string{"T1"}, now)
	if err != nil {
		t.Fatalf("NewPlanRun: %v", err)
	}
	runID := run.Snapshot().ID
	report, err := domain.NewTaskReport(runID, "T1", "explorer",
		domain.TaskReportEnvelope{Status: "done", Summary: "scanned", Confidence: "high"}, "raw", now)
	if err != nil {
		t.Fatalf("NewTaskReport: %v", err)
	}
	rs := fakeReadStore{rr: fakeRunByPlan{run: run, found: true}, trr: fakeReportsByRun{reports: []domain.TaskReport{report}}}
	h := query.NewGetReportsHandler(rs, discardLogger())

	res, err := h.Handle(context.Background(), query.GetReports{PlanID: "plan-1"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if res.PlanRunID != runID {
		t.Fatalf("run id = %q, want %q", res.PlanRunID, runID)
	}
	if len(res.Reports) != 1 || res.Reports[0].TaskRef != "T1" || res.Reports[0].Envelope.Summary != "scanned" {
		t.Fatalf("reports = %+v, want one T1 envelope", res.Reports)
	}
}

func TestGetReportsNoRunIsEmpty(t *testing.T) {
	rs := fakeReadStore{rr: fakeRunByPlan{found: false}}
	h := query.NewGetReportsHandler(rs, discardLogger())
	res, err := h.Handle(context.Background(), query.GetReports{PlanID: "plan-1"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(res.Reports) != 0 || res.PlanRunID != "" {
		t.Fatalf("res = %+v, want empty", res)
	}
}
