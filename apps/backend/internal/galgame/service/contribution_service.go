package service

import (
	"context"
	"errors"
	"strings"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"

	"gorm.io/gorm"
)

var ErrInvalidContribution = errors.New("invalid contribution")

type RecordContributionInput struct {
	GalgameID  uint
	UserID     uint
	Action     model.ContributionAction
	SourceType *string
	SourceID   *uint
}

type ContributionService struct {
	repository *repository.ContributionRepository
	galgames   *repository.GalgameRepository
	publicURL  string
}

func NewContributionService(
	repository *repository.ContributionRepository,
	galgames *repository.GalgameRepository,
	publicURL string,
) *ContributionService {
	return &ContributionService{repository: repository, galgames: galgames, publicURL: publicURL}
}

func (s *ContributionService) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.repository.Transaction(ctx, fn)
}

func (s *ContributionService) RecordContribution(
	ctx context.Context,
	input RecordContributionInput,
	tx ...*gorm.DB,
) error {
	if input.GalgameID == 0 || input.UserID == 0 || !validContributionAction(input.Action) {
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
	return repo.CreateContribution(ctx, &model.GalgameContribution{
		GalgameID:  input.GalgameID,
		UserID:     input.UserID,
		Action:     input.Action,
		SourceType: input.SourceType,
		SourceID:   input.SourceID,
	})
}

func (s *ContributionService) ListContributorsByGalgameID(
	ctx context.Context,
	galgameID uint,
	page, pageSize int,
) (dto.ContributorListData, error) {
	galgame, err := s.galgames.FindPublishedByID(ctx, galgameID)
	if err != nil {
		return dto.ContributorListData{}, err
	}
	if galgame == nil {
		return dto.ContributorListData{}, ErrGalgameNotFound
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	contributors, total, err := s.repository.ListContributorsByGalgameID(ctx, galgameID, page, pageSize)
	if err != nil {
		return dto.ContributorListData{}, err
	}
	return dto.ContributorListData{
		Items:    dto.NewContributorData(contributors),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func validContributionAction(action model.ContributionAction) bool {
	switch action {
	case model.ContributionActionCreate,
		model.ContributionActionEdit,
		model.ContributionActionCover,
		model.ContributionActionGallery,
		model.ContributionActionResource:
		return true
	default:
		return false
	}
}
