package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"
	"backend/pkg/logger"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	ErrDeveloperNotFound    = errors.New("developer not found")
	ErrDeveloperSlugExists  = errors.New("developer slug already exists")
	ErrTagNotFound          = errors.New("tag not found")
	ErrTagNameExists        = errors.New("tag name already exists")
	ErrTagSlugExists        = errors.New("tag slug already exists")
	ErrGalgameNotFound      = errors.New("galgame not found")
	ErrGalgameSlugExists    = errors.New("galgame slug already exists")
	ErrUnknownTagIDs        = errors.New("tag ids contain unknown tag")
	ErrInvalidReleaseDate   = errors.New("invalid release date")
	ErrInvalidAgeRating     = errors.New("invalid age rating")
	ErrInvalidStatus        = errors.New("invalid galgame status")
	ErrInvalidSort          = errors.New("invalid galgame sort")
	ErrInvalidReleaseRange  = errors.New("invalid release year range")
	ErrInvalidCatalogInput  = errors.New("invalid catalog input")
	ErrInvalidMyGalgameType = errors.New("invalid my galgame type")
)

type CatalogService struct {
	galgames   *repository.GalgameRepository
	developers *repository.DeveloperRepository
	tags       *repository.TagRepository
}

func NewCatalogService(
	galgames *repository.GalgameRepository,
	developers *repository.DeveloperRepository,
	tags *repository.TagRepository,
) *CatalogService {
	return &CatalogService{galgames: galgames, developers: developers, tags: tags}
}

func (s *CatalogService) CreateDeveloper(
	ctx context.Context,
	req *dto.CreateDeveloperRequest,
) (*model.Developer, error) {
	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if name == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	existing, err := s.developers.FindBySlug(ctx, slug)
	if err != nil {
		logger.Error("find developer by slug", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	if existing != nil {
		return nil, ErrDeveloperSlugExists
	}

	developer := &model.Developer{
		Name:         name,
		OriginalName: strings.TrimSpace(req.OriginalName),
		Slug:         slug,
		Description:  strings.TrimSpace(req.Description),
		LogoURL:      strings.TrimSpace(req.LogoURL),
		Website:      strings.TrimSpace(req.Website),
	}
	if err := s.developers.Create(ctx, developer); err != nil {
		if hasConstraint(err, "developers_slug_unique") {
			return nil, ErrDeveloperSlugExists
		}
		logger.Error("create developer", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	return developer, nil
}

func (s *CatalogService) UpdateDeveloper(
	ctx context.Context,
	id uint,
	req *dto.UpdateDeveloperRequest,
) (*model.Developer, error) {
	developer, err := s.developers.FindByID(ctx, id)
	if err != nil {
		logger.Error("find developer by id", zap.Uint("developer_id", id), zap.Error(err))
		return nil, err
	}
	if developer == nil {
		return nil, ErrDeveloperNotFound
	}

	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if name == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	existing, err := s.developers.FindBySlug(ctx, slug)
	if err != nil {
		logger.Error("find developer by slug", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	if existing != nil && existing.ID != id {
		return nil, ErrDeveloperSlugExists
	}

	developer.Name = name
	developer.OriginalName = strings.TrimSpace(req.OriginalName)
	developer.Slug = slug
	developer.Description = strings.TrimSpace(req.Description)
	developer.LogoURL = strings.TrimSpace(req.LogoURL)
	developer.Website = strings.TrimSpace(req.Website)
	if err := s.developers.Update(ctx, developer); err != nil {
		if hasConstraint(err, "developers_slug_unique") {
			return nil, ErrDeveloperSlugExists
		}
		logger.Error("update developer", zap.Uint("developer_id", id), zap.Error(err))
		return nil, err
	}
	return developer, nil
}

func (s *CatalogService) ListDevelopers(ctx context.Context) ([]model.Developer, error) {
	return s.developers.List(ctx)
}

// GetDeveloper returns a developer by ID for the public detail endpoint.
func (s *CatalogService) GetDeveloper(ctx context.Context, id uint) (*model.Developer, error) {
	developer, err := s.developers.FindByID(ctx, id)
	if err != nil {
		logger.Error("find developer by id", zap.Uint("developer_id", id), zap.Error(err))
		return nil, err
	}
	if developer == nil {
		return nil, ErrDeveloperNotFound
	}
	return developer, nil
}

func (s *CatalogService) CreateTag(ctx context.Context, req *dto.CreateTagRequest) (*model.Tag, error) {
	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if name == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	if err := s.ensureTagUnique(ctx, 0, name, slug); err != nil {
		return nil, err
	}

	tag := &model.Tag{
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(req.Description),
	}
	if err := s.tags.Create(ctx, tag); err != nil {
		switch {
		case hasConstraint(err, "tags_name_unique"):
			return nil, ErrTagNameExists
		case hasConstraint(err, "tags_slug_unique"):
			return nil, ErrTagSlugExists
		}
		logger.Error("create tag", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	return tag, nil
}

func (s *CatalogService) UpdateTag(
	ctx context.Context,
	id uint,
	req *dto.UpdateTagRequest,
) (*model.Tag, error) {
	tag, err := s.tags.FindByID(ctx, id)
	if err != nil {
		logger.Error("find tag by id", zap.Uint("tag_id", id), zap.Error(err))
		return nil, err
	}
	if tag == nil {
		return nil, ErrTagNotFound
	}

	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if name == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	if err := s.ensureTagUnique(ctx, id, name, slug); err != nil {
		return nil, err
	}
	tag.Name = name
	tag.Slug = slug
	tag.Description = strings.TrimSpace(req.Description)
	if err := s.tags.Update(ctx, tag); err != nil {
		switch {
		case hasConstraint(err, "tags_name_unique"):
			return nil, ErrTagNameExists
		case hasConstraint(err, "tags_slug_unique"):
			return nil, ErrTagSlugExists
		}
		logger.Error("update tag", zap.Uint("tag_id", id), zap.Error(err))
		return nil, err
	}
	return tag, nil
}

func (s *CatalogService) ListTags(ctx context.Context) ([]model.Tag, error) {
	return s.tags.List(ctx)
}

// GetTag returns a tag by ID for the public detail endpoint.
func (s *CatalogService) GetTag(ctx context.Context, id uint) (*model.Tag, error) {
	tag, err := s.tags.FindByID(ctx, id)
	if err != nil {
		logger.Error("find tag by id", zap.Uint("tag_id", id), zap.Error(err))
		return nil, err
	}
	if tag == nil {
		return nil, ErrTagNotFound
	}
	return tag, nil
}

func (s *CatalogService) CreateGalgame(
	ctx context.Context,
	userID uint,
	req *dto.CreateGalgameRequest,
) (*model.Galgame, error) {
	title := strings.TrimSpace(req.Title)
	slug := normalizeSlug(req.Slug)
	if title == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	releaseDate, err := parseReleaseDate(req.ReleaseDate)
	if err != nil {
		return nil, err
	}
	if !validAgeRating(req.AgeRating) {
		return nil, ErrInvalidAgeRating
	}
	if !validStatus(req.Status) {
		return nil, ErrInvalidStatus
	}

	tagIDs := uniqueUint(req.TagIDs)
	if err := s.validateRelations(ctx, req.DeveloperID, tagIDs); err != nil {
		return nil, err
	}
	if err := s.ensureGalgameSlugUnique(ctx, 0, slug); err != nil {
		return nil, err
	}

	galgame := &model.Galgame{
		Title:         title,
		OriginalTitle: strings.TrimSpace(req.OriginalTitle),
		RomajiTitle:   strings.TrimSpace(req.RomajiTitle),
		Slug:          slug,
		Description:   strings.TrimSpace(req.Description),
		CoverURL:      strings.TrimSpace(req.CoverURL),
		BannerURL:     strings.TrimSpace(req.BannerURL),
		DeveloperID:   req.DeveloperID,
		ReleaseDate:   releaseDate,
		AgeRating:     req.AgeRating,
		Status:        req.Status,
		CreatedBy:     &userID,
	}
	aliases := uniqueNonEmptyStrings(req.Aliases)
	err = s.galgames.Transaction(ctx, func(tx *repository.GalgameRepository) error {
		if err := tx.Create(ctx, galgame); err != nil {
			return err
		}
		if err := tx.ReplaceAliases(ctx, galgame.ID, aliases); err != nil {
			return err
		}
		return tx.ReplaceTags(ctx, galgame.ID, tagIDs)
	})
	if err != nil {
		if hasConstraint(err, "galgames_slug_unique") {
			return nil, ErrGalgameSlugExists
		}
		logger.Error("create galgame", zap.String("slug", slug), zap.Error(err))
		return nil, err
	}
	return s.getGalgame(ctx, galgame.ID, false)
}

func (s *CatalogService) UpdateGalgame(
	ctx context.Context,
	id uint,
	req *dto.UpdateGalgameRequest,
) (*model.Galgame, error) {
	title := strings.TrimSpace(req.Title)
	slug := normalizeSlug(req.Slug)
	if title == "" || slug == "" {
		return nil, ErrInvalidCatalogInput
	}
	galgame, err := s.galgames.FindByID(ctx, id)
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalgameNotFound
	}
	if req.AgeRating == nil || !validAgeRating(*req.AgeRating) {
		return nil, ErrInvalidAgeRating
	}
	if req.Status == nil || !validStatus(*req.Status) {
		return nil, ErrInvalidStatus
	}
	releaseDate, err := parseReleaseDate(req.ReleaseDate)
	if err != nil {
		return nil, err
	}
	tagIDs := uniqueUint(req.TagIDs)
	if err := s.validateRelations(ctx, req.DeveloperID, tagIDs); err != nil {
		return nil, err
	}
	if err := s.ensureGalgameSlugUnique(ctx, id, slug); err != nil {
		return nil, err
	}

	galgame.Title = title
	galgame.OriginalTitle = strings.TrimSpace(req.OriginalTitle)
	galgame.RomajiTitle = strings.TrimSpace(req.RomajiTitle)
	galgame.Slug = slug
	galgame.Description = strings.TrimSpace(req.Description)
	galgame.CoverURL = strings.TrimSpace(req.CoverURL)
	galgame.BannerURL = strings.TrimSpace(req.BannerURL)
	galgame.DeveloperID = req.DeveloperID
	galgame.ReleaseDate = releaseDate
	galgame.AgeRating = *req.AgeRating
	galgame.Status = *req.Status
	aliases := uniqueNonEmptyStrings(req.Aliases)
	err = s.galgames.Transaction(ctx, func(tx *repository.GalgameRepository) error {
		if err := tx.Update(ctx, galgame); err != nil {
			return err
		}
		if err := tx.ReplaceAliases(ctx, id, aliases); err != nil {
			return err
		}
		return tx.ReplaceTags(ctx, id, tagIDs)
	})
	if err != nil {
		if hasConstraint(err, "galgames_slug_unique") {
			return nil, ErrGalgameSlugExists
		}
		logger.Error("update galgame", zap.Uint("galgame_id", id), zap.Error(err))
		return nil, err
	}
	return s.getGalgame(ctx, id, false)
}

func (s *CatalogService) DeleteGalgame(ctx context.Context, id uint) error {
	galgame, err := s.galgames.FindByID(ctx, id)
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return err
	}
	if galgame == nil {
		return ErrGalgameNotFound
	}
	if err := s.galgames.Delete(ctx, id); err != nil {
		logger.Error("delete galgame", zap.Uint("galgame_id", id), zap.Error(err))
		return err
	}
	return nil
}

func (s *CatalogService) GetPublishedGalgame(ctx context.Context, id uint) (*model.Galgame, error) {
	return s.getGalgame(ctx, id, true)
}

// GetGalgame returns a galgame of any status for the galgame:review admin
// detail endpoint.
func (s *CatalogService) GetGalgame(ctx context.Context, id uint) (*model.Galgame, error) {
	return s.getGalgame(ctx, id, false)
}

// ListAllGalgames lists galgames of every status with an optional status
// filter for the galgame:review admin listing.
func (s *CatalogService) ListAllGalgames(
	ctx context.Context,
	query *dto.AdminGalgameQuery,
) ([]model.Galgame, int64, int, int, error) {
	page := query.Page
	if page == 0 {
		page = 1
	}
	limit := query.Limit
	if limit == 0 {
		limit = 20
	}
	sort := strings.ToLower(strings.TrimSpace(query.Sort))
	if sort == "" {
		sort = "latest"
	}
	if !validSort(sort) {
		return nil, 0, page, limit, ErrInvalidSort
	}
	if query.Status != nil && !validStatus(*query.Status) {
		return nil, 0, page, limit, ErrInvalidStatus
	}

	galgames, total, err := s.galgames.ListAdmin(ctx, repository.GalgameListOptions{
		Keyword: strings.TrimSpace(query.Keyword),
		Status:  query.Status,
		Sort:    sort,
		Page:    page,
		Limit:   limit,
	})
	if err != nil {
		logger.Error("list all galgames", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return galgames, total, page, limit, nil
}

func (s *CatalogService) ListPublishedGalgames(
	ctx context.Context,
	query *dto.GalgameQuery,
) ([]model.Galgame, int64, int, int, error) {
	page := query.Page
	if page == 0 {
		page = 1
	}
	limit := query.Limit
	if limit == 0 {
		limit = 20
	}
	sort := strings.ToLower(strings.TrimSpace(query.Sort))
	if sort == "" {
		sort = "latest"
	}
	if !validSort(sort) {
		return nil, 0, page, limit, ErrInvalidSort
	}
	if query.ReleaseFrom != nil && query.ReleaseTo != nil && *query.ReleaseFrom > *query.ReleaseTo {
		return nil, 0, page, limit, ErrInvalidReleaseRange
	}
	if query.AgeRating != nil && !validAgeRating(*query.AgeRating) {
		return nil, 0, page, limit, ErrInvalidAgeRating
	}

	galgames, total, err := s.galgames.ListPublished(ctx, repository.GalgameListOptions{
		Keyword:     strings.TrimSpace(query.Keyword),
		DeveloperID: query.DeveloperID,
		TagIDs:      uniqueUint(query.TagIDs),
		ReleaseFrom: query.ReleaseFrom,
		ReleaseTo:   query.ReleaseTo,
		AgeRating:   query.AgeRating,
		Sort:        sort,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		logger.Error("list published galgames", zap.Error(err))
		return nil, 0, page, limit, err
	}
	return galgames, total, page, limit, nil
}

func (s *CatalogService) ListMyGalgames(
	ctx context.Context,
	userID uint,
	query *dto.MyGalgameQuery,
) ([]model.Galgame, int64, int, int, error) {
	page := query.Page
	if page == 0 {
		page = 1
	}
	limit := query.Limit
	if limit == 0 {
		limit = 20
	}
	collectionType := strings.ToLower(strings.TrimSpace(query.Type))
	if collectionType == "" {
		collectionType = "uploaded"
	}

	var (
		galgames []model.Galgame
		total    int64
		err      error
	)
	switch collectionType {
	case "uploaded":
		galgames, total, err = s.galgames.ListByCreator(ctx, userID, page, limit)
	case "favorite":
		galgames, total, err = s.galgames.ListFavoritesByUser(ctx, userID, page, limit)
	default:
		return nil, 0, page, limit, ErrInvalidMyGalgameType
	}
	if err != nil {
		logger.Error("list my galgames", zap.Uint("user_id", userID), zap.Error(err))
		return nil, 0, page, limit, err
	}
	return galgames, total, page, limit, nil
}

func (s *CatalogService) getGalgame(
	ctx context.Context,
	id uint,
	publishedOnly bool,
) (*model.Galgame, error) {
	var (
		galgame *model.Galgame
		err     error
	)
	if publishedOnly {
		galgame, err = s.galgames.FindPublishedByID(ctx, id)
	} else {
		galgame, err = s.galgames.FindByID(ctx, id)
	}
	if err != nil {
		logger.Error("find galgame by id", zap.Uint("galgame_id", id), zap.Error(err))
		return nil, err
	}
	if galgame == nil {
		return nil, ErrGalgameNotFound
	}
	return galgame, nil
}

func (s *CatalogService) validateRelations(ctx context.Context, developerID *uint, tagIDs []uint) error {
	if developerID != nil {
		developer, err := s.developers.FindByID(ctx, *developerID)
		if err != nil {
			logger.Error("find developer by id", zap.Uint("developer_id", *developerID), zap.Error(err))
			return err
		}
		if developer == nil {
			return ErrDeveloperNotFound
		}
	}
	if len(tagIDs) == 0 {
		return nil
	}
	count, err := s.tags.CountByIDs(ctx, tagIDs)
	if err != nil {
		logger.Error("count tags by ids", zap.Error(err))
		return err
	}
	if count != int64(len(tagIDs)) {
		return ErrUnknownTagIDs
	}
	return nil
}

func (s *CatalogService) ensureGalgameSlugUnique(ctx context.Context, id uint, slug string) error {
	existing, err := s.galgames.FindBySlug(ctx, slug)
	if err != nil {
		logger.Error("find galgame by slug", zap.String("slug", slug), zap.Error(err))
		return err
	}
	if existing != nil && existing.ID != id {
		return ErrGalgameSlugExists
	}
	return nil
}

func (s *CatalogService) ensureTagUnique(ctx context.Context, id uint, name, slug string) error {
	existing, err := s.tags.FindByName(ctx, name)
	if err != nil {
		logger.Error("find tag by name", zap.String("name", name), zap.Error(err))
		return err
	}
	if existing != nil && existing.ID != id {
		return ErrTagNameExists
	}
	existing, err = s.tags.FindBySlug(ctx, slug)
	if err != nil {
		logger.Error("find tag by slug", zap.String("slug", slug), zap.Error(err))
		return err
	}
	if existing != nil && existing.ID != id {
		return ErrTagSlugExists
	}
	return nil
}

func parseReleaseDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, ErrInvalidReleaseDate
	}
	return &parsed, nil
}

func validAgeRating(value int16) bool {
	return value >= model.AgeRatingUnknown && value <= model.AgeRatingR18
}

func validStatus(value int16) bool {
	return value >= model.GalgameStatusPending && value <= model.GalgameStatusHidden
}

func validSort(value string) bool {
	switch value {
	case "latest", "oldest", "rating", "favorite", "popular":
		return true
	default:
		return false
	}
}

func normalizeSlug(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func hasConstraint(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == constraint
}

func uniqueUint(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	unique := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
