package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"backend/internal/community/dto"
	"backend/internal/community/model"
	communityRepo "backend/internal/community/repository"
	galgameDTO "backend/internal/galgame/dto"
	galgameRepo "backend/internal/galgame/repository"
	galgameService "backend/internal/galgame/service"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"

	"gorm.io/gorm"
)

type communityTestEnv struct {
	posts        *PostService
	comments     *CommentService
	interactions *InteractionService
	catalog      *galgameService.CatalogService
	rbac         *rbacService.RBACService
	db           *gorm.DB
}

func newCommunityTestEnv(t *testing.T) *communityTestEnv {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	galgameRepository := galgameRepo.NewGalgameRepository(db)
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	postRepo := communityRepo.NewPostRepository(db)
	commentRepo := communityRepo.NewCommentRepository(db)
	return &communityTestEnv{
		posts:        NewPostService(postRepo, galgameRepository, rbacSvc),
		comments:     NewCommentService(commentRepo, postRepo, rbacSvc),
		interactions: NewInteractionService(postRepo, commentRepo),
		catalog: galgameService.NewCatalogService(
			galgameRepository,
			galgameRepo.NewDeveloperRepository(db),
			galgameRepo.NewTagRepository(db),
		),
		rbac: rbacSvc,
		db:   db,
	}
}

func (e *communityTestEnv) createPublishedGalgame(t *testing.T, userID uint, title string) uint {
	t.Helper()
	galgame, err := e.catalog.CreateGalgame(context.Background(), userID, &galgameDTO.CreateGalgameRequest{
		Title:  title,
		Slug:   title,
		Status: 1,
	})
	if err != nil {
		t.Fatalf("create galgame %s: %v", title, err)
	}
	return galgame.ID
}

func (e *communityTestEnv) createPost(
	t *testing.T,
	authorID uint,
	title string,
	galgameID *uint,
) *model.Post {
	t.Helper()
	post, err := e.posts.Create(context.Background(), authorID, &dto.CreatePostRequest{
		Title:     title,
		Content:   title + " content",
		GalgameID: galgameID,
	})
	if err != nil {
		t.Fatalf("create post %s: %v", title, err)
	}
	return post
}

func (e *communityTestEnv) createComment(
	t *testing.T,
	authorID, postID uint,
	content string,
	parentID, replyToCommentID *uint,
) *model.Comment {
	t.Helper()
	comment, err := e.comments.Create(context.Background(), authorID, postID, &dto.CreateCommentRequest{
		Content:          content,
		ParentID:         parentID,
		ReplyToCommentID: replyToCommentID,
	})
	if err != nil {
		t.Fatalf("create comment %s: %v", content, err)
	}
	return comment
}

func (e *communityTestEnv) galgamePostCount(t *testing.T, galgameID uint) int64 {
	t.Helper()
	var count int64
	if err := e.db.Raw(
		"SELECT post_count FROM galgames WHERE id = ?", galgameID,
	).Scan(&count).Error; err != nil {
		t.Fatalf("read galgame post count: %v", err)
	}
	return count
}

func uintPtr(value uint) *uint { return &value }

func TestPostCrudAndGalgamePostCount(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	author := testutil.CreateUser(t, env.db, "post-author")
	galgameID := env.createPublishedGalgame(t, author, "post-crud-game")

	post := env.createPost(t, author, "first post", uintPtr(galgameID))
	if post.GalgameID == nil || *post.GalgameID != galgameID {
		t.Fatalf("expected galgame binding, got %+v", post)
	}
	if count := env.galgamePostCount(t, galgameID); count != 1 {
		t.Fatalf("expected post_count 1, got %d", count)
	}
	env.createPost(t, author, "plain post", nil)

	if _, err := env.posts.Create(ctx, author, &dto.CreatePostRequest{
		Title: "bad", Content: "x", GalgameID: uintPtr(999999),
	}); !errors.Is(err, ErrGalgameNotFound) {
		t.Fatalf("expected ErrGalgameNotFound, got %v", err)
	}
	if _, err := env.posts.Create(ctx, author, &dto.CreatePostRequest{Title: " ", Content: "x"}); !errors.Is(err, ErrInvalidPostInput) {
		t.Fatalf("expected ErrInvalidPostInput, got %v", err)
	}

	updated, err := env.posts.Update(ctx, author, post.ID, &dto.UpdatePostRequest{
		Title: "first post v2", Content: "updated content",
	})
	if err != nil || updated.Title != "first post v2" {
		t.Fatalf("unexpected update result: %+v err=%v", updated, err)
	}
	if count := env.galgamePostCount(t, galgameID); count != 1 {
		t.Fatalf("update must not change post_count, got %d", count)
	}

	if err := env.posts.Delete(ctx, author, post.ID); err != nil {
		t.Fatalf("delete post: %v", err)
	}
	if count := env.galgamePostCount(t, galgameID); count != 0 {
		t.Fatalf("expected post_count 0 after delete, got %d", count)
	}
	if err := env.posts.Delete(ctx, author, post.ID); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
	if count := env.galgamePostCount(t, galgameID); count != 0 {
		t.Fatalf("expected post_count to stay 0, got %d", count)
	}
}

func TestPostEditorMode(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	author := testutil.CreateUser(t, env.db, "post-editor-mode-author")

	// Historical clients omit editor_mode: it must default to plain.
	legacy, err := env.posts.Create(ctx, author, &dto.CreatePostRequest{
		Title:   "legacy post",
		Content: "# not a heading",
	})
	if err != nil || legacy.EditorMode != model.EditorModePlain {
		t.Fatalf("expected default plain editor mode, got %+v err=%v", legacy, err)
	}

	created, err := env.posts.Create(ctx, author, &dto.CreatePostRequest{
		Title:      "markdown post",
		Content:    "# heading",
		EditorMode: model.EditorModeMarkdown,
	})
	if err != nil || created.EditorMode != model.EditorModeMarkdown {
		t.Fatalf("expected markdown editor mode, got %+v err=%v", created, err)
	}

	if _, err := env.posts.Create(ctx, author, &dto.CreatePostRequest{
		Title:      "invalid mode",
		Content:    "x",
		EditorMode: model.EditorMode("abc"),
	}); !errors.Is(err, ErrInvalidPostInput) {
		t.Fatalf("expected ErrInvalidPostInput, got %v", err)
	}

	updated, err := env.posts.Update(ctx, author, legacy.ID, &dto.UpdatePostRequest{
		Title:      "legacy post v2",
		Content:    "still plain",
		EditorMode: model.EditorModeMarkdown,
	})
	if err != nil || updated.EditorMode != model.EditorModeMarkdown {
		t.Fatalf("expected updated markdown editor mode, got %+v err=%v", updated, err)
	}

	updated, err = env.posts.Update(ctx, author, created.ID, &dto.UpdatePostRequest{
		Title:   "markdown post v2",
		Content: "## smaller heading",
	})
	if err != nil || updated.EditorMode != model.EditorModeMarkdown {
		t.Fatalf("expected update without editor_mode to keep markdown, got %+v err=%v", updated, err)
	}

	if _, err := env.posts.Update(ctx, author, created.ID, &dto.UpdatePostRequest{
		Title:      "invalid update mode",
		Content:    "x",
		EditorMode: model.EditorMode("abc"),
	}); !errors.Is(err, ErrInvalidPostInput) {
		t.Fatalf("expected ErrInvalidPostInput on update, got %v", err)
	}
}

func TestPostListFiltersAndPagination(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	author := testutil.CreateUser(t, env.db, "post-list-author")
	galgameA := env.createPublishedGalgame(t, author, "list-game-a")
	galgameB := env.createPublishedGalgame(t, author, "list-game-b")

	env.createPost(t, author, "a1", uintPtr(galgameA))
	env.createPost(t, author, "a2", uintPtr(galgameA))
	env.createPost(t, author, "b1", uintPtr(galgameB))
	env.createPost(t, author, "plain", nil)

	posts, total, _, _, err := env.posts.List(ctx, &dto.PostQuery{})
	if err != nil || total != 4 || len(posts) != 4 {
		t.Fatalf("expected 4 posts, got total=%d items=%d err=%v", total, len(posts), err)
	}
	if posts[0].Title != "plain" {
		t.Fatalf("expected latest-first ordering, got %+v", posts[0].Title)
	}

	posts, total, _, _, err = env.posts.List(ctx, &dto.PostQuery{GalgameID: uintPtr(galgameA)})
	if err != nil || total != 2 || len(posts) != 2 {
		t.Fatalf("expected 2 posts for galgame A, got total=%d items=%d err=%v", total, len(posts), err)
	}

	posts, total, page, limit, err := env.posts.List(ctx, &dto.PostQuery{Page: 2, Limit: 2})
	if err != nil || total != 4 || page != 2 || limit != 2 || len(posts) != 2 || posts[0].Title != "a2" {
		t.Fatalf("unexpected pagination: total=%d page=%d limit=%d first=%+v err=%v",
			total, page, limit, posts, err)
	}
}

func TestCommentTwoLevelStructure(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	userA := testutil.CreateUser(t, env.db, "comment-a")
	userB := testutil.CreateUser(t, env.db, "comment-b")
	userC := testutil.CreateUser(t, env.db, "comment-c")
	post := env.createPost(t, userA, "comment-target", nil)
	otherPost := env.createPost(t, userA, "comment-other", nil)

	top := env.createComment(t, userA, post.ID, "top level", nil, nil)
	reply := env.createComment(t, userB, post.ID, "reply to top", uintPtr(top.ID), nil)
	replyToReply := env.createComment(t, userC, post.ID, "reply to reply",
		uintPtr(top.ID), uintPtr(reply.ID))

	if replyToReply.ParentID == nil || *replyToReply.ParentID != top.ID {
		t.Fatalf("expected parent to stay top-level, got %+v", replyToReply.ParentID)
	}
	if replyToReply.ReplyToUserID == nil || *replyToReply.ReplyToUserID != userB {
		t.Fatalf("expected reply_to_user to be reply author, got %+v", replyToReply.ReplyToUserID)
	}

	derived := env.createComment(t, userB, post.ID, "derived parent", nil, uintPtr(reply.ID))
	if derived.ParentID == nil || *derived.ParentID != top.ID {
		t.Fatalf("expected derived parent, got %+v", derived.ParentID)
	}

	comments, replyCounts, total, _, _, err := env.comments.ListByPost(ctx, post.ID, 1, 20)
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if total != 1 || len(comments) != 1 {
		t.Fatalf("expected 1 top-level thread, got total=%d comments=%d", total, len(comments))
	}
	if replyCounts[top.ID] != 3 {
		t.Fatalf("expected reply count 3, got %d", replyCounts[top.ID])
	}
	replies, replyTotal, replyPage, replyLimit, err := env.comments.ListReplies(ctx, top.ID, 1, 2)
	if err != nil || replyTotal != 3 || replyPage != 1 || replyLimit != 2 || len(replies) != 2 {
		t.Fatalf("unexpected reply pagination: total=%d page=%d limit=%d replies=%d err=%v",
			replyTotal, replyPage, replyLimit, len(replies), err)
	}

	post2, err := env.posts.Get(ctx, post.ID)
	if err != nil || post2.CommentCount != 4 {
		t.Fatalf("expected comment_count 4, got %d err=%v", post2.CommentCount, err)
	}

	if _, err := env.comments.Create(ctx, userB, post.ID, &dto.CreateCommentRequest{
		Content: "nested parent", ParentID: uintPtr(reply.ID),
	}); !errors.Is(err, ErrInvalidCommentParent) {
		t.Fatalf("expected ErrInvalidCommentParent for reply parent, got %v", err)
	}
	if _, err := env.comments.Create(ctx, userB, post.ID, &dto.CreateCommentRequest{
		Content: "cross post", ParentID: uintPtr(env.createComment(t, userA, otherPost.ID, "other top", nil, nil).ID),
	}); !errors.Is(err, ErrInvalidCommentParent) {
		t.Fatalf("expected ErrInvalidCommentParent for cross-post parent, got %v", err)
	}
	if _, err := env.comments.Create(ctx, userB, post.ID, &dto.CreateCommentRequest{
		Content: "cross thread", ParentID: uintPtr(top.ID),
		ReplyToCommentID: uintPtr(env.createComment(t, userA, otherPost.ID, "other reply", nil, nil).ID),
	}); !errors.Is(err, ErrInvalidCommentReplyTo) {
		t.Fatalf("expected ErrInvalidCommentReplyTo for cross-post target, got %v", err)
	}
	if _, err := env.comments.Create(ctx, userB, post.ID, &dto.CreateCommentRequest{
		Content: " ", ParentID: uintPtr(top.ID),
	}); !errors.Is(err, ErrInvalidCommentInput) {
		t.Fatalf("expected ErrInvalidCommentInput, got %v", err)
	}
	if _, err := env.comments.Create(ctx, userB, 999999, &dto.CreateCommentRequest{
		Content: "ghost post",
	}); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestCommentDeleteAdjustsPostCount(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	author := testutil.CreateUser(t, env.db, "comment-delete-author")
	post := env.createPost(t, author, "comment-delete-target", nil)

	top := env.createComment(t, author, post.ID, "top", nil, nil)
	env.createComment(t, author, post.ID, "r1", uintPtr(top.ID), nil)
	env.createComment(t, author, post.ID, "r2", uintPtr(top.ID), nil)
	standalone := env.createComment(t, author, post.ID, "solo", nil, nil)

	if err := env.comments.Delete(ctx, author, top.ID); err != nil {
		t.Fatalf("delete top comment: %v", err)
	}
	after, _ := env.posts.Get(ctx, post.ID)
	if after.CommentCount != 1 {
		t.Fatalf("expected comment_count 1 after subtree delete, got %d", after.CommentCount)
	}

	if err := env.comments.Delete(ctx, author, standalone.ID); err != nil {
		t.Fatalf("delete standalone comment: %v", err)
	}
	after, _ = env.posts.Get(ctx, post.ID)
	if after.CommentCount != 0 {
		t.Fatalf("expected comment_count 0, got %d", after.CommentCount)
	}
	if err := env.comments.Delete(ctx, author, standalone.ID); !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
	after, _ = env.posts.Get(ctx, post.ID)
	if after.CommentCount != 0 {
		t.Fatalf("expected comment_count to stay 0, got %d", after.CommentCount)
	}
}

func TestCommunityModerationPermissions(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	author := testutil.CreateUser(t, env.db, "community-author")
	stranger := testutil.CreateUser(t, env.db, "community-stranger")
	moderator := testutil.CreateUser(t, env.db, "community-moderator")
	if err := env.rbac.AssignRoleByCode(ctx, moderator, rbacService.RoleCodeAdmin); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}

	post := env.createPost(t, author, "moderated post", nil)
	comment := env.createComment(t, author, post.ID, "moderated comment", nil, nil)

	if _, err := env.posts.Update(ctx, stranger, post.ID, &dto.UpdatePostRequest{
		Title: "hijack", Content: "hijack",
	}); !errors.Is(err, ErrForbiddenPost) {
		t.Fatalf("expected ErrForbiddenPost, got %v", err)
	}
	if err := env.posts.Delete(ctx, stranger, post.ID); !errors.Is(err, ErrForbiddenPost) {
		t.Fatalf("expected ErrForbiddenPost, got %v", err)
	}
	if _, err := env.comments.Update(ctx, stranger, comment.ID, &dto.UpdateCommentRequest{Content: "hijack"}); !errors.Is(err, ErrForbiddenComment) {
		t.Fatalf("expected ErrForbiddenComment, got %v", err)
	}
	if err := env.comments.Delete(ctx, stranger, comment.ID); !errors.Is(err, ErrForbiddenComment) {
		t.Fatalf("expected ErrForbiddenComment, got %v", err)
	}

	if _, err := env.posts.Update(ctx, moderator, post.ID, &dto.UpdatePostRequest{
		Title: "moderated", Content: "moderated",
	}); err != nil {
		t.Fatalf("moderator update post: %v", err)
	}
	edited, err := env.comments.Update(ctx, moderator, comment.ID, &dto.UpdateCommentRequest{Content: "moderated"})
	if err != nil || edited.Content != "moderated" {
		t.Fatalf("moderator update comment: %+v err=%v", edited, err)
	}
}

func TestLikesAndFavoritesLifecycle(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, env.db, "interaction-user")
	post := env.createPost(t, user, "interaction post", nil)
	comment := env.createComment(t, user, post.ID, "interaction comment", nil, nil)

	post2, err := env.interactions.LikePost(ctx, user, post.ID)
	if err != nil || post2.LikeCount != 1 {
		t.Fatalf("expected like_count 1, got %+v err=%v", post2, err)
	}
	if _, err := env.interactions.LikePost(ctx, user, post.ID); !errors.Is(err, ErrAlreadyLiked) {
		t.Fatalf("expected ErrAlreadyLiked, got %v", err)
	}
	post2, err = env.interactions.UnlikePost(ctx, user, post.ID)
	if err != nil || post2.LikeCount != 0 {
		t.Fatalf("expected like_count 0, got %+v err=%v", post2, err)
	}
	if _, err := env.interactions.UnlikePost(ctx, user, post.ID); !errors.Is(err, ErrLikeNotFound) {
		t.Fatalf("expected ErrLikeNotFound, got %v", err)
	}

	post2, err = env.interactions.FavoritePost(ctx, user, post.ID)
	if err != nil || post2.FavoriteCount != 1 {
		t.Fatalf("expected favorite_count 1, got %+v err=%v", post2, err)
	}
	if _, err := env.interactions.FavoritePost(ctx, user, post.ID); !errors.Is(err, ErrAlreadyFavorited) {
		t.Fatalf("expected ErrAlreadyFavorited, got %v", err)
	}
	post2, err = env.interactions.UnfavoritePost(ctx, user, post.ID)
	if err != nil || post2.FavoriteCount != 0 {
		t.Fatalf("expected favorite_count 0, got %+v err=%v", post2, err)
	}
	if _, err := env.interactions.UnfavoritePost(ctx, user, post.ID); !errors.Is(err, ErrFavoriteNotFound) {
		t.Fatalf("expected ErrFavoriteNotFound, got %v", err)
	}

	comment2, err := env.interactions.LikeComment(ctx, user, comment.ID)
	if err != nil || comment2.LikeCount != 1 {
		t.Fatalf("expected comment like_count 1, got %+v err=%v", comment2, err)
	}
	if _, err := env.interactions.LikeComment(ctx, user, comment.ID); !errors.Is(err, ErrAlreadyLiked) {
		t.Fatalf("expected ErrAlreadyLiked, got %v", err)
	}
	comment2, err = env.interactions.UnlikeComment(ctx, user, comment.ID)
	if err != nil || comment2.LikeCount != 0 {
		t.Fatalf("expected comment like_count 0, got %+v err=%v", comment2, err)
	}
}

func TestPostLikeConcurrentCountNotLost(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	owner := testutil.CreateUser(t, env.db, "like-owner")
	post := env.createPost(t, owner, "concurrent like post", nil)

	const users = 12
	userIDs := make([]uint, 0, users)
	for i := 0; i < users; i++ {
		userIDs = append(userIDs, testutil.CreateUser(t, env.db, fmt.Sprintf("like-user-%d", i)))
	}

	errs := make(chan error, users)
	var wg sync.WaitGroup
	for _, userID := range userIDs {
		wg.Add(1)
		go func(userID uint) {
			defer wg.Done()
			if _, err := env.interactions.LikePost(ctx, userID, post.ID); err != nil {
				errs <- err
			}
		}(userID)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent like: %v", err)
	}

	liked, err := env.posts.Get(ctx, post.ID)
	if err != nil || liked.LikeCount != users {
		t.Fatalf("expected like_count %d, got %d err=%v", users, liked.LikeCount, err)
	}
}

func TestCommunityWritesRollbackOnError(t *testing.T) {
	env := newCommunityTestEnv(t)
	ctx := context.Background()
	author := testutil.CreateUser(t, env.db, "community-rollback")
	galgameID := env.createPublishedGalgame(t, author, "rollback-game")

	injected := errors.New("injected failure")
	post := env.createPost(t, author, "rollback post", uintPtr(galgameID))

	err := env.db.Transaction(func(tx *gorm.DB) error {
		comments := communityRepo.NewCommentRepository(tx)
		comment := &model.Comment{
			PostID:   post.ID,
			AuthorID: uintPtr(author),
			Content:  "rollback comment",
		}
		if err := comments.Create(ctx, comment); err != nil {
			return err
		}
		if err := comments.IncrementPostCommentCount(ctx, comment.PostID); err != nil {
			return err
		}
		return injected
	})
	if !errors.Is(err, injected) {
		t.Fatalf("expected injected failure, got %v", err)
	}

	var commentCount int64
	if err := env.db.Raw(
		"SELECT comment_count FROM posts WHERE id = ?", post.ID,
	).Scan(&commentCount).Error; err != nil || commentCount != 0 {
		t.Fatalf("expected comment_count rolled back to 0, got %d err=%v", commentCount, err)
	}
	var commentRows int64
	if err := env.db.Raw(
		"SELECT COUNT(*) FROM comments WHERE post_id = ?", post.ID,
	).Scan(&commentRows).Error; err != nil || commentRows != 0 {
		t.Fatalf("expected comment row rolled back, got %d err=%v", commentRows, err)
	}
}
