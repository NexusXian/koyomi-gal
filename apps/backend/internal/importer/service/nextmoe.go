package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	galgameModel "backend/internal/galgame/model"
	galgameRepository "backend/internal/galgame/repository"
	"backend/internal/infrastructures/nextmoe"
	"backend/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Chinese description outcomes reported by imports and previews.
const (
	ZhDescriptionImported     = "imported"
	ZhDescriptionUnchanged    = "unchanged"
	ZhDescriptionKeptExisting = "kept_existing"
	ZhDescriptionProtected    = "protected"
	ZhDescriptionAvailable    = "available"
	ZhDescriptionNotFound     = "not_found"
	ZhDescriptionAmbiguous    = "ambiguous"
	ZhDescriptionNoIntro      = "no_intro"
	ZhDescriptionDisabled     = "disabled"
	ZhDescriptionFailed       = "failed"
	ZhDescriptionSkipped      = "skipped"
)

const (
	nextMoeLookupTimeout = 30 * time.Second
	maxUpstreamSourceLen = 64
)

// NextMoeClient is the subset of the NextMoe catalog client the importer
// uses; tests substitute an httptest-backed client.
type NextMoeClient interface {
	FindWorksByVNDBIDs(ctx context.Context, vndbIDs []string, includeIntros bool) (map[string]nextmoe.Match, error)
	ChineseIntroForWork(ctx context.Context, work *nextmoe.Work) (*nextmoe.ChineseIntro, error)
}

// ZhDescriptionOptions controls how an import treats an existing Chinese
// description. OverwriteManual only takes effect together with Force.
type ZhDescriptionOptions struct {
	Force           bool
	OverwriteManual bool
}

// ZhDescriptionPreview is the NextMoe Chinese intro shown before importing.
type ZhDescriptionPreview struct {
	Status     string `json:"status"`
	Content    string `json:"content,omitempty"`
	SourceName string `json:"source_name,omitempty"`
}

// nextMoeLookup is the pre-fetched NextMoe result for one VNDB ID. Intro is
// set only when Status is available.
type nextMoeLookup struct {
	Status string
	Intro  *nextmoe.ChineseIntro
}

// SetNextMoeClient enables NextMoe Chinese intro imports for VNDB games.
func (s *Service) SetNextMoeClient(client NextMoeClient) {
	s.nextMoe = client
}

// lookupNextMoeZh fetches the Chinese intro for one VNDB ID outside of any
// database transaction. Failures only downgrade the status.
func (s *Service) lookupNextMoeZh(ctx context.Context, vndbID string) nextMoeLookup {
	lookups, _ := s.prefetchNextMoeZh(ctx, []string{vndbID})
	return lookups[vndbID]
}

// prefetchNextMoeZh resolves a set of VNDB IDs with one batch request and
// selects each work's Chinese intro. The returned error is non-nil only for
// failures that will repeat for every later request (bad key, quota).
func (s *Service) prefetchNextMoeZh(ctx context.Context, vndbIDs []string) (map[string]nextMoeLookup, error) {
	lookups := make(map[string]nextMoeLookup, len(vndbIDs))
	if s.nextMoe == nil {
		for _, id := range vndbIDs {
			lookups[id] = nextMoeLookup{Status: ZhDescriptionDisabled}
		}
		return lookups, nil
	}
	ctx, cancel := context.WithTimeout(ctx, nextMoeLookupTimeout+time.Duration(len(vndbIDs))*time.Second)
	defer cancel()

	matches, err := s.nextMoe.FindWorksByVNDBIDs(ctx, vndbIDs, true)
	if err != nil {
		logNextMoeFailure("NextMoe work lookup failed", vndbIDs, err)
		for _, id := range vndbIDs {
			lookups[id] = nextMoeLookup{Status: ZhDescriptionFailed}
		}
		return lookups, fatalNextMoeError(err)
	}
	var fatal error
	for _, id := range vndbIDs {
		match := matches[id]
		switch {
		case fatal != nil:
			lookups[id] = nextMoeLookup{Status: ZhDescriptionFailed}
		case errors.Is(match.Err, nextmoe.ErrAmbiguous):
			lookups[id] = nextMoeLookup{Status: ZhDescriptionAmbiguous}
		case errors.Is(match.Err, nextmoe.ErrInvalidID):
			lookups[id] = nextMoeLookup{Status: ZhDescriptionSkipped}
		case match.Work == nil:
			lookups[id] = nextMoeLookup{Status: ZhDescriptionNotFound}
		default:
			intro, err := s.nextMoe.ChineseIntroForWork(ctx, match.Work)
			switch {
			case err == nil:
				lookups[id] = nextMoeLookup{Status: ZhDescriptionAvailable, Intro: intro}
			case errors.Is(err, nextmoe.ErrNoIntro), errors.Is(err, nextmoe.ErrNotFound):
				lookups[id] = nextMoeLookup{Status: ZhDescriptionNoIntro}
			default:
				logNextMoeFailure("NextMoe intro lookup failed", []string{id}, err)
				lookups[id] = nextMoeLookup{Status: ZhDescriptionFailed}
				fatal = fatalNextMoeError(err)
			}
		}
	}
	return lookups, fatal
}

func fatalNextMoeError(err error) error {
	if errors.Is(err, nextmoe.ErrUnauthorized) || errors.Is(err, nextmoe.ErrQuotaExceeded) ||
		errors.Is(err, nextmoe.ErrNotConfigured) {
		return err
	}
	return nil
}

func logNextMoeFailure(message string, vndbIDs []string, err error) {
	fields := []zap.Field{zap.Int("count", len(vndbIDs)), zap.Error(err)}
	if len(vndbIDs) == 1 {
		fields = append(fields, zap.String("vndb_id", vndbIDs[0]))
	}
	logger.Warn(message, fields...)
}

// previewNextMoeZh returns the NextMoe intro preview for a VNDB game.
func (s *Service) previewNextMoeZh(ctx context.Context, vndbID string) *ZhDescriptionPreview {
	lookup := s.lookupNextMoeZh(ctx, vndbID)
	preview := &ZhDescriptionPreview{Status: lookup.Status}
	if lookup.Intro != nil {
		preview.Content = normalizeDescription(lookup.Intro.Content)
		preview.SourceName = galgameModel.DefaultDescriptionSourceName(galgameModel.DescriptionSourceNextMoe)
	}
	return preview
}

// syncNextMoeZh applies a looked-up intro to an existing galgame. A failed
// write is logged and reported, never propagated: VNDB data stays committed.
func (s *Service) syncNextMoeZh(
	ctx context.Context,
	galgameID uint,
	lookup nextMoeLookup,
	opts ZhDescriptionOptions,
) string {
	if lookup.Intro == nil {
		return lookup.Status
	}
	status, err := s.applyNextMoeZh(ctx, galgameID, lookup.Intro, opts)
	if err != nil {
		logger.Warn("apply NextMoe Chinese description failed",
			zap.Uint("galgame_id", galgameID), zap.Error(err))
		return ZhDescriptionFailed
	}
	return status
}

// zhDescriptionState is the current zh-CN description of a galgame.
type zhDescriptionState struct {
	Content          string
	SourceType       string
	SourceExternalID string
	UpstreamSource   string
	HasRow           bool
}

type zhIncoming struct {
	Content        string
	WorkID         string
	UpstreamSource string
}

// decideZhDescription decides whether a NextMoe intro may be written:
//
//  1. an empty current description is always filled;
//  2. identical content is left alone unless a forced refresh of an
//     automatic row changes its provenance;
//  3. automatic rows (NextMoe with a work ID, or VNDB/Bangumi/Steam
//     imports) are refreshed only with Force;
//  4. everything else counts as human-maintained (manual, official,
//     unknown, legacy column, NextMoe chosen in the form) and needs Force
//     plus OverwriteManual.
func decideZhDescription(current zhDescriptionState, incoming zhIncoming, opts ZhDescriptionOptions) (bool, string) {
	if incoming.Content == "" {
		return false, ZhDescriptionNoIntro
	}
	if strings.TrimSpace(current.Content) == "" {
		return true, ZhDescriptionImported
	}
	automatic := current.HasRow && isAutomaticZhRow(current)
	sameProvenance := current.SourceType == galgameModel.DescriptionSourceNextMoe &&
		current.SourceExternalID == incoming.WorkID &&
		current.UpstreamSource == incoming.UpstreamSource
	if current.Content == incoming.Content {
		if sameProvenance || !automatic || !opts.Force {
			return false, ZhDescriptionUnchanged
		}
		return true, ZhDescriptionImported
	}
	if automatic {
		if opts.Force {
			return true, ZhDescriptionImported
		}
		return false, ZhDescriptionKeptExisting
	}
	if opts.Force && opts.OverwriteManual {
		return true, ZhDescriptionImported
	}
	return false, ZhDescriptionProtected
}

func isAutomaticZhRow(current zhDescriptionState) bool {
	switch normalizeDescriptionSource(current.SourceType) {
	case galgameModel.DescriptionSourceNextMoe:
		return current.SourceExternalID != ""
	case galgameModel.DescriptionSourceVNDB,
		galgameModel.DescriptionSourceBangumi,
		galgameModel.DescriptionSourceSteam:
		return true
	default:
		return false
	}
}

// applyNextMoeZh writes the intro into the zh-CN row and the legacy column
// in a short transaction that locks the galgame row, so concurrent imports
// and edits see a consistent state.
func (s *Service) applyNextMoeZh(
	ctx context.Context,
	galgameID uint,
	intro *nextmoe.ChineseIntro,
	opts ZhDescriptionOptions,
) (string, error) {
	incoming := zhIncoming{
		Content:        normalizeDescription(intro.Content),
		WorkID:         intro.WorkID,
		UpstreamSource: truncateRunes(strings.TrimSpace(intro.Source), maxUpstreamSourceLen),
	}
	status := ZhDescriptionNoIntro
	err := s.repository.Transaction(ctx, func(tx *gorm.DB) error {
		var galgame galgameModel.Galgame
		err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "description", "description_source").
			First(&galgame, galgameID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrExistingGalgameNotFound
		}
		if err != nil {
			return fmt.Errorf("lock galgame for chinese description: %w", err)
		}
		current, err := loadZhDescriptionState(ctx, tx, &galgame)
		if err != nil {
			return err
		}
		write, decided := decideZhDescription(current, incoming, opts)
		status = decided
		if !write {
			return nil
		}
		row := galgameModel.GalgameDescription{
			GalgameID:        galgameID,
			Language:         galgameModel.LanguageZhCN,
			Content:          incoming.Content,
			SourceType:       galgameModel.DescriptionSourceNextMoe,
			SourceName:       galgameModel.DefaultDescriptionSourceName(galgameModel.DescriptionSourceNextMoe),
			SourceExternalID: incoming.WorkID,
			UpstreamSource:   incoming.UpstreamSource,
		}
		if err := galgameRepository.NewGalgameDescriptionRepository(tx).UpsertWithProvenance(ctx, &row); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&galgameModel.Galgame{}).
			Where("id = ?", galgameID).
			Updates(map[string]any{
				"description":        incoming.Content,
				"description_source": galgameModel.DescriptionSourceNextMoe,
				"updated_at":         time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("mirror chinese description: %w", err)
		}
		return nil
	})
	if err != nil {
		return ZhDescriptionFailed, err
	}
	return status, nil
}

func loadZhDescriptionState(
	ctx context.Context,
	tx *gorm.DB,
	galgame *galgameModel.Galgame,
) (zhDescriptionState, error) {
	var row galgameModel.GalgameDescription
	err := tx.WithContext(ctx).
		Where("galgame_id = ? AND language = ?", galgame.ID, galgameModel.LanguageZhCN).
		First(&row).Error
	if err == nil {
		return zhDescriptionState{
			Content:          row.Content,
			SourceType:       row.SourceType,
			SourceExternalID: row.SourceExternalID,
			UpstreamSource:   row.UpstreamSource,
			HasRow:           true,
		}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return zhDescriptionState{}, fmt.Errorf("find chinese description: %w", err)
	}
	// Without a row the legacy column holds Chinese text unless it is VNDB's.
	if normalizeDescriptionSource(galgame.DescriptionSource) == galgameModel.DescriptionSourceVNDB {
		return zhDescriptionState{}, nil
	}
	return zhDescriptionState{
		Content:    galgame.Description,
		SourceType: normalizeDescriptionSource(galgame.DescriptionSource),
	}, nil
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}
