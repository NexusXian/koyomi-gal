package ipgeo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LogRepository struct {
	db *gorm.DB
}

func NewLogRepository(db *gorm.DB) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) Create(ctx context.Context, entry *UserIPLog) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(entry).Error; err != nil {
		return fmt.Errorf("create user IP log: %w", err)
	}
	return nil
}

func (r *LogRepository) ListByUser(ctx context.Context, userID uint, page, limit int) ([]UserIPLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&UserIPLog{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user IP logs: %w", err)
	}
	items := make([]UserIPLog, 0)
	if err := query.Order("created_at DESC").Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list user IP logs: %w", err)
	}
	return items, total, nil
}

type AuditService struct {
	repository *LogRepository
}

func NewAuditService(repository *LogRepository) *AuditService {
	return &AuditService{repository: repository}
}

func (s *AuditService) Record(ctx context.Context, entry UserIPLog) error {
	return s.repository.Create(ctx, &entry)
}

func (s *AuditService) ListByUser(ctx context.Context, userID uint, page, limit int) ([]UserIPLog, int64, int, int, error) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	items, total, err := s.repository.ListByUser(ctx, userID, page, limit)
	return items, total, page, limit, err
}
