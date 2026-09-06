package handler

import (
	"errors"
	"strconv"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/service"
	appErrors "backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CharacterHandler struct {
	characters *service.CharacterService
}

func NewCharacterHandler(characters *service.CharacterService) *CharacterHandler {
	return &CharacterHandler{characters: characters}
}

// ListGalgameCharacters godoc
// @Summary List characters of a published galgame
// @Description Sorted by sort_order ASC, relation id ASC. With spoiler=false (default), appearance spoilers contain only id (relation ID), role, appearance_spoiler, has_spoiler and sort_order. Visible entries never include spoiler_description unless spoiler=true. has_spoiler reflects appearance_spoiler, a non-none spoiler_level or nonempty spoiler_description. No global public character lookup exists. Responses must not be cached.
// @ID listGalgameCharacters
// @Tags galgames
// @Produce json
// @Param id path int true "Galgame ID"
// @Param spoiler query bool false "Explicitly reveal all character identity and spoiler text" default(false)
// @Success 200 {object} dto.GalgameCharacterListResponse
// @Header 200 {string} Cache-Control "private, no-store"
// @Failure 400 {object} response.ErrorResponse "Invalid ID or spoiler query"
// @Failure 404 {object} response.ErrorResponse "Galgame missing or not published"
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/galgames/{id}/characters [get]
func (h *CharacterHandler) ListGalgameCharacters(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	spoiler := c.DefaultQuery("spoiler", "false")
	if spoiler != "true" && spoiler != "false" {
		response.Error(c, appErrors.ErrValidation("spoiler must be true or false"))
		return
	}
	data, err := h.characters.ListPublishedCharacters(c.Request.Context(), id, spoiler == "true")
	if err != nil {
		h.respondCharacterError(c, err)
		return
	}
	response.Ok(c, data)
}

// ListAdminGalgameCharacters godoc
// @Summary List all galgame character associations for administration
// @Description Requires character:manage. Includes full identity and spoiler text for galgames in any status, including drafts. Sorted by sort_order ASC, relation id ASC. id is the relation ID; character_id is the shared character ID. Responses must not be cached.
// @ID listAdminGalgameCharacters
// @Tags admin
// @Produce json
// @Param id path int true "Galgame ID"
// @Success 200 {object} dto.GalgameCharacterListResponse
// @Header 200 {string} Cache-Control "private, no-store"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/galgames/{id}/characters [get]
func (h *CharacterHandler) ListAdminGalgameCharacters(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	data, err := h.characters.ListAdminCharacters(c.Request.Context(), id)
	if err != nil {
		h.respondCharacterError(c, err)
		return
	}
	response.Ok(c, data)
}

// SearchAdminCharacters godoc
// @Summary Search shared characters for administration
// @Description Requires character:manage. Case-insensitive literal substring search of name, original_name and source_id, ordered by id DESC. Omit q to list all. Responses must not be cached.
// @ID searchAdminCharacters
// @Tags admin
// @Produce json
// @Param q query string false "Search text (at most 255 characters)"
// @Param page query int false "Page (1-1000000)" default(1)
// @Param page_size query int false "Page size (1-100)" default(20)
// @Success 200 {object} dto.CharacterSearchResponse
// @Header 200 {string} Cache-Control "private, no-store"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/characters [get]
func (h *CharacterHandler) SearchAdminCharacters(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	var query dto.CharacterSearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, appErrors.ErrValidation("Invalid character search query"))
		return
	}
	data, err := h.characters.SearchCharacters(c.Request.Context(), query)
	if err != nil {
		h.respondCharacterError(c, err)
		return
	}
	response.Ok(c, data)
}

// CreateCharacter godoc
// @Summary Create or reuse a shared character
// @Description Requires character:manage. A nonempty source and source_id pair atomically reuses an existing character without overwriting any metadata. description must contain only non-spoiler public text. image_url may be empty or an HTTP/HTTPS URL without credentials or whitespace (maximum 2048 bytes); uploaded images use their existing public URL, not an asset ID. birthday is free-form month/day text; height is integer centimeters, 0 for unknown.
// @ID createCharacter
// @Tags admin
// @Accept json
// @Produce json
// @Param request body dto.CharacterRequest true "Shared character metadata; name is required"
// @Success 200 {object} dto.CharacterDataResponse "Created or existing unchanged character"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/characters [post]
func (h *CharacterHandler) CreateCharacter(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	var req dto.CharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("Invalid character metadata"))
		return
	}
	data, err := h.characters.CreateCharacter(c.Request.Context(), &req)
	if err != nil {
		h.respondCharacterError(c, err)
		return
	}
	response.Ok(c, data)
}

// UpdateCharacter godoc
// @Summary Replace shared character metadata
// @Description Requires character:manage. Full replacement: omitted optional fields reset to empty strings or 0. Changes affect every linked galgame. description must be spoiler-free. image_url accepts only an empty string or an HTTP/HTTPS URL without credentials or whitespace, at most 2048 bytes. birthday is free-form text; height is integer centimeters, 0 unknown. A conflicting nonempty source/source_id returns 409 and never merges characters.
// @ID updateCharacter
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Shared character ID"
// @Param request body dto.CharacterRequest true "Replacement metadata; name is required"
// @Success 200 {object} dto.CharacterDataResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/characters/{id} [put]
func (h *CharacterHandler) UpdateCharacter(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	id, ok := parseID(c, "Character")
	if !ok {
		return
	}
	var req dto.CharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("Invalid character metadata"))
		return
	}
	data, err := h.characters.UpdateCharacter(c.Request.Context(), id, &req)
	if err != nil {
		h.respondCharacterError(c, err)
		return
	}
	response.Ok(c, data)
}

// DeleteCharacter godoc
// @Summary Delete a shared character and all its galgame associations
// @Description Requires character:manage. Cascades through all galgame associations; does not delete the image URL's storage object or any galgames.
// @ID deleteCharacter
// @Tags admin
// @Produce json
// @Param id path int true "Shared character ID"
// @Success 200 {object} response.MessageResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/characters/{id} [delete]
func (h *CharacterHandler) DeleteCharacter(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	id, ok := parseID(c, "Character")
	if !ok {
		return
	}
	if err := h.characters.DeleteCharacter(c.Request.Context(), id); err != nil {
		h.respondCharacterError(c, err)
		return
	}
	response.OkWithMsg(c, "Character deleted")
}

// BindGalgameCharacter godoc
// @Summary Bind an existing character to a galgame
// @Description Requires character:manage. Works with drafts. character_id is the shared character ID. role defaults to other and accepts other/protagonist/main/supporting/guest; spoiler_level defaults to none and accepts none/minor/major. description is game-specific non-spoiler text; spoiler_description is revealed only with spoiler=true. appearance_spoiler conceals all identity fields by default. Returns full association metadata; id is the relation ID. Duplicate bindings return 409.
// @ID bindGalgameCharacter
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Galgame ID"
// @Param request body dto.GalgameCharacterRequest true "Character ID and association metadata"
// @Success 200 {object} dto.GalgameCharacterDataResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/galgames/{id}/characters [post]
func (h *CharacterHandler) BindGalgameCharacter(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	id, ok := parseID(c, "Galgame")
	if !ok {
		return
	}
	var req dto.GalgameCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("Invalid galgame character metadata"))
		return
	}
	data, err := h.characters.BindGalgameCharacter(c.Request.Context(), id, &req)
	if err != nil {
		h.respondCharacterError(c, err)
		return
	}
	response.Ok(c, data)
}

// UpdateGalgameCharacter godoc
// @Summary Replace a galgame character association's metadata
// @Description Requires character:manage. Full replacement, including drafts: omitted fields reset to defaults (role=other, spoiler_level=none, false, empty strings, sort_order=0). characterId is the shared character ID, not the relation ID. Neither parent ID nor shared character metadata changes. description must be game-specific non-spoiler text; put spoilers in spoiler_description. Returns full association metadata, including spoiler text.
// @ID updateGalgameCharacter
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Galgame ID"
// @Param characterId path int true "Shared character ID, not relation ID"
// @Param request body dto.UpdateGalgameCharacterRequest true "Replacement association metadata"
// @Success 200 {object} dto.GalgameCharacterDataResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/galgames/{id}/characters/{characterId} [put]
func (h *CharacterHandler) UpdateGalgameCharacter(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	galgameID, characterID, ok := parseGalgameCharacterIDs(c)
	if !ok {
		return
	}
	var req dto.UpdateGalgameCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ErrValidation("Invalid galgame character metadata"))
		return
	}
	data, err := h.characters.UpdateGalgameCharacter(c.Request.Context(), galgameID, characterID, &req)
	if err != nil {
		h.respondCharacterError(c, err)
		return
	}
	response.Ok(c, data)
}

// UnbindGalgameCharacter godoc
// @Summary Unbind a character from one galgame
// @Description Requires character:manage. characterId is the shared character ID, not the relation ID. Only deletes the scoped association; preserves the shared character, all other associations and image storage.
// @ID unbindGalgameCharacter
// @Tags admin
// @Produce json
// @Param id path int true "Galgame ID"
// @Param characterId path int true "Shared character ID, not relation ID"
// @Success 200 {object} response.MessageResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/admin/galgames/{id}/characters/{characterId} [delete]
func (h *CharacterHandler) UnbindGalgameCharacter(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	galgameID, characterID, ok := parseGalgameCharacterIDs(c)
	if !ok {
		return
	}
	if err := h.characters.UnbindGalgameCharacter(c.Request.Context(), galgameID, characterID); err != nil {
		h.respondCharacterError(c, err)
		return
	}
	response.OkWithMsg(c, "Character unbound")
}

func (h *CharacterHandler) respondCharacterError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCharacterNotFound), errors.Is(err, service.ErrCharacterGalgameNotFound):
		response.Error(c, appErrors.ErrNotFound(err.Error()))
	case errors.Is(err, service.ErrCharacterInvalid), errors.Is(err, service.ErrCharacterInvalidURL):
		response.Error(c, appErrors.ErrValidation(err.Error()))
	case errors.Is(err, service.ErrCharacterConflict):
		response.Error(c, appErrors.ErrConflict(err.Error()))
	default:
		logger.Error("character operation failed", zap.Error(err))
		response.Error(c, appErrors.ErrInternal("Character operation failed"))
	}
}

func parseGalgameCharacterIDs(c *gin.Context) (uint, uint, bool) {
	galgameID, ok := parseID(c, "Galgame")
	if !ok {
		return 0, 0, false
	}
	characterID, err := strconv.ParseUint(c.Param("characterId"), 10, 0)
	if err != nil || characterID == 0 {
		response.Error(c, appErrors.ErrValidation("Invalid character ID"))
		return 0, 0, false
	}
	return galgameID, uint(characterID), true
}
