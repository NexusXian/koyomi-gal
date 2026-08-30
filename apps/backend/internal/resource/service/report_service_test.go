package service

import (
	"context"
	"errors"
	"testing"

	"backend/internal/resource/dto"
	"backend/internal/resource/model"
	"backend/internal/testutil"
)

func TestResourceReportLifecycle(t *testing.T) {
	env := newResourceTestEnv(t)
	ctx := context.Background()
	uploader := testutil.CreateUser(t, env.db, "report-uploader")
	reporter := testutil.CreateUser(t, env.db, "report-reporter")
	otherReporter := testutil.CreateUser(t, env.db, "report-other-reporter")
	admin := testutil.CreateUser(t, env.db, "report-admin")
	galgameID := env.createPublishedGalgame(t, uploader, "report-game")
	published := env.createResource(t, uploader, galgameID, "reported resource",
		model.ResourceStatusPublished, "https://example.com/reported")

	report, err := env.reports.Create(ctx, reporter, published.ID, &dto.CreateResourceReportRequest{
		Reason:      model.ReportReasonInvalidLink,
		Description: " link broken ",
	})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	if report.Status != model.ReportStatusPending || report.Description != "link broken" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.HandledBy != nil || report.HandledAt != nil {
		t.Fatalf("pending report must not have handle stamps: %+v", report)
	}

	if _, err := env.reports.Create(ctx, reporter, published.ID, &dto.CreateResourceReportRequest{
		Reason: model.ReportReasonOther,
	}); !errors.Is(err, ErrAlreadyReported) {
		t.Fatalf("expected ErrAlreadyReported, got %v", err)
	}
	second, err := env.reports.Create(ctx, otherReporter, published.ID, &dto.CreateResourceReportRequest{
		Reason: model.ReportReasonMalware,
	})
	if err != nil || second.ID == report.ID {
		t.Fatalf("expected distinct report from another user: %+v err=%v", second, err)
	}

	if _, err := env.reports.Create(ctx, reporter, published.ID, &dto.CreateResourceReportRequest{
		Reason: 7,
	}); !errors.Is(err, ErrInvalidReportReason) {
		t.Fatalf("expected ErrInvalidReportReason, got %v", err)
	}
	if _, err := env.reports.Create(ctx, reporter, 999999, &dto.CreateResourceReportRequest{
		Reason: model.ReportReasonOther,
	}); !errors.Is(err, ErrResourceNotFound) {
		t.Fatalf("expected ErrResourceNotFound, got %v", err)
	}

	reports, total, _, _, err := env.reports.List(ctx, nil, 0, 0)
	if err != nil || total != 2 || len(reports) != 2 {
		t.Fatalf("expected 2 reports, got total=%d err=%v", total, err)
	}
	if reports[0].Resource == nil || reports[0].Resource.Title != "reported resource" {
		t.Fatalf("expected reported resource preloaded, got %+v", reports[0].Resource)
	}

	handled, err := env.reports.Handle(ctx, admin, second.ID, &dto.HandleResourceReportRequest{
		Status: model.ReportStatusResolved,
	})
	if err != nil {
		t.Fatalf("handle report: %v", err)
	}
	if handled.Status != model.ReportStatusResolved ||
		handled.HandledBy == nil || *handled.HandledBy != admin || handled.HandledAt == nil {
		t.Fatalf("unexpected handled report: %+v", handled)
	}

	pendingStatus := model.ReportStatusPending
	resolvedStatus := model.ReportStatusResolved
	reports, total, _, _, err = env.reports.List(ctx, &pendingStatus, 0, 0)
	if err != nil || total != 1 || reports[0].ID != report.ID {
		t.Fatalf("expected 1 pending report, got total=%d err=%v", total, err)
	}
	reports, total, _, _, err = env.reports.List(ctx, &resolvedStatus, 0, 0)
	if err != nil || total != 1 || reports[0].ID != second.ID {
		t.Fatalf("expected 1 resolved report, got total=%d err=%v", total, err)
	}

	rejected, err := env.reports.Handle(ctx, admin, report.ID, &dto.HandleResourceReportRequest{
		Status: model.ReportStatusRejected,
	})
	if err != nil || rejected.Status != model.ReportStatusRejected {
		t.Fatalf("expected rejected report, got %+v err=%v", rejected, err)
	}

	if _, err := env.reports.Handle(ctx, admin, report.ID, &dto.HandleResourceReportRequest{
		Status: 0,
	}); !errors.Is(err, ErrInvalidReportHandle) {
		t.Fatalf("expected ErrInvalidReportHandle, got %v", err)
	}
	if _, err := env.reports.Handle(ctx, admin, 999999, &dto.HandleResourceReportRequest{
		Status: model.ReportStatusResolved,
	}); !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("expected ErrReportNotFound, got %v", err)
	}
	invalidStatus := int16(5)
	if _, _, _, _, err := env.reports.List(ctx, &invalidStatus, 0, 0); !errors.Is(err, ErrInvalidReportStatus) {
		t.Fatalf("expected ErrInvalidReportStatus, got %v", err)
	}
}

func TestResourceReportOnlyPublishedResources(t *testing.T) {
	env := newResourceTestEnv(t)
	ctx := context.Background()
	uploader := testutil.CreateUser(t, env.db, "report-pending-uploader")
	reporter := testutil.CreateUser(t, env.db, "report-pending-reporter")
	galgameID := env.createPublishedGalgame(t, uploader, "report-pending-game")
	pending := env.createResource(t, uploader, galgameID, "pending reported",
		model.ResourceStatusPending, "https://example.com/pending")

	if _, err := env.reports.Create(ctx, reporter, pending.ID, &dto.CreateResourceReportRequest{
		Reason: model.ReportReasonOther,
	}); !errors.Is(err, ErrResourceNotFound) {
		t.Fatalf("unpublished resources must not be reportable, got %v", err)
	}
}
