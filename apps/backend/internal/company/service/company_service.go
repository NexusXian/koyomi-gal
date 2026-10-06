package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"backend/internal/company/dto"
	"backend/internal/company/model"
	"backend/internal/company/repository"
	galgameDTO "backend/internal/galgame/dto"
	galgameRepo "backend/internal/galgame/repository"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrCompanyNotFound  = repository.ErrCompanyNotFound
	ErrParentNotFound   = repository.ErrParentNotFound
	ErrParentCycle      = repository.ErrParentCycle
	ErrGameNotFound     = repository.ErrGameNotFound
	ErrUnknownCompanies = repository.ErrUnknownCompanies
	ErrSlugExists       = errors.New("company slug already exists")
	ErrInvalidInput     = errors.New("invalid company input")
	ErrInvalidDate      = errors.New("invalid company date")
	ErrDuplicateBinding = errors.New("duplicate company and role")
)

var validate = func() *validator.Validate { v := validator.New(); v.SetTagName("binding"); return v }()

type CompanyService struct {
	companies *repository.CompanyRepository
	games     *galgameRepo.GalgameRepository
}

func NewCompanyService(companies *repository.CompanyRepository, games *galgameRepo.GalgameRepository) *CompanyService {
	return &CompanyService{companies: companies, games: games}
}

func (s *CompanyService) Get(ctx context.Context, id uint) (*model.Company, error) {
	c, err := s.companies.FindByID(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCompanyNotFound
	}
	return c, nil
}

func (s *CompanyService) GetBySlug(ctx context.Context, slug string) (*model.Company, error) {
	c, err := s.companies.FindBySlug(ctx, strings.TrimSpace(slug), true)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCompanyNotFound
	}
	return c, nil
}

func pagination(page, size int) (int, int) {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	return page, size
}

func (s *CompanyService) List(ctx context.Context, query dto.CompanyQuery, public bool) (dto.CompanyListData, error) {
	if err := validate.Struct(query); err != nil {
		return dto.CompanyListData{}, ErrInvalidInput
	}
	page, size := pagination(query.Page, query.PageSize)
	companies, total, err := s.companies.List(ctx, repository.ListOptions{
		Keyword: strings.TrimSpace(query.Keyword), Status: query.Status, Country: strings.TrimSpace(query.Country), Sort: query.Sort,
		Page: page, PageSize: size, PublishedOnly: public,
	})
	if err != nil {
		return dto.CompanyListData{}, err
	}
	items := make([]dto.CompanyResponse, 0, len(companies))
	for i := range companies {
		items = append(items, dto.NewCompanyResponse(&companies[i]))
	}
	return dto.CompanyListData{Items: items, Total: total, Page: page, PageSize: size}, nil
}

func parseDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *value)
	if err != nil || t.Year() < 1 {
		return nil, ErrInvalidDate
	}
	return &t, nil
}

func (s *CompanyService) Save(ctx context.Context, id uint, req *dto.CompanyRequest) (*model.Company, error) {
	if req == nil || validate.Struct(req) != nil {
		return nil, ErrInvalidInput
	}
	name, slug := strings.TrimSpace(req.Name), strings.ToLower(strings.TrimSpace(req.Slug))
	if name == "" || slug == "" || slug == "." || slug == ".." || strings.ContainsAny(slug, `/\`) || strings.IndexFunc(slug, unicode.IsControl) >= 0 {
		return nil, ErrInvalidInput
	}
	founded, err := parseDate(req.FoundedDate)
	if err != nil {
		return nil, err
	}
	closed, err := parseDate(req.ClosedDate)
	if err != nil {
		return nil, err
	}
	if founded != nil && closed != nil && closed.Before(*founded) {
		return nil, ErrInvalidDate
	}
	aliases := make([]string, 0, len(req.Aliases))
	seen := make(map[string]bool)
	for _, alias := range req.Aliases {
		alias = strings.TrimSpace(alias)
		if alias != "" && !seen[alias] {
			aliases = append(aliases, alias)
			seen[alias] = true
		}
	}
	c := &model.Company{ID: id, Name: name, NameJa: strings.TrimSpace(req.NameJa), NameZh: strings.TrimSpace(req.NameZh), NameEn: strings.TrimSpace(req.NameEn),
		Slug: slug, Description: strings.TrimSpace(req.Description), LogoURL: strings.TrimSpace(req.LogoURL), BannerURL: strings.TrimSpace(req.BannerURL),
		WebsiteURL: strings.TrimSpace(req.WebsiteURL), Country: strings.TrimSpace(req.Country), FoundedDate: founded, ClosedDate: closed, Status: req.Status, ParentCompanyID: req.ParentCompanyID}
	if err := s.companies.Save(ctx, c, aliases); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "companies_slug_unique" {
			return nil, ErrSlugExists
		}
		return nil, err
	}
	return s.Get(ctx, c.ID)
}

func (s *CompanyService) Delete(ctx context.Context, id uint) error {
	return s.companies.Delete(ctx, id)
}

func (s *CompanyService) ListGames(ctx context.Context, slug string, query dto.CompanyGamesQuery) (galgameDTO.GalgameListData, error) {
	if validate.Struct(query) != nil {
		return galgameDTO.GalgameListData{}, ErrInvalidInput
	}
	c, err := s.GetBySlug(ctx, slug)
	if err != nil {
		return galgameDTO.GalgameListData{}, err
	}
	page, size := pagination(query.Page, query.PageSize)
	games, total, err := s.games.ListPublished(ctx, galgameRepo.GalgameListOptions{CompanyID: &c.ID, CompanyRole: query.Role,
		Keyword: strings.TrimSpace(query.Keyword), Page: page, Limit: size, Sort: query.Sort})
	if err != nil {
		return galgameDTO.GalgameListData{}, err
	}
	return galgameDTO.GalgameListData{Items: galgameDTO.NewGalgameListItems(games), Total: total, Page: page, Limit: size}, nil
}

func (s *CompanyService) ReplaceGameCompanies(ctx context.Context, gameID uint, req *dto.ReplaceGameCompaniesRequest) ([]dto.GameCompanyResponse, error) {
	if gameID == 0 || req == nil || validate.Struct(req) != nil {
		return nil, ErrInvalidInput
	}
	bindings := make([]model.GameCompany, 0, len(req.Companies))
	seen := make(map[dto.GameCompanyInput]bool)
	for _, input := range req.Companies {
		if seen[input] {
			return nil, ErrDuplicateBinding
		}
		seen[input] = true
		bindings = append(bindings, model.GameCompany{GameID: gameID, CompanyID: input.CompanyID, Role: input.Role})
	}
	result, err := s.companies.ReplaceGameCompanies(ctx, gameID, bindings)
	if err != nil {
		return nil, err
	}
	return dto.NewGameCompanyResponses(result), nil
}
