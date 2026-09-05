package service

import (
	"context"
	"errors"
	"strings"

	contributionModel "backend/internal/contribution/model"
	"backend/internal/contribution/repository"
	galgameRepository "backend/internal/galgame/repository"
	relationModel "backend/internal/relation/model"

	"gorm.io/gorm"
)

var (
	ErrInvalidContribution = errors.New("invalid contribution")
	ErrGalgameNotFound     = errors.New("galgame not found")
)

type RecordContributionInput struct {
	TargetType string
	TargetID   uint
	UserID     uint
	Action     contributionModel.ContributionAction
	SourceType *string
	SourceID   *uint
}

type ContributionService struct {
	repository *repository.ContributionRepository
	galgames   *galgameRepository.GalgameRepository
	publicURL  string
}

func NewContributionService(
	repository *repository.ContributionRepository,
	galgames *galgameRepository.GalgameRepository,
	publicURL string,
) *ContributionService {
	return &ContributionService{repository: repository, galgames: galgames, publicURL: publicURL}
}

func (s *ContributionService) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.repository.Transaction(ctx, fn)
}

// HasSourceContribution reports whether a contribution already exists for the
// given source, so an entity credited once (e.g. a gallery image) is never
// double-counted across review transitions.
func (s *ContributionService) HasSourceContribution(
	ctx context.Context,
	sourceType string,
	sourceID uint,
	tx ...*gorm.DB,
) (bool, error) {
	repo := s.repository
	if len(tx) > 0 && tx[0] != nil {
		repo = repository.NewContributionRepository(tx[0], s.publicURL)
	}
	return repo.ExistsBySource(ctx, sourceType, sourceID)
}

func (s *ContributionService) RecordContribution(
	ctx context.Context,
	input RecordContributionInput,
	tx ...*gorm.DB,
) error {
	if !relationModel.ValidWorkType(input.TargetType) || input.TargetID == 0 ||
		input.UserID == 0 || !contributionModel.ValidContributionAction(input.Action) {
		return ErrInvalidContribution
	}
	if (input.SourceType == nil) != (input.SourceID == nil) {
		return ErrInvalidContribution
	}
	if input.SourceType != nil {
		sourceType := strings.TrimSpace(*input.SourceType)
		if sourceType == "" || len(sourceType) > 32 || *input.SourceID == 0 {
			return ErrInvalidContribution
		}
		input.SourceType = &sourceType
	}
	repo := s.repository
	if len(tx) > 0 && tx[0] != nil {
		repo = repository.NewContributionRepository(tx[0], s.publicURL)
	}
	return repo.CreateContribution(ctx, &contributionModel.WorkContribution{
		TargetType: input.TargetType,
		TargetID:   input.TargetID,
		UserID:     input.UserID,
		Action:     input.Action,
		SourceType: input.SourceType,
		SourceID:   input.SourceID,
	})
}

// ListGalgameContributorRows backs the public galgame contributors endpoint
// and only exposes published galgames.
func (s *ContributionService) ListGalgameContributorRows(
	ctx context.Context,
	galgameID uint,
	page, pageSize int,
) ([]contributionModel.WorkContributor, int64, error) {
	galgame, err := s.galgames.FindPublishedByID(ctx, galgameID)
	if err != nil {
		return nil, 0, err
	}
	if galgame == nil {
		return nil, 0, ErrGalgameNotFound
	}
	return s.ListContributorRows(ctx, relationModel.WorkTypeGalgame, galgameID, page, pageSize)
}

// ListContributorRows returns the raw aggregated contributor rows so domain
// services can embed them into their detail responses.
func (s *ContributionService) ListContributorRows(
	ctx context.Context,
	targetType string,
	targetID uint,
	page, pageSize int,
) ([]contributionModel.WorkContributor, int64, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	return s.repository.ListContributors(ctx, targetType, targetID, page, pageSize)
}
