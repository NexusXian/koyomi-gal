package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	notificationModel "backend/internal/notification/model"
	notificationService "backend/internal/notification/service"
	rbacService "backend/internal/rbac/service"
	relationModel "backend/internal/relation/model"
	"backend/internal/resource/dto"
	resourceModel "backend/internal/resource/model"
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
	reports       *repository.ReportRepository
	resources     *repository.ResourceRepository
	rbac          *rbacService.RBACService
	notifications *notificationService.NotificationService
}

func (s *ReportService) SetNotificationDependencies(
	rbac *rbacService.RBACService,
	notifications *notificationService.NotificationService,
) {
	s.rbac = rbac
	s.notifications = notifications
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
) (*resourceModel.ResourceReport, error) {
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

	report := &resourceModel.ResourceReport{
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
	created, err := s.reports.FindByID(ctx, report.ID)
	if err != nil {
		return nil, err
	}
	s.notifyResourceReported(ctx, userID, created)
	return created, nil
}

// List returns reports with the reported resource preloaded; an optional
// status filter narrows pending / resolved / rejected.
func (s *ReportService) List(
	ctx context.Context,
	status *int16,
	page, limit int,
) ([]resourceModel.ResourceReport, int64, int, int, error) {
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
) (*resourceModel.ResourceReport, error) {
	if req.Status != resourceModel.ReportStatusResolved && req.Status != resourceModel.ReportStatusRejected {
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
	if report.Status == req.Status {
		return report, nil
	}

	report.Status = req.Status
	report.HandledBy = &adminID
	handledAt := time.Now()
	report.HandledAt = &handledAt
	if err := s.reports.Update(ctx, report); err != nil {
		logger.Error("update resource report", zap.Uint("report_id", id), zap.Error(err))
		return nil, err
	}
	handled, err := s.reports.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.notifyReportHandled(ctx, adminID, handled)
	return handled, nil
}

func (s *ReportService) notifyResourceReported(
	ctx context.Context,
	actorID uint,
	report *resourceModel.ResourceReport,
) {
	if s.rbac == nil || s.notifications == nil || report.Resource == nil {
		return
	}
	recipientIDs, err := s.rbac.FindUserIDsByPermission(ctx, "resource_report:handle")
	if err != nil {
		logger.Error("find resource report handlers for notification", zap.Uint("report_id", report.ID), zap.Error(err))
		return
	}
	inputs := make([]notificationService.CreateInput, 0, len(recipientIDs))
	for _, recipientID := range recipientIDs {
		inputs = append(inputs, notificationService.CreateInput{
			RecipientID: recipientID,
			ActorID:     &actorID,
			Category:    notificationModel.CategoryModeration,
			Type:        notificationModel.TypeResourceReported,
			EntityType:  "resource_report",
			EntityID:    report.ID,
			Title:       "新的资源举报",
			Content:     fmt.Sprintf("举报了资源「%s」", report.Resource.Title),
			TargetURL:   "/admin/reports",
			Metadata: map[string]any{
				"description": report.Description,
				"target_type": report.Resource.TargetType,
				"target_id":   report.Resource.TargetID,
				"reason":      report.Reason,
				"resource_id": report.ResourceID,
			},
		})
	}
	if _, err := s.notifications.CreateMany(ctx, inputs); err != nil {
		logger.Error("create resource report notifications", zap.Uint("report_id", report.ID), zap.Error(err))
	}
}

func (s *ReportService) notifyReportHandled(ctx context.Context, actorID uint, report *resourceModel.ResourceReport) {
	if s.notifications == nil || report.Resource == nil {
		return
	}
	notificationType := notificationModel.TypeReportResolved
	title := "资源举报已处理"
	content := "你提交的资源举报已处理"
	if report.Status == resourceModel.ReportStatusRejected {
		notificationType = notificationModel.TypeReportRejected
		title = "资源举报未被采纳"
		content = "你提交的资源举报未被采纳"
	}
	if _, err := s.notifications.Create(ctx, notificationService.CreateInput{
		RecipientID: report.UserID,
		ActorID:     &actorID,
		Category:    notificationModel.CategoryModeration,
		Type:        notificationType,
		EntityType:  "resource_report",
		EntityID:    report.ID,
		Title:       title,
		Content:     content,
		TargetURL:   resourceReportTargetURL(report.Resource),
		Metadata: map[string]any{
			"description": report.Description,
			"target_type": report.Resource.TargetType,
			"target_id":   report.Resource.TargetID,
			"reason":      report.Reason,
			"resource_id": report.ResourceID,
			"status":      report.Status,
		},
	}); err != nil {
		logger.Error("create resource report result notification", zap.Uint("report_id", report.ID), zap.Error(err))
	}
}

func resourceReportTargetURL(resource *resourceModel.Resource) string {
	if resource.TargetType == relationModel.WorkTypeNovel {
		return fmt.Sprintf("/novels/%d", resource.TargetID)
	}
	return fmt.Sprintf("/galgames/%d", resource.TargetID)
}

func validReportReason(value int16) bool {
	return value >= resourceModel.ReportReasonInvalidLink && value <= resourceModel.ReportReasonOther
}

func validReportStatus(value int16) bool {
	return value >= resourceModel.ReportStatusPending && value <= resourceModel.ReportStatusRejected
}

func hasConstraint(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == constraint
}
