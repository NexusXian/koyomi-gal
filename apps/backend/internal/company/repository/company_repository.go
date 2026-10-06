package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"backend/internal/company/model"
	galgameModel "backend/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrCompanyNotFound  = errors.New("company not found")
	ErrParentNotFound   = errors.New("parent company not found")
	ErrParentCycle      = errors.New("company parent cycle")
	ErrGameNotFound     = errors.New("game not found")
	ErrUnknownCompanies = errors.New("unknown company ids")
)

type ListOptions struct {
	Keyword       string
	Status        string
	Country       string
	Sort          string
	Page          int
	PageSize      int
	PublishedOnly bool
}

type CompanyRepository struct{ db *gorm.DB }

func NewCompanyRepository(db *gorm.DB) *CompanyRepository { return &CompanyRepository{db: db} }

func (r *CompanyRepository) projection(ctx context.Context, publishedOnly bool) *gorm.DB {
	counts := r.db.WithContext(ctx).Table("game_companies AS gc").
		Select(`gc.company_id, COUNT(DISTINCT gc.game_id) AS game_count,
COUNT(DISTINCT gc.game_id) FILTER (WHERE gc.role = 'developer') AS developer_game_count,
COUNT(DISTINCT gc.game_id) FILTER (WHERE gc.role = 'publisher') AS publisher_game_count`).
		Joins("JOIN galgames AS g ON g.id = gc.game_id").Group("gc.company_id")
	if publishedOnly {
		counts = counts.Where("g.status = ?", galgameModel.GalgameStatusPublished)
	}
	return r.db.WithContext(ctx).Model(&model.Company{}).Table("companies AS c").
		Select(`c.*, COALESCE(counts.game_count, 0) AS game_count,
COALESCE(counts.developer_game_count, 0) AS developer_game_count,
COALESCE(counts.publisher_game_count, 0) AS publisher_game_count`).
		Joins("LEFT JOIN (?) AS counts ON counts.company_id = c.id", counts).
		Preload("Aliases", func(db *gorm.DB) *gorm.DB { return db.Order("company_aliases.id") }).
		Preload("ParentCompany").
		Preload("Children", func(db *gorm.DB) *gorm.DB { return db.Order("companies.name, companies.id") })
}

func (r *CompanyRepository) find(ctx context.Context, condition string, value any, publishedOnly bool) (*model.Company, error) {
	var company model.Company
	err := r.projection(ctx, publishedOnly).Where(condition, value).Take(&company).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find company: %w", err)
	}
	return &company, nil
}

func (r *CompanyRepository) FindByID(ctx context.Context, id uint, publishedOnly bool) (*model.Company, error) {
	return r.find(ctx, "c.id = ?", id, publishedOnly)
}

func (r *CompanyRepository) FindBySlug(ctx context.Context, slug string, publishedOnly bool) (*model.Company, error) {
	return r.find(ctx, "c.slug = ?", slug, publishedOnly)
}

func filters(query *gorm.DB, options ListOptions) *gorm.DB {
	if options.Status != "" {
		query = query.Where("c.status = ?", options.Status)
	}
	if options.Country != "" {
		query = query.Where("c.country = ?", options.Country)
	}
	if options.Keyword != "" {
		pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(options.Keyword) + "%"
		query = query.Where(`(c.name ILIKE ? ESCAPE E'\\' OR c.name_ja ILIKE ? ESCAPE E'\\'
OR c.name_zh ILIKE ? ESCAPE E'\\' OR c.name_en ILIKE ? ESCAPE E'\\'
OR EXISTS (SELECT 1 FROM company_aliases ca WHERE ca.company_id = c.id AND ca.alias ILIKE ? ESCAPE E'\\'))`, pattern, pattern, pattern, pattern, pattern)
	}
	return query
}

func (r *CompanyRepository) List(ctx context.Context, options ListOptions) ([]model.Company, int64, error) {
	var total int64
	if err := filters(r.db.WithContext(ctx).Table("companies AS c"), options).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "c.created_at DESC, c.id DESC"
	switch options.Sort {
	case "name":
		order = "c.name ASC, c.id ASC"
	case "game_count":
		order = "game_count DESC, c.id DESC"
	}
	var companies []model.Company
	err := filters(r.projection(ctx, options.PublishedOnly), options).Order(order).
		Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&companies).Error
	return companies, total, err
}

// Tree mutations share a transaction lock so concurrent reparenting cannot create cycles.
func lockTree(tx *gorm.DB) error { return tx.Exec("SELECT pg_advisory_xact_lock(727127)").Error }

func (r *CompanyRepository) Save(ctx context.Context, company *model.Company, aliases []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockTree(tx); err != nil {
			return err
		}
		if company.ID != 0 {
			var existing model.Company
			if err := tx.First(&existing, company.ID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrCompanyNotFound
				}
				return err
			}
			company.CreatedAt = existing.CreatedAt
		}
		if company.ParentCompanyID != nil {
			var parent model.Company
			if err := tx.First(&parent, *company.ParentCompanyID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrParentNotFound
				}
				return err
			}
			var cycle bool
			if err := tx.Raw(`WITH RECURSIVE ancestors AS (
SELECT id, parent_company_id FROM companies WHERE id = ?
UNION SELECT c.id, c.parent_company_id FROM companies c JOIN ancestors a ON c.id = a.parent_company_id
) SELECT EXISTS(SELECT 1 FROM ancestors WHERE id = ?)`, *company.ParentCompanyID, company.ID).Scan(&cycle).Error; err != nil {
				return err
			}
			if cycle {
				return ErrParentCycle
			}
		}
		if company.ID == 0 {
			if err := tx.Omit(clause.Associations).Create(company).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&model.Company{}).Where("id = ?", company.ID).
				Select("Name", "NameJa", "NameZh", "NameEn", "Slug", "Description", "LogoURL", "BannerURL", "WebsiteURL", "Country", "FoundedDate", "ClosedDate", "Status", "ParentCompanyID", "UpdatedAt").Updates(company).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("company_id = ?", company.ID).Delete(&model.CompanyAlias{}).Error; err != nil {
			return err
		}
		if len(aliases) == 0 {
			return nil
		}
		rows := make([]model.CompanyAlias, 0, len(aliases))
		for _, alias := range aliases {
			rows = append(rows, model.CompanyAlias{CompanyID: company.ID, Alias: alias})
		}
		return tx.Create(&rows).Error
	})
}

func (r *CompanyRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockTree(tx); err != nil {
			return err
		}
		result := tx.Delete(&model.Company{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrCompanyNotFound
		}
		return nil
	})
}

func (r *CompanyRepository) ReplaceGameCompanies(ctx context.Context, gameID uint, bindings []model.GameCompany) ([]model.GameCompany, error) {
	var result []model.GameCompany
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game galgameModel.Galgame
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&game, gameID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGameNotFound
			}
			return err
		}
		ids := make([]uint, 0, len(bindings))
		seen := make(map[uint]bool)
		for i := range bindings {
			bindings[i].GameID = gameID
			if !seen[bindings[i].CompanyID] {
				ids = append(ids, bindings[i].CompanyID)
				seen[bindings[i].CompanyID] = true
			}
		}
		if len(ids) > 0 {
			var companies []model.Company
			if err := tx.Clauses(clause.Locking{Strength: "KEY SHARE"}).Where("id IN ?", ids).Order("id").Find(&companies).Error; err != nil {
				return err
			}
			if len(companies) != len(ids) {
				return ErrUnknownCompanies
			}
		}
		if err := tx.Where("game_id = ?", gameID).Delete(&model.GameCompany{}).Error; err != nil {
			return err
		}
		if len(bindings) > 0 {
			if err := tx.Omit(clause.Associations).Create(&bindings).Error; err != nil {
				return err
			}
		}
		return tx.Where("game_id = ?", gameID).Order("company_id, role").Preload("Company").Find(&result).Error
	})
	return result, err
}
