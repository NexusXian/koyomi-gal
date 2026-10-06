package handler

import (
	"errors"
	"strconv"

	"backend/internal/company/dto"
	"backend/internal/company/service"
	galgamedto "backend/internal/galgame/dto"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CompanyHandler struct{ companies *service.CompanyService }

func NewCompanyHandler(companies *service.CompanyService) *CompanyHandler {
	return &CompanyHandler{companies: companies}
}

// ListCompanies godoc
// @Summary List companies
// @Description Search names and aliases by literal case-insensitive substring. Counts include distinct published games only. Sort defaults to created_at descending; name is ascending and game_count descending.
// @ID listCompanies
// @Tags companies
// @Produce json
// @Param keyword query string false "Name or alias"
// @Param page query int false "Page" default(1)
// @Param page_size query int false "Page size (1-100)" default(20)
// @Param status query string false "Company status" Enums(active,inactive,closed)
// @Param country query string false "Country"
// @Param sort query string false "Sort" Enums(created_at,name,game_count)
// @Success 200 {object} dto.CompanyListResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/companies [get]
func (h *CompanyHandler) ListCompanies(c *gin.Context) { h.list(c, true) }

// GetCompany godoc
// @Summary Get company by slug
// @Description Includes aliases, parent and children summaries. Counts include distinct published games only. Dates are nullable YYYY-MM-DD strings.
// @ID getCompany
// @Tags companies
// @Produce json
// @Param slug path string true "Company slug"
// @Success 200 {object} dto.CompanyDataResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/companies/{slug} [get]
func (h *CompanyHandler) GetCompany(c *gin.Context) {
	company, err := h.companies.GetBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Ok(c, dto.NewCompanyResponse(company))
}

// ListCompanyGames godoc
// @Summary List published games associated with a company
// @Description A game appears once even when associated with multiple roles. Response pagination uses limit, matching the existing GalgameListData contract.
// @ID listCompanyGames
// @Tags companies
// @Produce json
// @Param slug path string true "Company slug"
// @Param page query int false "Page" default(1)
// @Param page_size query int false "Page size (1-100)" default(20)
// @Param role query string false "Association role" Enums(developer,publisher,brand,co_developer)
// @Param keyword query string false "Game title or alias"
// @Param sort query string false "Sort" Enums(latest,oldest,rating,favorite,popular)
// @Success 200 {object} galgamedto.GalgameListResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/companies/{slug}/games [get]
func (h *CompanyHandler) ListCompanyGames(c *gin.Context) {
	var query dto.CompanyGamesQuery
	if c.ShouldBindQuery(&query) != nil {
		response.Error(c, appErrors.ErrValidation("Invalid company games query"))
		return
	}
	var data galgamedto.GalgameListData
	data, err := h.companies.ListGames(c.Request.Context(), c.Param("slug"), query)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Ok(c, data)
}

// ListAdminCompanies godoc
// @Summary List companies for administration
// @Description Requires company:manage. Counts include games of every status. Sort defaults to created_at descending; name is ascending and game_count descending.
// @ID listAdminCompanies
// @Tags admin
// @Produce json
// @Param keyword query string false "Name or alias"
// @Param page query int false "Page" default(1)
// @Param page_size query int false "Page size (1-100)" default(20)
// @Param status query string false "Company status" Enums(active,inactive,closed)
// @Param country query string false "Country"
// @Param sort query string false "Sort" Enums(created_at,name,game_count)
// @Success 200 {object} dto.CompanyListResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/companies [get]
func (h *CompanyHandler) ListAdminCompanies(c *gin.Context) { h.list(c, false) }

func (h *CompanyHandler) list(c *gin.Context, public bool) {
	var query dto.CompanyQuery
	if c.ShouldBindQuery(&query) != nil {
		response.Error(c, appErrors.ErrValidation("Invalid company query"))
		return
	}
	data, err := h.companies.List(c.Request.Context(), query, public)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Ok(c, data)
}

// GetAdminCompany godoc
// @Summary Get company by ID for administration
// @Description Requires company:manage. Counts include games of every status.
// @ID getAdminCompany
// @Tags admin
// @Produce json
// @Param id path int true "Company ID"
// @Success 200 {object} dto.CompanyDataResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/companies/{id} [get]
func (h *CompanyHandler) GetAdminCompany(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	company, err := h.companies.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Ok(c, dto.NewCompanyResponse(company))
}

// CreateCompany godoc
// @Summary Create company
// @Description Requires company:manage. Slugs are trimmed and lowercase; the normalized slug must be nonempty, cannot be '.' or '..', and cannot contain '/' or '\\' or Unicode control characters. Spaces and Unicode characters are allowed for compatibility with existing developer slugs. Aliases are trimmed, deduplicated and empty aliases omitted. Dates must be nullable YYYY-MM-DD, closed_date cannot precede founded_date.
// @ID createCompany
// @Tags admin
// @Accept json
// @Produce json
// @Param request body dto.CompanyRequest true "Company metadata"
// @Success 200 {object} dto.CompanyDataResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/companies [post]
func (h *CompanyHandler) CreateCompany(c *gin.Context) { h.save(c, 0) }

// UpdateCompany godoc
// @Summary Replace company metadata and aliases
// @Description Requires company:manage. Full replacement; omitted optional fields reset. Slugs follow the same rules as creation: trim and lowercase, nonempty, not '.' or '..', no '/' or '\\' or Unicode control characters; spaces and Unicode characters are allowed. Parent must exist and cannot create a cycle. Dates must be nullable YYYY-MM-DD, closed_date cannot precede founded_date.
// @ID updateCompany
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Company ID"
// @Param request body dto.CompanyRequest true "Replacement company metadata"
// @Success 200 {object} dto.CompanyDataResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/companies/{id} [put]
func (h *CompanyHandler) UpdateCompany(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	h.save(c, id)
}

func (h *CompanyHandler) save(c *gin.Context, id uint) {
	var req dto.CompanyRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, appErrors.ErrValidation("Invalid company metadata"))
		return
	}
	company, err := h.companies.Save(c.Request.Context(), id, &req)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Ok(c, dto.NewCompanyResponse(company))
}

// DeleteCompany godoc
// @Summary Delete company
// @Description Requires company:manage. Removes aliases and game associations; children become parentless. Games and legacy developers are preserved.
// @ID deleteCompany
// @Tags admin
// @Produce json
// @Param id path int true "Company ID"
// @Success 200 {object} response.MessageResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/companies/{id} [delete]
func (h *CompanyHandler) DeleteCompany(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.companies.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	response.OkWithMsg(c, "success")
}

// ReplaceGameCompanies godoc
// @Summary Atomically replace all company associations of a game
// @Description Requires company:manage. Game and all companies must exist. Rejects duplicate company_id/role tuples and invalid roles. An empty companies array clears all associations; multiple distinct roles for one company are allowed.
// @ID replaceGameCompanies
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Game ID"
// @Param request body dto.ReplaceGameCompaniesRequest true "Complete company associations"
// @Success 200 {object} dto.GameCompaniesDataResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/games/{id}/companies [put]
func (h *CompanyHandler) ReplaceGameCompanies(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.ReplaceGameCompaniesRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, appErrors.ErrValidation("Invalid game companies"))
		return
	}
	data, err := h.companies.ReplaceGameCompanies(c.Request.Context(), id, &req)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Ok(c, data)
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		response.Error(c, appErrors.ErrValidation("Invalid ID"))
		return 0, false
	}
	return uint(id), true
}

func (h *CompanyHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCompanyNotFound), errors.Is(err, service.ErrGameNotFound):
		response.Error(c, appErrors.ErrNotFound(err.Error()))
	case errors.Is(err, service.ErrSlugExists):
		response.Error(c, appErrors.ErrConflict(err.Error()))
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrInvalidDate), errors.Is(err, service.ErrParentNotFound),
		errors.Is(err, service.ErrParentCycle), errors.Is(err, service.ErrUnknownCompanies), errors.Is(err, service.ErrDuplicateBinding):
		response.Error(c, appErrors.ErrValidation(err.Error()))
	default:
		logger.Error("company request", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("Company request failed"))
	}
}
