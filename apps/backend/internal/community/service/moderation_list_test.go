package service

import (
	"context"
	"testing"

	"backend/internal/community/dto"
	"backend/internal/testutil"
)

func TestCommunityAdminListsSearchAndPaginate(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	author := testutil.CreateUser(t, env.db, "ModerationAuthor")
	other := testutil.CreateUser(t, env.db, "OtherAuthor")
	galgameID := env.createPublishedGalgame(t, author, "Moderation Game")
	first := env.createPost(t, author, "First moderation post", &galgameID)
	second := env.createPost(t, other, "Second post", nil)
	env.createComment(t, author, first.ID, "review this comment", nil, nil)
	env.createComment(t, other, second.ID, "ordinary comment", nil, nil)

	posts, total, page, limit, err := env.posts.ListAdmin(ctx, &dto.AdminCommunityQuery{
		Keyword: "moderationauthor",
	})
	if err != nil || total != 1 || len(posts) != 1 || posts[0].ID != first.ID || posts[0].AuthorName != "ModerationAuthor" {
		t.Fatalf("post author search: total=%d page=%d limit=%d posts=%+v err=%v", total, page, limit, posts, err)
	}
	posts, total, page, limit, err = env.posts.ListAdmin(ctx, &dto.AdminCommunityQuery{Page: 2, Limit: 1})
	if err != nil || total != 2 || page != 2 || limit != 1 || len(posts) != 1 || posts[0].ID != first.ID {
		t.Fatalf("post pagination: total=%d page=%d limit=%d posts=%+v err=%v", total, page, limit, posts, err)
	}
	posts, total, _, _, err = env.posts.ListAdmin(ctx, &dto.AdminCommunityQuery{Keyword: "MODERATION GAME"})
	if err != nil || total != 1 || len(posts) != 1 || posts[0].GalgameTitle != "Moderation Game" {
		t.Fatalf("post galgame search: total=%d posts=%+v err=%v", total, posts, err)
	}

	comments, total, page, limit, err := env.comments.ListAdmin(ctx, &dto.AdminCommunityQuery{
		Keyword: "FIRST MODERATION POST",
	})
	if err != nil || total != 1 || len(comments) != 1 || comments[0].PostTitle != first.Title {
		t.Fatalf("comment post search: total=%d page=%d limit=%d comments=%+v err=%v", total, page, limit, comments, err)
	}
	comments, total, _, _, err = env.comments.ListAdmin(ctx, &dto.AdminCommunityQuery{Keyword: "REVIEW THIS"})
	if err != nil || total != 1 || len(comments) != 1 || comments[0].AuthorName != "ModerationAuthor" {
		t.Fatalf("comment content search: total=%d comments=%+v err=%v", total, comments, err)
	}
}
