package service

import (
	"context"
	"errors"
	"strings"

	contributionModel "backend/internal/contribution/model"
	contributionService "backend/internal/contribution/service"
	galgameRepository "backend/internal/galgame/repository"
	"backend/internal/novel/dto"
	"backend/internal/novel/repository"
	relationModel "backend/internal/relation/model"
	relationRepository "backend/internal/relation/repository"
	"backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrRelationNotFound     = errors.New("work relation not found")
	ErrRelationExists       = errors.New("work relation already exists")
	ErrInvalidRelationInput = errors.New("invalid work relation input")
	ErrRelationTargetAbsent = errors.New("relation target not found")
)

type RelationService struct {
	relations     *relationRepository.RelationRepository
	novels        *repository.NovelRepository
	galgames      *galgameRepository.GalgameRepository
	contributions *contributionService.ContributionService
}

func NewRelationService(
	relations *relationRepository.RelationRepository,
	novels *repository.NovelRepository,
	galgames *galgameRepository.GalgameRepository,
) *RelationService {
	return &RelationService{relations: relations, novels: novels, galgames: galgames}
}

func (s *RelationService) SetContributionService(contributions *contributionService.ContributionService) {
	s.contributions = contributions
}

// ListRelations returns both directions of the novel's raw relations.
func (s *RelationService) ListRelations(
	ctx context.Context,
	novelID uint,
) ([]relationModel.WorkRelation, error) {
	return s.relations.ListForNovel(ctx, novelID)
}

// CreateRelation links the novel to a published galgame or another published
// novel. The novel itself is always the source row; display queries resolve
// both directions.
func (s *RelationService) CreateRelation(
	ctx context.Context,
	actorID, novelID uint,
	req *dto.CreateRelationRequest,
) (*relationModel.WorkRelation, error) {
	if !relationModel.ValidWorkType(req.TargetType) || req.TargetID == 0 {
		return nil, ErrInvalidRelationInput
	}
	relationType := relationModel.RelationType(strings.TrimSpace(req.RelationType))
	if !relationModel.ValidRelationType(relationType) {
		return nil, ErrInvalidRelationInput
	}
	if req.TargetType == relationModel.WorkTypeNovel && req.TargetID == novelID {
		return nil, ErrInvalidRelationInput
	}
	if err := s.ensureTargetPublished(ctx, req.TargetType, req.TargetID); err != nil {
		return nil, err
	}
	if err := s.ensureRelationAbsent(ctx, novelID, req.TargetType, req.TargetID, relationType); err != nil {
		return nil, err
	}

	relation := &relationModel.WorkRelation{
		SourceType:   relationModel.WorkTypeNovel,
		SourceID:     novelID,
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		RelationType: relationType,
		CreatedBy:    &actorID,
	}
	write := func(db *gorm.DB) error {
		if err := relationRepository.NewRelationRepository(db).Create(ctx, relation); err != nil {
			return err
		}
		if s.contributions != nil {
			sourceType := contributionModel.ContributionSourceWorkRelation
			sourceID := relation.ID
			return s.contributions.RecordContribution(ctx, contributionService.RecordContributionInput{
				TargetType: relationModel.WorkTypeNovel,
				TargetID:   novelID,
				UserID:     actorID,
				Action:     contributionModel.ContributionActionAddRelation,
				SourceType: &sourceType,
				SourceID:   &sourceID,
			}, db)
		}
		return nil
	}
	var err error
	if s.contributions != nil {
		err = s.contributions.Transaction(ctx, write)
	} else {
		err = s.relations.Transaction(ctx, write)
	}
	if err != nil {
		if hasConstraint(err, "work_relations_unique") {
			return nil, ErrRelationExists
		}
		logger.Error("create work relation", zap.Uint("novel_id", novelID), zap.Error(err))
		return nil, err
	}
	return relation, nil
}

// DeleteRelation removes a relation that references the novel on either side.
func (s *RelationService) DeleteRelation(ctx context.Context, novelID, relationID uint) error {
	relation, err := s.relations.FindByID(ctx, relationID)
	if err != nil {
		logger.Error("find work relation by id", zap.Uint("relation_id", relationID), zap.Error(err))
		return err
	}
	if relation == nil {
		return ErrRelationNotFound
	}
	involvesNovel := (relation.SourceType == relationModel.WorkTypeNovel && relation.SourceID == novelID) ||
		(relation.TargetType == relationModel.WorkTypeNovel && relation.TargetID == novelID)
	if !involvesNovel {
		return ErrRelationNotFound
	}
	removed, err := s.relations.Delete(ctx, relationID)
	if err != nil {
		logger.Error("delete work relation", zap.Uint("relation_id", relationID), zap.Error(err))
		return err
	}
	if !removed {
		return ErrRelationNotFound
	}
	return nil
}

func (s *RelationService) ensureTargetPublished(ctx context.Context, targetType string, targetID uint) error {
	switch targetType {
	case relationModel.WorkTypeGalgame:
		galgame, err := s.galgames.FindPublishedByID(ctx, targetID)
		if err != nil {
			logger.Error("find galgame by id", zap.Uint("galgame_id", targetID), zap.Error(err))
			return err
		}
		if galgame == nil {
			return ErrRelationTargetAbsent
		}
		return nil
	case relationModel.WorkTypeNovel:
		novel, err := s.novels.FindPublishedByID(ctx, targetID)
		if err != nil {
			logger.Error("find novel by id", zap.Uint("novel_id", targetID), zap.Error(err))
			return err
		}
		if novel == nil {
			return ErrRelationTargetAbsent
		}
		return nil
	default:
		return ErrInvalidRelationInput
	}
}

// ensureRelationAbsent rejects duplicates in either direction so a pair is
// never shown twice on the same page.
func (s *RelationService) ensureRelationAbsent(
	ctx context.Context,
	novelID uint,
	targetType string,
	targetID uint,
	relationType relationModel.RelationType,
) error {
	existing, err := s.relations.ListForNovel(ctx, novelID)
	if err != nil {
		logger.Error("list relations for novel", zap.Uint("novel_id", novelID), zap.Error(err))
		return err
	}
	for _, relation := range existing {
		if relation.RelationType != relationType {
			continue
		}
		forward := relation.SourceType == relationModel.WorkTypeNovel &&
			relation.SourceID == novelID &&
			relation.TargetType == targetType &&
			relation.TargetID == targetID
		backward := relation.TargetType == relationModel.WorkTypeNovel &&
			relation.TargetID == novelID &&
			relation.SourceType == targetType &&
			relation.SourceID == targetID
		if forward || backward {
			return ErrRelationExists
		}
	}
	return nil
}
