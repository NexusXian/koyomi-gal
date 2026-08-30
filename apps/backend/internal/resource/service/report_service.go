package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend/internal/resource/dto"
	"backend/internal/resource/model"
	"backend/internal/resource/repository"
	"backend/pkg/logger"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	ErrReportNotFound      = errors.New("resource report not found")
	ErrAlreadyReported     = errors.New("resource already reported by user")
	ErrInvalidReportReason = errors.New("invalid resource report reason")
	ErrInvalidReportStatus = errors.New("invalid resource report status")
	ErrInvalidReportHandle = errors.New("resource report handle status must be resolved or rejected")
)

// ReportService covers the resource report workflow: users report published
// resources once, and admins with resource_report:list / resource_report:handle
// review and close reports. Handling a report only marks it; it never mutates
// the resource itself.
type ReportService struct {
	reports   *repository.ReportRepository
	resources *repository.ResourceRepository
}

func NewReportService(
	reports *repository.ReportRepository,
	resources *repository.ResourceRepository,
) *ReportService {
	return &ReportService{reports: reports, resources: resources}
}

// Create files one report per user per resource; duplicates fail with
// ErrAlreadyReported via the resource_reports_unique index.
func (s *ReportService) Create(
	ctx context.Context,
	userID, resourceID uint,
	req *dto.CreateResourceReportRequest,
) (*model.ResourceReport, error) {
	if !validReportReason(req.Reason) {
		return nil, ErrInvalidReportReason
	}
	resource, err := s.resources.FindPublishedByID(ctx, resourceID)
	if err != nil {
		logger.Error("find resource by id", zap.Uint("resource_id", resourceID), zap.Error(err))
		return nil, err
	}
	if resource == nil {
		return nil, ErrResourceNotFound
	}

	report := &model.ResourceReport{
		ResourceID:  resourceID,
		UserID:      userID,
		Reason:      req.Reason,
		Description: strings.TrimSpace(req.Description),
	}
	if err := s.reports.Create(ctx, report); err != nil {
		if hasConstraint(err, "resource_reports_unique") {
			return nil, ErrAlreadyReported
		}
		logger.Error("create resource report",
			zap.Uint("resource_id", resourceID), zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	return s.reports.FindByID(ctx, report.ID)
}

// List returns reports with the reported resource preloaded; an optional
// status filter narrows pending / resolved / rejected.
func (s *ReportService) List(
	ctx context.Context,
	status *int16,
	page, limit int,
) ([]model.ResourceReport, int64, int, int, error) {
	if status != nil && !validReportStatus(*status) {
		return nil, 0, page, limit, ErrInvalidReportStatus
	}
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	reports, total, err := s.reports.List(ctx, repository.ReportListOptions{
		Status: status,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		logger.Error("list resource reports", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return reports, total, page, limit, nil
}

// Handle closes a report as resolved or rejected, stamping the acting admin.
func (s *ReportService) Handle(
	ctx context.Context,
	adminID, id uint,
	req *dto.HandleResourceReportRequest,
) (*model.ResourceReport, error) {
	if req.Status != model.ReportStatusResolved && req.Status != model.ReportStatusRejected {
		return nil, ErrInvalidReportHandle
	}
	report, err := s.reports.FindByID(ctx, id)
	if err != nil {
		logger.Error("find resource report by id", zap.Uint("report_id", id), zap.Error(err))
		return nil, err
	}
	if report == nil {
		return nil, ErrReportNotFound
	}

	report.Status = req.Status
	report.HandledBy = &adminID
	handledAt := time.Now()
	report.HandledAt = &handledAt
	if err := s.reports.Update(ctx, report); err != nil {
		logger.Error("update resource report", zap.Uint("report_id", id), zap.Error(err))
		return nil, err
	}
	return s.reports.FindByID(ctx, id)
}

func validReportReason(value int16) bool {
	return value >= model.ReportReasonInvalidLink && value <= model.ReportReasonOther
}

func validReportStatus(value int16) bool {
	return value >= model.ReportStatusPending && value <= model.ReportStatusRejected
}

func hasConstraint(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == constraint
}
