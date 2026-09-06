package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"backend/internal/galgame/dto"
	"backend/internal/galgame/model"
	"backend/internal/galgame/repository"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrCharacterNotFound        = errors.New("character or galgame character not found")
	ErrCharacterGalgameNotFound = errors.New("galgame not found")
	ErrCharacterInvalid         = errors.New("invalid character metadata, role or spoiler level")
	ErrCharacterInvalidURL      = errors.New("image_url must be an HTTP or HTTPS URL without credentials or whitespace, at most 2048 bytes")
	ErrCharacterConflict        = errors.New("character source or galgame association already exists")
)

type CharacterService struct {
	galgames   *repository.GalgameRepository
	characters *repository.CharacterRepository
}

func NewCharacterService(galgames *repository.GalgameRepository, characters *repository.CharacterRepository) *CharacterService {
	return &CharacterService{galgames: galgames, characters: characters}
}

// ProjectGalgameCharacter is the sole whitelist for public and admin association responses.
func ProjectGalgameCharacter(relation *model.GalgameCharacter, spoiler bool) dto.GalgameCharacterResponse {
	data := dto.GalgameCharacterResponse{
		ID:                relation.ID,
		Role:              relation.Role.String(),
		AppearanceSpoiler: relation.AppearanceSpoiler,
		HasSpoiler:        relation.AppearanceSpoiler || relation.SpoilerLevel != model.SpoilerLevelNone || relation.SpoilerDescription != "",
		SortOrder:         relation.SortOrder,
	}
	if (!spoiler && relation.AppearanceSpoiler) || relation.Character == nil {
		return data
	}
	character := relation.Character
	level := relation.SpoilerLevel.String()
	data.CharacterID = &relation.CharacterID
	data.Name = &character.Name
	data.OriginalName = &character.OriginalName
	data.ImageURL = &character.ImageURL
	data.Description = &relation.Description
	data.PublicDescription = &character.Description
	data.Gender = &character.Gender
	data.Birthday = &character.Birthday
	data.BloodType = &character.BloodType
	data.Height = &character.Height
	data.Source = &character.Source
	data.SourceID = &character.SourceID
	data.SpoilerLevel = &level
	data.CreatedAt = &relation.CreatedAt
	data.UpdatedAt = &relation.UpdatedAt
	if spoiler {
		data.SpoilerDescription = &relation.SpoilerDescription
	}
	return data
}

func (s *CharacterService) ListPublishedCharacters(ctx context.Context, galgameID uint, spoiler bool) (dto.GalgameCharacterListData, error) {
	galgame, err := s.galgames.FindPublishedByID(ctx, galgameID)
	if err != nil {
		return dto.GalgameCharacterListData{}, err
	}
	if galgame == nil {
		return dto.GalgameCharacterListData{}, ErrCharacterGalgameNotFound
	}
	return s.listCharacters(ctx, galgameID, spoiler)
}

func (s *CharacterService) ListAdminCharacters(ctx context.Context, galgameID uint) (dto.GalgameCharacterListData, error) {
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		return dto.GalgameCharacterListData{}, err
	}
	if galgame == nil {
		return dto.GalgameCharacterListData{}, ErrCharacterGalgameNotFound
	}
	return s.listCharacters(ctx, galgameID, true)
}

func (s *CharacterService) listCharacters(ctx context.Context, galgameID uint, spoiler bool) (dto.GalgameCharacterListData, error) {
	relations, err := s.characters.ListByGalgameID(ctx, galgameID)
	if err != nil {
		return dto.GalgameCharacterListData{}, err
	}
	items := make([]dto.GalgameCharacterResponse, 0, len(relations))
	for i := range relations {
		items = append(items, ProjectGalgameCharacter(&relations[i], spoiler))
	}
	return dto.GalgameCharacterListData{Items: items}, nil
}

func (s *CharacterService) SearchCharacters(ctx context.Context, query dto.CharacterSearchQuery) (dto.CharacterSearchData, error) {
	page, pageSize := query.Page, query.PageSize
	if page < 1 {
		page = 1
	}
	if page > 1000000 {
		page = 1000000
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	characters, total, err := s.characters.Search(ctx, strings.TrimSpace(query.Q), page, pageSize)
	if err != nil {
		return dto.CharacterSearchData{}, err
	}
	items := make([]dto.CharacterResponse, 0, len(characters))
	for i := range characters {
		items = append(items, characterResponse(&characters[i]))
	}
	return dto.CharacterSearchData{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *CharacterService) CreateCharacter(ctx context.Context, req *dto.CharacterRequest) (*dto.CharacterResponse, error) {
	character, err := characterFromRequest(req)
	if err != nil {
		return nil, err
	}
	created, err := s.characters.CreateOrReuse(ctx, character)
	if err != nil {
		return nil, characterWriteError(err)
	}
	data := characterResponse(created)
	return &data, nil
}

func (s *CharacterService) UpdateCharacter(ctx context.Context, id uint, req *dto.CharacterRequest) (*dto.CharacterResponse, error) {
	character, err := characterFromRequest(req)
	if err != nil {
		return nil, err
	}
	character.ID = id
	if id == 0 {
		return nil, ErrCharacterNotFound
	}
	if err := s.characters.Update(ctx, character); err != nil {
		return nil, characterWriteError(err)
	}
	data := characterResponse(character)
	return &data, nil
}

func (s *CharacterService) DeleteCharacter(ctx context.Context, id uint) error {
	return characterWriteError(s.characters.Delete(ctx, id))
}

func (s *CharacterService) BindGalgameCharacter(ctx context.Context, galgameID uint, req *dto.GalgameCharacterRequest) (*dto.GalgameCharacterResponse, error) {
	relation, err := relationFromRequest(&req.UpdateGalgameCharacterRequest)
	if err != nil {
		return nil, err
	}
	galgame, err := s.galgames.FindByID(ctx, galgameID)
	if err != nil {
		return nil, err
	}
	if galgame == nil {
		return nil, ErrCharacterGalgameNotFound
	}
	character, err := s.characters.FindByID(ctx, req.CharacterID)
	if err != nil {
		return nil, characterWriteError(err)
	}
	relation.GalgameID = galgameID
	relation.CharacterID = req.CharacterID
	relation.Character = character
	if err := s.characters.Bind(ctx, relation); err != nil {
		return nil, characterWriteError(err)
	}
	data := ProjectGalgameCharacter(relation, true)
	return &data, nil
}

func (s *CharacterService) UpdateGalgameCharacter(ctx context.Context, galgameID, characterID uint, req *dto.UpdateGalgameCharacterRequest) (*dto.GalgameCharacterResponse, error) {
	relation, err := relationFromRequest(req)
	if err != nil {
		return nil, err
	}
	existing, err := s.characters.FindRelation(ctx, galgameID, characterID)
	if err != nil {
		return nil, characterWriteError(err)
	}
	relation.ID = existing.ID
	relation.GalgameID = galgameID
	relation.CharacterID = characterID
	relation.Character = existing.Character
	if err := s.characters.UpdateRelation(ctx, relation); err != nil {
		return nil, characterWriteError(err)
	}
	data := ProjectGalgameCharacter(relation, true)
	return &data, nil
}

func (s *CharacterService) UnbindGalgameCharacter(ctx context.Context, galgameID, characterID uint) error {
	return characterWriteError(s.characters.Unbind(ctx, galgameID, characterID))
}

func characterFromRequest(req *dto.CharacterRequest) (*model.Character, error) {
	character := &model.Character{
		Name: strings.TrimSpace(req.Name), OriginalName: strings.TrimSpace(req.OriginalName),
		Description: strings.TrimSpace(req.Description), ImageURL: req.ImageURL,
		Gender: strings.TrimSpace(req.Gender), Birthday: strings.TrimSpace(req.Birthday),
		BloodType: strings.TrimSpace(req.BloodType), Height: req.Height,
		Source: strings.TrimSpace(req.Source), SourceID: strings.TrimSpace(req.SourceID),
	}
	if character.Name == "" || character.Height < 0 || character.Height > 2147483647 {
		return nil, ErrCharacterInvalid
	}
	for _, value := range []string{character.Name, character.OriginalName, character.SourceID} {
		if utf8.RuneCountInString(value) > 255 {
			return nil, ErrCharacterInvalid
		}
	}
	for _, value := range []string{character.Gender, character.Birthday, character.BloodType, character.Source} {
		if utf8.RuneCountInString(value) > 50 {
			return nil, ErrCharacterInvalid
		}
	}
	if character.ImageURL != "" {
		if validateExternalImageURL(character.ImageURL) != nil {
			return nil, ErrCharacterInvalidURL
		}
		parsed, _ := url.Parse(character.ImageURL)
		if parsed.Hostname() == "" {
			return nil, ErrCharacterInvalidURL
		}
	}
	return character, nil
}

func relationFromRequest(req *dto.UpdateGalgameCharacterRequest) (*model.GalgameCharacter, error) {
	relation := &model.GalgameCharacter{
		AppearanceSpoiler: req.AppearanceSpoiler, Description: strings.TrimSpace(req.Description),
		SpoilerDescription: strings.TrimSpace(req.SpoilerDescription), SortOrder: req.SortOrder,
	}
	switch req.Role {
	case "", "other":
		relation.Role = model.CharacterRoleOther
	case "protagonist":
		relation.Role = model.CharacterRoleProtagonist
	case "main":
		relation.Role = model.CharacterRoleMain
	case "supporting":
		relation.Role = model.CharacterRoleSupporting
	case "guest":
		relation.Role = model.CharacterRoleGuest
	default:
		return nil, ErrCharacterInvalid
	}
	switch req.SpoilerLevel {
	case "", "none":
		relation.SpoilerLevel = model.SpoilerLevelNone
	case "minor":
		relation.SpoilerLevel = model.SpoilerLevelMinor
	case "major":
		relation.SpoilerLevel = model.SpoilerLevelMajor
	default:
		return nil, ErrCharacterInvalid
	}
	if req.SortOrder < -2147483648 || req.SortOrder > 2147483647 {
		return nil, ErrCharacterInvalid
	}
	return relation, nil
}

func characterResponse(character *model.Character) dto.CharacterResponse {
	return dto.CharacterResponse{
		ID: character.ID, Name: character.Name, OriginalName: character.OriginalName,
		Description: character.Description, ImageURL: character.ImageURL, Gender: character.Gender,
		Birthday: character.Birthday, BloodType: character.BloodType, Height: character.Height,
		Source: character.Source, SourceID: character.SourceID,
		CreatedAt: character.CreatedAt, UpdatedAt: character.UpdatedAt,
	}
}

func characterWriteError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrCharacterNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrCharacterConflict
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrCharacterConflict
		case "23503":
			return ErrCharacterNotFound
		}
	}
	return err
}
