package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	contributionModel "backend/internal/contribution/model"
	contributionService "backend/internal/contribution/service"
	notificationModel "backend/internal/notification/model"
	notificationService "backend/internal/notification/service"
	"backend/internal/novel/dto"
	"backend/internal/novel/model"
	"backend/internal/novel/repository"
	rbacService "backend/internal/rbac/service"
	relationModel "backend/internal/relation/model"
	"backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrVolumeNotFound       = errors.New("novel volume not found")
	ErrInvalidVolumeInput   = errors.New("invalid novel volume input")
	ErrInvalidVolumeURL     = errors.New("invalid novel volume url")
	ErrInvalidISBN          = errors.New("invalid isbn")
	ErrInvalidVolumeStatus  = errors.New("invalid novel volume status")
	ErrInvalidVolumeReorder = errors.New("invalid novel volume reorder")
)

type VolumeService struct {
	volumes       *repository.VolumeRepository
	novels        *repository.NovelRepository
	contributions *contributionService.ContributionService
	rbac          *rbacService.RBACService
	notifications *notificationService.NotificationService
}

func NewVolumeService(
	volumes *repository.VolumeRepository,
	novels *repository.NovelRepository,
) *VolumeService {
	return &VolumeService{volumes: volumes, novels: novels}
}

func (s *VolumeService) SetContributionService(contributions *contributionService.ContributionService) {
	s.contributions = contributions
}

func (s *VolumeService) SetNotificationDependencies(
	rbac *rbacService.RBACService,
	notifications *notificationService.NotificationService,
) {
	s.rbac = rbac
	s.notifications = notifications
}

func (s *VolumeService) CreateVolume(
	ctx context.Context,
	userID, novelID uint,
	req *dto.CreateVolumeRequest,
) (*model.NovelVolume, error) {
	if err := s.ensureNovelExists(ctx, novelID); err != nil {
		return nil, err
	}
	volume, err := buildVolumeFromCreate(novelID, userID, req)
	if err != nil {
		return nil, err
	}
	maxOrder, err := s.volumes.MaxSortOrder(ctx, novelID)
	if err != nil {
		logger.Error("max volume sort order", zap.Uint("novel_id", novelID), zap.Error(err))
		return nil, err
	}
	volume.SortOrder = maxOrder + 1

	write := func(tx *repository.VolumeRepository, db *gorm.DB) error {
		if err := tx.Create(ctx, volume); err != nil {
			return err
		}
		if volume.Status == model.NovelStatusPublished && s.contributions != nil {
			sourceType := contributionModel.ContributionSourceNovelVolume
			sourceID := volume.ID
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeNovel,
				TargetID:   novelID,
				UserID:     userID,
				Action:     contributionModel.ContributionActionAddVolume,
				SourceType: &sourceType,
				SourceID:   &sourceID,
			}, db)
		}
		return nil
	}
	var err2 error
	if s.contributions != nil {
		err2 = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewVolumeRepository(db), db)
		})
	} else {
		err2 = s.volumes.Transaction(ctx, func(tx *repository.VolumeRepository) error {
			return write(tx, nil)
		})
	}
	if err2 != nil {
		logger.Error("create volume", zap.Uint("novel_id", novelID), zap.Error(err2))
		return nil, err2
	}
	created, err := s.volumes.FindByID(ctx, volume.ID)
	if err != nil {
		return nil, err
	}
	if created.Status == model.NovelStatusPending {
		s.notifyVolumeSubmitted(ctx, userID, novelID, created)
	}
	return created, nil
}

func (s *VolumeService) UpdateVolume(
	ctx context.Context,
	actorID, novelID, volumeID uint,
	req *dto.UpdateVolumeRequest,
) (*model.NovelVolume, error) {
	volume, err := s.volumes.FindByIDAndNovel(ctx, novelID, volumeID)
	if err != nil {
		logger.Error("find volume by id and novel", zap.Uint("volume_id", volumeID), zap.Uint("novel_id", novelID), zap.Error(err))
		return nil, err
	}
	if volume == nil {
		return nil, ErrVolumeNotFound
	}
	if !validVolumeStatus(*req.Status) {
		return nil, ErrInvalidVolumeStatus
	}
	releaseDate, err := parseReleaseDate(req.ReleaseDate)
	if err != nil {
		return nil, err
	}
	if !validISBN(req.ISBN) {
		return nil, ErrInvalidISBN
	}
	if !validHTTPURL(req.CoverURL) {
		return nil, ErrInvalidVolumeURL
	}

	oldStatus := volume.Status
	changed := volumeChanged(volume, req, releaseDate)
	volume.VolumeNumber = req.VolumeNumber
	volume.Title = strings.TrimSpace(req.Title)
	volume.OriginalTitle = strings.TrimSpace(req.OriginalTitle)
	volume.CoverURL = strings.TrimSpace(req.CoverURL)
	volume.ISBN = strings.TrimSpace(req.ISBN)
	volume.ReleaseDate = releaseDate
	volume.Summary = strings.TrimSpace(req.Summary)
	volume.Status = *req.Status

	write := func(tx *repository.VolumeRepository, db *gorm.DB) error {
		if err := tx.Update(ctx, volume); err != nil {
			return err
		}
		if changed && volume.Status == model.NovelStatusPublished && s.contributions != nil {
			input := contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeNovel,
				TargetID:   novelID,
				UserID:     actorID,
				Action:     contributionModel.ContributionActionUpdateVolume,
			}
			if oldStatus != model.NovelStatusPublished {
				input.Action = contributionModel.ContributionActionAddVolume
				sourceType := contributionModel.ContributionSourceNovelVolume
				sourceID := volume.ID
				input.SourceType = &sourceType
				input.SourceID = &sourceID
			}
			return s.contributions.RecordContribution(ctx, input, db)
		}
		return nil
	}
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			return write(repository.NewVolumeRepository(db), db)
		})
	} else {
		err = s.volumes.Transaction(ctx, func(tx *repository.VolumeRepository) error {
			return write(tx, nil)
		})
	}
	if err != nil {
		logger.Error("update volume", zap.Uint("volume_id", volumeID), zap.Uint("novel_id", novelID), zap.Error(err))
		return nil, err
	}
	return s.volumes.FindByIDAndNovel(ctx, novelID, volumeID)
}

func (s *VolumeService) DeleteVolume(ctx context.Context, novelID, volumeID uint) error {
	volume, err := s.volumes.FindByIDAndNovel(ctx, novelID, volumeID)
	if err != nil {
		logger.Error("find volume by id and novel", zap.Uint("volume_id", volumeID), zap.Uint("novel_id", novelID), zap.Error(err))
		return err
	}
	if volume == nil {
		return ErrVolumeNotFound
	}
	if err := s.volumes.Delete(ctx, volumeID); err != nil {
		logger.Error("delete volume", zap.Uint("volume_id", volumeID), zap.Error(err))
		return err
	}
	return nil
}

func (s *VolumeService) ReviewVolume(
	ctx context.Context,
	actorID, volumeID uint,
	req *dto.ReviewVolumeRequest,
) (*model.NovelVolume, error) {
	if req.Status != model.NovelStatusPublished && req.Status != model.NovelStatusRejected {
		return nil, ErrInvalidVolumeStatus
	}
	volume, err := s.volumes.FindByID(ctx, volumeID)
	if err != nil {
		logger.Error("find volume by id", zap.Uint("volume_id", volumeID), zap.Error(err))
		return nil, err
	}
	if volume == nil {
		return nil, ErrVolumeNotFound
	}
	if volume.Status == req.Status {
		return volume, nil
	}

	reason := strings.TrimSpace(req.Reason)
	now := time.Now()
	if req.Status == model.NovelStatusPublished && volume.CreatedBy != nil && s.contributions != nil {
		err = s.contributions.Transaction(ctx, func(db *gorm.DB) error {
			if err := repository.NewVolumeRepository(db).UpdateReview(ctx, volumeID, req.Status, &actorID, &now, ""); err != nil {
				return err
			}
			sourceType := contributionModel.ContributionSourceNovelVolume
			sourceID := volume.ID
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeNovel,
				TargetID:   volume.NovelID,
				UserID:     *volume.CreatedBy,
				Action:     contributionModel.ContributionActionAddVolume,
				SourceType: &sourceType,
				SourceID:   &sourceID,
			}, db)
		})
	} else {
		err = s.volumes.UpdateReview(ctx, volumeID, req.Status, &actorID, &now, reason)
	}
	if err != nil {
		logger.Error("review volume", zap.Uint("volume_id", volumeID), zap.Uint("actor_id", actorID), zap.Error(err))
		return nil, err
	}
	reviewed, err := s.volumes.FindByID(ctx, volumeID)
	if err != nil {
		return nil, err
	}
	s.notifyVolumeReviewResult(ctx, actorID, reviewed, reason)
	return reviewed, nil
}

// ListVolumes pages through the novel's volumes for the public endpoint.
func (s *VolumeService) ListVolumes(
	ctx context.Context,
	novelID uint,
	publishedOnly bool,
	page, limit int,
) ([]model.NovelVolume, int64, int, int, error) {
	if publishedOnly {
		if err := s.ensurePublishedNovel(ctx, novelID); err != nil {
			return nil, 0, page, limit, err
		}
	} else if err := s.ensureNovelExists(ctx, novelID); err != nil {
		return nil, 0, page, limit, err
	}
	page, limit = normalizePage(page, limit)
	volumes, err := s.volumes.ListByNovel(ctx, novelID, publishedOnly)
	if err != nil {
		logger.Error("list novel volumes", zap.Uint("novel_id", novelID), zap.Error(err))
		return nil, 0, page, limit, err
	}
	total := int64(len(volumes))
	start := (page - 1) * limit
	if start >= len(volumes) {
		return []model.NovelVolume{}, total, page, limit, nil
	}
	end := start + limit
	if end > len(volumes) {
		end = len(volumes)
	}
	return volumes[start:end], total, page, limit, nil
}

func (s *VolumeService) GetVolume(
	ctx context.Context,
	novelID, volumeID uint,
	publishedOnly bool,
) (*model.NovelVolume, error) {
	if publishedOnly {
		if err := s.ensurePublishedNovel(ctx, novelID); err != nil {
			return nil, err
		}
	}
	volume, err := s.volumes.FindByIDAndNovel(ctx, novelID, volumeID)
	if err != nil {
		logger.Error("find volume by id and novel", zap.Uint("volume_id", volumeID), zap.Uint("novel_id", novelID), zap.Error(err))
		return nil, err
	}
	if volume == nil {
		return nil, ErrVolumeNotFound
	}
	if publishedOnly && volume.Status != model.NovelStatusPublished {
		return nil, ErrVolumeNotFound
	}
	return volume, nil
}

// ReorderVolumes rewrites sort_order so ids map to 0..n-1. The id set must
// exactly match the novel's volumes to avoid stale sort_order leftovers.
func (s *VolumeService) ReorderVolumes(ctx context.Context, novelID uint, req *dto.ReorderVolumesRequest) error {
	if err := s.ensureNovelExists(ctx, novelID); err != nil {
		return err
	}
	volumes, err := s.volumes.ListByNovel(ctx, novelID, false)
	if err != nil {
		logger.Error("list volumes for reorder", zap.Uint("novel_id", novelID), zap.Error(err))
		return err
	}
	if len(req.IDs) != len(volumes) {
		return ErrInvalidVolumeReorder
	}
	known := make(map[uint]struct{}, len(volumes))
	for i := range volumes {
		known[volumes[i].ID] = struct{}{}
	}
	seen := make(map[uint]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		if _, ok := known[id]; !ok {
			return ErrInvalidVolumeReorder
		}
		if _, dup := seen[id]; dup {
			return ErrInvalidVolumeReorder
		}
		seen[id] = struct{}{}
	}
	if err := s.volumes.UpdateOrder(ctx, novelID, req.IDs); err != nil {
		logger.Error("update volume order", zap.Uint("novel_id", novelID), zap.Error(err))
		return err
	}
	return nil
}

func (s *VolumeService) ListAdminVolumes(
	ctx context.Context,
	status *int16,
	novelID *uint,
	page, limit int,
) ([]model.NovelVolume, int64, int, int, error) {
	if status != nil && !validVolumeStatus(*status) {
		return nil, 0, page, limit, ErrInvalidVolumeStatus
	}
	page, limit = normalizePage(page, limit)
	volumes, total, err := s.volumes.ListAdmin(ctx, repository.VolumeListOptions{
		Status:  status,
		NovelID: novelID,
		Page:    page,
		Limit:   limit,
	})
	if err != nil {
		logger.Error("list admin volumes", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return volumes, total, page, limit, nil
}

func (s *VolumeService) ensureNovelExists(ctx context.Context, novelID uint) error {
	novel, err := s.novels.FindByID(ctx, novelID)
	if err != nil {
		logger.Error("find novel by id", zap.Uint("novel_id", novelID), zap.Error(err))
		return err
	}
	if novel == nil {
		return ErrNovelNotFound
	}
	return nil
}

func (s *VolumeService) ensurePublishedNovel(ctx context.Context, novelID uint) error {
	novel, err := s.novels.FindPublishedByID(ctx, novelID)
	if err != nil {
		logger.Error("find published novel by id", zap.Uint("novel_id", novelID), zap.Error(err))
		return err
	}
	if novel == nil {
		return ErrNovelNotFound
	}
	return nil
}

func (s *VolumeService) notifyVolumeSubmitted(
	ctx context.Context,
	actorID, novelID uint,
	volume *model.NovelVolume,
) {
	if s.rbac == nil || s.notifications == nil {
		return
	}
	novel, err := s.novels.FindByID(ctx, novelID)
	if err != nil || novel == nil {
		logger.Error("find novel for volume notification", zap.Uint("novel_id", novelID), zap.Error(err))
		return
	}
	recipientIDs, err := s.rbac.FindUserIDsByPermission(ctx, "novel:review")
	if err != nil {
		logger.Error("find novel reviewers for volume notification", zap.Uint("volume_id", volume.ID), zap.Error(err))
		return
	}
	inputs := make([]notificationService.CreateInput, 0, len(recipientIDs))
	for _, recipientID := range recipientIDs {
		inputs = append(inputs, notificationService.CreateInput{
			RecipientID: recipientID,
			ActorID:     &actorID,
			Category:    notificationModel.CategoryReview,
			Type:        notificationModel.TypeNovelVolumeSubmitted,
			EntityType:  "novel_volume",
			EntityID:    volume.ID,
			Title:       "新的小说卷册待审核",
			Content:     fmt.Sprintf("提交了小说「%s」的卷册「%s」，等待审核", novel.Title, volumeLabel(volume)),
			TargetURL:   "/admin/novels?volumes=1",
			Metadata:    map[string]any{"novel_id": novelID, "title": novel.Title},
		})
	}
	if _, err := s.notifications.CreateMany(ctx, inputs); err != nil {
		logger.Error("create volume submission notifications", zap.Uint("volume_id", volume.ID), zap.Error(err))
	}
}

func (s *VolumeService) notifyVolumeReviewResult(
	ctx context.Context,
	actorID uint,
	volume *model.NovelVolume,
	reason string,
) {
	if s.notifications == nil || volume.CreatedBy == nil {
		return
	}
	notificationType := notificationModel.TypeNovelVolumeApproved
	title := "小说卷册审核通过"
	content := fmt.Sprintf("你提交的卷册「%s」已通过审核", volumeLabel(volume))
	if volume.Status == model.NovelStatusRejected {
		notificationType = notificationModel.TypeNovelVolumeRejected
		title = "小说卷册审核未通过"
		content = fmt.Sprintf("你提交的卷册「%s」未通过审核", volumeLabel(volume))
		if reason != "" {
			content = fmt.Sprintf("你提交的卷册「%s」未通过审核：%s", volumeLabel(volume), reason)
		}
	}
	if _, err := s.notifications.Create(ctx, notificationService.CreateInput{
		RecipientID: *volume.CreatedBy,
		ActorID:     &actorID,
		Category:    notificationModel.CategoryReview,
		Type:        notificationType,
		EntityType:  "novel_volume",
		EntityID:    volume.ID,
		Title:       title,
		Content:     content,
		TargetURL:   fmt.Sprintf("/novels/%d", volume.NovelID),
		Metadata: map[string]any{
			"novel_id": volume.NovelID,
			"status":   volume.Status,
			"reason":   reason,
		},
	}); err != nil {
		logger.Error("create volume review notification", zap.Uint("volume_id", volume.ID), zap.Error(err))
	}
}

func buildVolumeFromCreate(novelID, userID uint, req *dto.CreateVolumeRequest) (*model.NovelVolume, error) {
	if !validVolumeCreateStatus(req.Status) {
		return nil, ErrInvalidVolumeStatus
	}
	releaseDate, err := parseReleaseDate(req.ReleaseDate)
	if err != nil {
		return nil, err
	}
	if !validISBN(req.ISBN) {
		return nil, ErrInvalidISBN
	}
	if !validHTTPURL(req.CoverURL) {
		return nil, ErrInvalidVolumeURL
	}
	status := model.NovelStatusPending
	if req.Status != nil {
		status = *req.Status
	}
	return &model.NovelVolume{
		NovelID:       novelID,
		VolumeNumber:  req.VolumeNumber,
		Title:         strings.TrimSpace(req.Title),
		OriginalTitle: strings.TrimSpace(req.OriginalTitle),
		CoverURL:      strings.TrimSpace(req.CoverURL),
		ISBN:          strings.TrimSpace(req.ISBN),
		ReleaseDate:   releaseDate,
		Summary:       strings.TrimSpace(req.Summary),
		CreatedBy:     &userID,
		Status:        status,
	}, nil
}

func volumeChanged(volume *model.NovelVolume, req *dto.UpdateVolumeRequest, releaseDate *time.Time) bool {
	return !equalIntPointers(volume.VolumeNumber, req.VolumeNumber) ||
		volume.Title != strings.TrimSpace(req.Title) ||
		volume.OriginalTitle != strings.TrimSpace(req.OriginalTitle) ||
		volume.CoverURL != strings.TrimSpace(req.CoverURL) ||
		volume.ISBN != strings.TrimSpace(req.ISBN) ||
		!equalTimePointers(volume.ReleaseDate, releaseDate) ||
		volume.Summary != strings.TrimSpace(req.Summary) ||
		volume.Status != *req.Status
}

func equalIntPointers(left, right *int) bool {
	return (left == nil && right == nil) ||
		(left != nil && right != nil && *left == *right)
}

func validVolumeStatus(value int16) bool {
	return value >= model.NovelStatusPending && value <= model.NovelStatusHidden
}

func validVolumeCreateStatus(value *int16) bool {
	if value == nil {
		return true
	}
	return *value == model.NovelStatusPending || *value == model.NovelStatusPublished
}

// validISBN accepts empty values and ISBN-10 / ISBN-13 with optional hyphens
// or spaces; only length and character checks are performed.
func validISBN(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	digits := strings.NewReplacer("-", "", " ", "").Replace(value)
	if len(digits) != 10 && len(digits) != 13 {
		return false
	}
	for i, c := range digits {
		if c >= '0' && c <= '9' {
			continue
		}
		if (c == 'X' || c == 'x') && len(digits) == 10 && i == len(digits)-1 {
			continue
		}
		return false
	}
	return true
}

func volumeLabel(volume *model.NovelVolume) string {
	if volume.Title != "" {
		return volume.Title
	}
	if volume.VolumeNumber != nil {
		return fmt.Sprintf("Vol.%d", *volume.VolumeNumber)
	}
	return fmt.Sprintf("#%d", volume.ID)
}
