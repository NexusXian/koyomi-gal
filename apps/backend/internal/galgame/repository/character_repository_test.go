package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"backend/internal/galgame/model"
	"backend/internal/testutil"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func TestCharacterRepositoryCreateFindAndReuse(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := NewCharacterRepository(db)
	ctx := context.Background()
	created, err := repo.CreateOrReuse(ctx, &model.Character{
		Name: "Original", OriginalName: "Original name", Description: "Public biography",
		ImageURL: "https://example.com/original.png", Gender: "female", Birthday: "03-14",
		BloodType: "AB", Height: 165, Source: "vndb", SourceID: "c123",
	})
	if err != nil || created == nil || created.ID == 0 {
		t.Fatalf("create character: %+v, err=%v", created, err)
	}
	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("find character: %v", err)
	}
	if found.Name != created.Name || found.OriginalName != created.OriginalName ||
		found.Description != created.Description || found.ImageURL != created.ImageURL ||
		found.Gender != created.Gender || found.Birthday != created.Birthday ||
		found.BloodType != created.BloodType || found.Height != created.Height ||
		found.Source != created.Source || found.SourceID != created.SourceID ||
		found.CreatedAt.IsZero() || found.UpdatedAt.IsZero() {
		t.Fatalf("metadata did not round trip: %+v", found)
	}
	reused, err := repo.CreateOrReuse(ctx, &model.Character{
		Name: "Replacement", OriginalName: "Replacement name", Description: "Replacement biography",
		ImageURL: "https://example.com/replacement.png", Gender: "male", Birthday: "12-31",
		BloodType: "O", Height: 180, Source: "vndb", SourceID: "c123",
	})
	if err != nil || !reflect.DeepEqual(reused, found) {
		t.Fatalf("reuse must return unchanged metadata: got=%+v want=%+v err=%v", reused, found, err)
	}
	persisted, err := repo.FindByID(ctx, created.ID)
	if err != nil || !reflect.DeepEqual(persisted, found) {
		t.Fatalf("reuse overwrote stored metadata: got=%+v want=%+v err=%v", persisted, found, err)
	}
	var count int64
	if err := db.Model(&model.Character{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("reuse must not insert a second character: count=%d err=%v", count, err)
	}
	if _, err := repo.FindByID(ctx, created.ID+1000); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing character: got %v", err)
	}
}

func TestCharacterRepositoryPartialSourceUniqueness(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := NewCharacterRepository(db)
	ctx := context.Background()
	for _, pair := range []struct{ source, id string }{
		{"", ""}, {"vndb", ""}, {"", "c123"},
	} {
		t.Run(fmt.Sprintf("source=%s/id=%s", pair.source, pair.id), func(t *testing.T) {
			first, err := repo.CreateOrReuse(ctx, &model.Character{Name: "Same name", Source: pair.source, SourceID: pair.id})
			if err != nil {
				t.Fatal(err)
			}
			second, err := repo.CreateOrReuse(ctx, &model.Character{Name: "Same name", Source: pair.source, SourceID: pair.id})
			if err != nil || second == nil || second.ID == first.ID {
				t.Fatalf("incomplete source pairs must not deduplicate: first=%+v second=%+v err=%v", first, second, err)
			}
		})
	}
	ids := make(map[uint]bool)
	for _, pair := range []struct{ source, id string }{
		{"vndb", "c123"}, {"bangumi", "c123"}, {"vndb", "c124"},
	} {
		created, err := repo.CreateOrReuse(ctx, &model.Character{Name: "Same name", Source: pair.source, SourceID: pair.id})
		if err != nil || created == nil || ids[created.ID] {
			t.Fatalf("distinct source pairs must not deduplicate: %+v err=%v", created, err)
		}
		ids[created.ID] = true
	}
	err := db.Create(&model.Character{Name: "Bypass repository", Source: "vndb", SourceID: "c123"}).Error
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "uk_characters_source_source_id" {
		t.Fatalf("database must enforce complete source pair uniqueness: %v", err)
	}
}

func TestCharacterRepositoryConcurrentCreateOrReuse(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	repo := NewCharacterRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const workers = 8
	start := make(chan struct{})
	results := make([]*model.Character, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i], errs[i] = repo.CreateOrReuse(ctx, &model.Character{
				Name: fmt.Sprintf("Importer %d", i), Description: fmt.Sprintf("Biography %d", i),
				Source: "vndb", SourceID: "concurrent",
			})
		}()
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil || results[i] == nil || results[i].ID == 0 {
			t.Fatalf("importer %d: result=%+v err=%v", i, results[i], err)
		}
	}
	winner, err := repo.FindByID(ctx, results[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for i, result := range results {
		if result.ID != winner.ID || result.Name != winner.Name || result.Description != winner.Description {
			t.Fatalf("importer %d did not reuse the winner: got=%+v want=%+v", i, result, winner)
		}
	}
	var count int64
	if err := db.Model(&model.Character{}).Where("source = ? AND source_id = ?", "vndb", "concurrent").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("concurrent imports must create exactly one row: count=%d err=%v", count, err)
	}
}

func TestCharacterRepositoryAssociationLifecycle(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	for _, action := range []string{"unbind", "delete game", "delete character"} {
		t.Run(action, func(t *testing.T) {
			db := testutil.NewPostgres(t)
			repo := NewCharacterRepository(db)
			ctx := context.Background()
			games := []model.Galgame{
				{Title: "First", Slug: "character-first"},
				{Title: "Second", Slug: "character-second"},
			}
			if err := db.Create(&games).Error; err != nil {
				t.Fatalf("create games: %v", err)
			}
			character, err := repo.CreateOrReuse(ctx, &model.Character{Name: "Shared", Source: "vndb", SourceID: "c123"})
			if err != nil {
				t.Fatal(err)
			}
			for i, game := range games {
				relation := &model.GalgameCharacter{
					GalgameID: game.ID, CharacterID: character.ID, Role: model.CharacterRoleMain,
					Description: game.Title, SortOrder: i,
					Character: &model.Character{ID: character.ID, Name: "Must not overwrite shared metadata"},
				}
				if err := repo.Bind(ctx, relation); err != nil {
					t.Fatalf("bind game %d: %v", game.ID, err)
				}
				found, err := repo.FindRelation(ctx, game.ID, character.ID)
				if err != nil || found == nil || found.ID != relation.ID || found.Character == nil ||
					found.Character.Name != "Shared" || found.CharacterID != character.ID ||
					found.Description != game.Title || found.Role != model.CharacterRoleMain || found.SortOrder != i {
					t.Fatalf("find preloaded relation: %+v err=%v", found, err)
				}
			}
			err = repo.Bind(ctx, &model.GalgameCharacter{GalgameID: games[0].ID, CharacterID: character.ID})
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "uk_galgame_characters_galgame_character" {
				t.Fatalf("duplicate association must violate unique constraint: %v", err)
			}
			switch action {
			case "unbind":
				err = repo.Unbind(ctx, games[0].ID, character.ID)
			case "delete game":
				err = NewGalgameRepository(db).Delete(ctx, games[0].ID)
			case "delete character":
				err = repo.Delete(ctx, character.ID)
			}
			if err != nil {
				t.Fatalf("%s: %v", action, err)
			}
			if _, err := repo.FindRelation(ctx, games[0].ID, character.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("first association must be deleted: %v", err)
			}
			remaining, err := repo.ListByGalgameID(ctx, games[1].ID)
			if err != nil {
				t.Fatal(err)
			}
			_, characterErr := repo.FindByID(ctx, character.ID)
			if action == "delete character" {
				if len(remaining) != 0 || !errors.Is(characterErr, gorm.ErrRecordNotFound) {
					t.Fatalf("character deletion must cascade all associations: remaining=%+v err=%v", remaining, characterErr)
				}
			} else if len(remaining) != 1 || remaining[0].CharacterID != character.ID || characterErr != nil {
				t.Fatalf("shared character and other binding must survive: remaining=%+v err=%v", remaining, characterErr)
			}
			var gameCount int64
			wantGames := int64(2)
			if action == "delete game" {
				wantGames = 1
			}
			if err := db.Model(&model.Galgame{}).Count(&gameCount).Error; err != nil || gameCount != wantGames {
				t.Fatalf("unexpected surviving games: got=%d want=%d err=%v", gameCount, wantGames, err)
			}
		})
	}
}
