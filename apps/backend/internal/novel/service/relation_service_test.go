package service

import (
	"context"
	"errors"
	"testing"

	"backend/internal/novel/dto"
	"backend/internal/novel/model"
	relationRepo "backend/internal/relation/repository"
	"backend/internal/testutil"
)

func TestRelationCreateAndDuplicateGuard(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "relation-create-creator")

	novel := createTestNovel(t, env.novels, creator, "relation-novel", model.NovelStatusPublished)
	otherNovel := createTestNovel(t, env.novels, creator, "relation-other", model.NovelStatusPublished)
	galgameID := createTestGalgameForNovel(t, env.catalog, creator, "relation-galgame")

	// galgame target
	relation, err := env.relations.CreateRelation(ctx, creator, novel.ID, &dto.CreateRelationRequest{
		TargetType:   "galgame",
		TargetID:     galgameID,
		RelationType: "adaptation",
	})
	if err != nil {
		t.Fatalf("create relation: %v", err)
	}
	if relation.SourceType != "novel" || relation.SourceID != novel.ID {
		t.Fatalf("relation should be directed from novel: %+v", relation)
	}

	// duplicate in either direction is rejected
	if _, err := env.relations.CreateRelation(ctx, creator, novel.ID, &dto.CreateRelationRequest{
		TargetType: "galgame", TargetID: galgameID, RelationType: "adaptation",
	}); !errors.Is(err, ErrRelationExists) {
		t.Fatalf("expected ErrRelationExists for exact duplicate, got %v", err)
	}
	// reverse direction (source already stored as other -> novel by another user)
	if _, err := env.relations.CreateRelation(ctx, creator, novel.ID, &dto.CreateRelationRequest{
		TargetType: "novel", TargetID: otherNovel.ID, RelationType: "adaptation",
	}); err != nil {
		t.Fatalf("create novel-to-novel relation: %v", err)
	}
	if _, err := env.relations.CreateRelation(ctx, creator, otherNovel.ID, &dto.CreateRelationRequest{
		TargetType: "novel", TargetID: novel.ID, RelationType: "adaptation",
	}); !errors.Is(err, ErrRelationExists) {
		t.Fatalf("expected ErrRelationExists for reverse duplicate, got %v", err)
	}

	// relation creations are credited as contributions to the source novel
	assertContributionRows(t, env.db, novel.ID, 3) // novel create + 2 relations
}

func TestRelationValidation(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "relation-validate-creator")

	novel := createTestNovel(t, env.novels, creator, "relation-validate", model.NovelStatusPublished)
	hiddenNovel := createTestNovel(t, env.novels, creator, "relation-hidden", model.NovelStatusPending)

	// self relation
	if _, err := env.relations.CreateRelation(ctx, creator, novel.ID, &dto.CreateRelationRequest{
		TargetType: "novel", TargetID: novel.ID, RelationType: "related",
	}); !errors.Is(err, ErrInvalidRelationInput) {
		t.Fatalf("expected ErrInvalidRelationInput for self relation, got %v", err)
	}
	// unpublished targets are rejected
	if _, err := env.relations.CreateRelation(ctx, creator, novel.ID, &dto.CreateRelationRequest{
		TargetType: "novel", TargetID: hiddenNovel.ID, RelationType: "related",
	}); !errors.Is(err, ErrRelationTargetAbsent) {
		t.Fatalf("expected ErrRelationTargetAbsent for hidden novel, got %v", err)
	}
	if _, err := env.relations.CreateRelation(ctx, creator, novel.ID, &dto.CreateRelationRequest{
		TargetType: "galgame", TargetID: 999999, RelationType: "related",
	}); !errors.Is(err, ErrRelationTargetAbsent) {
		t.Fatalf("expected ErrRelationTargetAbsent for missing galgame, got %v", err)
	}
	// unknown relation types are rejected
	if _, err := env.relations.CreateRelation(ctx, creator, novel.ID, &dto.CreateRelationRequest{
		TargetType: "novel", TargetID: 1, RelationType: "remake",
	}); !errors.Is(err, ErrInvalidRelationInput) {
		t.Fatalf("expected ErrInvalidRelationInput for unknown type, got %v", err)
	}
	// a pending novel may still add relations as source (editors prepare first)
	if _, err := env.relations.CreateRelation(ctx, creator, hiddenNovel.ID, &dto.CreateRelationRequest{
		TargetType: "novel", TargetID: novel.ID, RelationType: "sequel",
	}); err != nil {
		t.Fatalf("pending novel should allow adding relations: %v", err)
	}
}

func TestRelationDeleteScopedToNovel(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "relation-delete-creator")

	novelA := createTestNovel(t, env.novels, creator, "relation-a", model.NovelStatusPublished)
	novelB := createTestNovel(t, env.novels, creator, "relation-b", model.NovelStatusPublished)
	novelC := createTestNovel(t, env.novels, creator, "relation-c", model.NovelStatusPublished)

	relation, err := env.relations.CreateRelation(ctx, creator, novelA.ID, &dto.CreateRelationRequest{
		TargetType: "novel", TargetID: novelB.ID, RelationType: "same_series",
	})
	if err != nil {
		t.Fatalf("create relation: %v", err)
	}

	// deleting through an unrelated novel fails
	if err := env.relations.DeleteRelation(ctx, novelC.ID, relation.ID); !errors.Is(err, ErrRelationNotFound) {
		t.Fatalf("expected ErrRelationNotFound via unrelated novel, got %v", err)
	}
	// deleting through the source novel works
	if err := env.relations.DeleteRelation(ctx, novelA.ID, relation.ID); err != nil {
		t.Fatalf("delete relation: %v", err)
	}
	if err := env.relations.DeleteRelation(ctx, novelA.ID, relation.ID); !errors.Is(err, ErrRelationNotFound) {
		t.Fatalf("expected ErrRelationNotFound after delete, got %v", err)
	}

	// a relation created by novel B can be removed from novel A's page as well
	// when it points at A (bidirectional management)
	incoming, err := env.relations.CreateRelation(ctx, creator, novelB.ID, &dto.CreateRelationRequest{
		TargetType: "novel", TargetID: novelA.ID, RelationType: "spin_off",
	})
	if err != nil {
		t.Fatalf("create incoming relation: %v", err)
	}
	if err := env.relations.DeleteRelation(ctx, novelA.ID, incoming.ID); err != nil {
		t.Fatalf("delete incoming relation from target side: %v", err)
	}
}

func TestGalgameRelationBidirectionalQuery(t *testing.T) {
	env := newNovelTestServices(t)
	ctx := context.Background()
	creator := testutil.CreateUser(t, env.db, "relation-bi-creator")

	novel := createTestNovel(t, env.novels, creator, "relation-bi-novel", model.NovelStatusPublished)
	galgameID := createTestGalgameForNovel(t, env.catalog, creator, "relation-bi-galgame")

	if _, err := env.relations.CreateRelation(ctx, creator, novel.ID, &dto.CreateRelationRequest{
		TargetType: "galgame", TargetID: galgameID, RelationType: "original",
	}); err != nil {
		t.Fatalf("create relation: %v", err)
	}

	// the galgame side resolves the novel even though the row points novel->galgame
	related, err := relationRepo.NewRelationRepository(env.db).ListRelatedNovelsForGalgame(ctx, galgameID)
	if err != nil {
		t.Fatalf("query related novels: %v", err)
	}
	if len(related) != 1 || related[0].WorkID != novel.ID {
		t.Fatalf("expected 1 related novel, got %+v", related)
	}
}
