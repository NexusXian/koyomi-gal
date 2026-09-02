package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"backend/internal/community/dto"
	communityRepo "backend/internal/community/repository"
	communityService "backend/internal/community/service"
	galgameRepo "backend/internal/galgame/repository"
	"backend/internal/middleware"
	rbacRepo "backend/internal/rbac/repository"
	rbacService "backend/internal/rbac/service"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const testAuthSecret = "community-test-secret"

type testTokenClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func accessTokenFor(t *testing.T, userID uint) string {
	t.Helper()
	claims := testTokenClaims{
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "koyomi-gal",
			Subject:   strconv.FormatUint(uint64(userID), 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testAuthSecret))
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}
	return token
}

type communityHandlerEnv struct {
	router *gin.Engine
	db     *gorm.DB
	rbac   *rbacService.RBACService
}

func newCommunityHandlerEnv(t *testing.T) *communityHandlerEnv {
	t.Helper()
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	rbacSvc := rbacService.NewRBACService(rbacRepo.NewRBACRepository(db))
	if err := rbacSvc.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seed rbac defaults: %v", err)
	}
	postRepo := communityRepo.NewPostRepository(db, "https://img.example.com")
	commentRepo := communityRepo.NewCommentRepository(db, "https://img.example.com")
	postHandler := NewPostHandler(communityService.NewPostService(
		postRepo, galgameRepo.NewGalgameRepository(db), rbacSvc,
	))
	commentHandler := NewCommentHandler(
		communityService.NewCommentService(commentRepo, postRepo, rbacSvc),
	)
	interactionHandler := NewInteractionHandler(
		communityService.NewInteractionService(postRepo, commentRepo),
	)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/posts", postHandler.ListPosts)
	router.GET("/api/v1/posts/:id", postHandler.GetPost)
	router.GET("/api/v2/posts/:id/comments", commentHandler.ListPostComments)
	router.GET("/api/v2/comments/:id/replies", commentHandler.ListCommentReplies)

	protected := router.Group("/api/v1", middleware.Auth(testAuthSecret))
	{
		protected.POST("/posts", postHandler.CreatePost)
		protected.PUT("/posts/:id", postHandler.UpdatePost)
		protected.DELETE("/posts/:id", postHandler.DeletePost)
		protected.POST("/posts/:id/like", interactionHandler.LikePost)
		protected.DELETE("/posts/:id/like", interactionHandler.UnlikePost)
		protected.POST("/posts/:id/favorite", interactionHandler.FavoritePost)
		protected.DELETE("/posts/:id/favorite", interactionHandler.UnfavoritePost)
		protected.POST("/posts/:id/comments", commentHandler.CreateComment)
		protected.PUT("/comments/:id", commentHandler.UpdateComment)
		protected.DELETE("/comments/:id", commentHandler.DeleteComment)
		protected.POST("/comments/:id/like", interactionHandler.LikeComment)
		protected.DELETE("/comments/:id/like", interactionHandler.UnlikeComment)
	}

	return &communityHandlerEnv{router: router, db: db, rbac: rbacSvc}
}

func doCommunityRequest(router *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func TestCommunityEndpointsRequireAuth(t *testing.T) {
	env := newCommunityHandlerEnv(t)

	protected := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodPost, "/api/v1/posts", map[string]any{"title": "t", "content": "c"}},
		{http.MethodPut, "/api/v1/posts/1", map[string]any{"title": "t", "content": "c"}},
		{http.MethodDelete, "/api/v1/posts/1", nil},
		{http.MethodPost, "/api/v1/posts/1/like", nil},
		{http.MethodPost, "/api/v1/posts/1/favorite", nil},
		{http.MethodPost, "/api/v1/posts/1/comments", map[string]any{"content": "c"}},
		{http.MethodPut, "/api/v1/comments/1", map[string]any{"content": "c"}},
		{http.MethodDelete, "/api/v1/comments/1", nil},
		{http.MethodPost, "/api/v1/comments/1/like", nil},
	}
	for _, endpoint := range protected {
		res := doCommunityRequest(env.router, endpoint.method, endpoint.path, "", endpoint.body)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without token: expected 401, got %d", endpoint.method, endpoint.path, res.Code)
		}
	}

	res := doCommunityRequest(env.router, http.MethodGet, "/api/v1/posts", "", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("public post list: expected 200, got %d", res.Code)
	}
	res = doCommunityRequest(env.router, http.MethodGet, "/api/v2/posts/999999/comments", "", nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("comments of unknown post: expected 404, got %d", res.Code)
	}
}

func TestCommunityEndpointsFlow(t *testing.T) {
	env := newCommunityHandlerEnv(t)
	author := testutil.CreateUser(t, env.db, "community-http-author")
	stranger := testutil.CreateUser(t, env.db, "community-http-stranger")
	authorToken := accessTokenFor(t, author)
	strangerToken := accessTokenFor(t, stranger)

	res := doCommunityRequest(env.router, http.MethodPost, "/api/v1/posts", authorToken, map[string]any{
		"title":   "http post",
		"content": "http content",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("create post: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var created struct {
		Code int          `json:"code"`
		Data dto.PostData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create post response: %v", err)
	}
	if created.Data.AuthorID == nil || *created.Data.AuthorID != author {
		t.Fatalf("expected author from login state, got %+v", created.Data.AuthorID)
	}
	postPath := "/api/v1/posts/" + strconv.FormatUint(uint64(created.Data.ID), 10)

	res = doCommunityRequest(env.router, http.MethodPost, postPath+"/like", authorToken, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("like post: expected 200, got %d", res.Code)
	}
	res = doCommunityRequest(env.router, http.MethodPost, postPath+"/like", authorToken, nil)
	if res.Code != http.StatusConflict {
		t.Fatalf("duplicate like: expected 409, got %d", res.Code)
	}
	res = doCommunityRequest(env.router, http.MethodPost, postPath+"/favorite", authorToken, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("favorite post: expected 200, got %d", res.Code)
	}

	res = doCommunityRequest(env.router, http.MethodPost, postPath+"/comments", authorToken, map[string]any{
		"content": "top comment",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("create comment: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var topComment struct {
		Code int             `json:"code"`
		Data dto.CommentData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &topComment); err != nil {
		t.Fatalf("decode comment response: %v", err)
	}

	res = doCommunityRequest(env.router, http.MethodPost, postPath+"/comments", strangerToken, map[string]any{
		"content":             "reply to reply",
		"reply_to_comment_id": topComment.Data.ID,
	})
	if res.Code != http.StatusOK {
		t.Fatalf("create reply: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var reply struct {
		Code int             `json:"code"`
		Data dto.CommentData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &reply); err != nil {
		t.Fatalf("decode reply response: %v", err)
	}
	if reply.Data.ParentID == nil || *reply.Data.ParentID != topComment.Data.ID {
		t.Fatalf("expected parent derived from reply target, got %+v", reply.Data.ParentID)
	}
	if reply.Data.ReplyToUserID == nil || *reply.Data.ReplyToUserID != author {
		t.Fatalf("expected reply_to_user from target author, got %+v", reply.Data.ReplyToUserID)
	}

	res = doCommunityRequest(env.router, http.MethodPost,
		"/api/v1/comments/"+strconv.FormatUint(uint64(reply.Data.ID), 10)+"/like", authorToken, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("like comment: expected 200, got %d", res.Code)
	}

	res = doCommunityRequest(env.router, http.MethodGet,
		"/api/v2/posts/"+strconv.FormatUint(uint64(created.Data.ID), 10)+"/comments", "", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list comments: expected 200, got %d", res.Code)
	}
	var listed struct {
		Code int                 `json:"code"`
		Data dto.CommentListData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode comment list response: %v", err)
	}
	if len(listed.Data.Items) != 1 || listed.Data.Items[0].ReplyCount != 1 {
		t.Fatalf("expected one thread with one reply count, got %+v", listed.Data)
	}
	res = doCommunityRequest(env.router, http.MethodGet,
		"/api/v2/comments/"+strconv.FormatUint(uint64(topComment.Data.ID), 10)+"/replies", "", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list replies: expected 200, got %d", res.Code)
	}
	var replyList struct {
		Code int                 `json:"code"`
		Data dto.CommentListData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &replyList); err != nil {
		t.Fatalf("decode reply list response: %v", err)
	}
	if replyList.Data.Total != 1 || len(replyList.Data.Items) != 1 || replyList.Data.Items[0].LikeCount != 1 {
		t.Fatalf("expected one liked reply, got %+v", replyList.Data)
	}

	res = doCommunityRequest(env.router, http.MethodPut, postPath, strangerToken, map[string]any{
		"title": "hijack", "content": "hijack",
	})
	if res.Code != http.StatusForbidden {
		t.Fatalf("stranger update post: expected 403, got %d", res.Code)
	}

	res = doCommunityRequest(env.router, http.MethodDelete, postPath, authorToken, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("author delete post: expected 200, got %d", res.Code)
	}
	res = doCommunityRequest(env.router, http.MethodGet, postPath, "", nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("post after delete: expected 404, got %d", res.Code)
	}

	var commentRows int64
	if err := env.db.Raw("SELECT COUNT(*) FROM comments").Scan(&commentRows).Error; err != nil || commentRows != 0 {
		t.Fatalf("expected comments cascaded, got %d err=%v", commentRows, err)
	}
}
