package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"backend/internal/middleware"
	"backend/internal/notification/dto"
	"backend/internal/notification/model"
	"backend/internal/notification/repository"
	"backend/internal/notification/service"
	"backend/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const notificationTestSecret = "notification-test-secret"

type notificationTestClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func notificationToken(t *testing.T, userID uint) string {
	t.Helper()
	claims := notificationTestClaims{
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "koyomi-gal", Subject: strconv.FormatUint(uint64(userID), 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(notificationTestSecret))
	if err != nil {
		t.Fatalf("sign notification token: %v", err)
	}
	return token
}

func TestNotificationEndpointsOwnIsolationAndReadFlow(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	svc := service.NewNotificationService(repository.NewNotificationRepository(db, "https://img.example.com"))
	handler := NewNotificationHandler(svc)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	protected := router.Group("/api/v1", middleware.Auth(notificationTestSecret))
	protected.GET("/notifications", handler.ListNotifications)
	protected.GET("/notifications/unread-count", handler.UnreadCount)
	protected.PATCH("/notifications/:id/read", handler.MarkRead)
	protected.PATCH("/notifications/read-all", handler.MarkAllRead)

	recipient := testutil.CreateUser(t, db, "notification-http-recipient")
	other := testutil.CreateUser(t, db, "notification-http-other")
	actor := testutil.CreateUser(t, db, "notification-http-actor")
	created, err := svc.CreateMany(context.Background(), []service.CreateInput{
		{
			RecipientID: recipient, ActorID: &actor, Category: model.CategoryInteraction,
			Type: model.TypePostLiked, EntityType: "post", EntityID: 1,
			Title: "first", TargetURL: "/posts/1",
		},
		{
			RecipientID: recipient, Category: model.CategoryReview,
			Type: model.TypeResourceApproved, EntityType: "resource", EntityID: 2,
			Title: "second", TargetURL: "/resources/2",
		},
		{
			RecipientID: other, ActorID: &actor, Category: model.CategoryInteraction,
			Type: model.TypePostLiked, EntityType: "post", EntityID: 3,
			Title: "foreign", TargetURL: "/posts/3",
		},
	})
	if err != nil {
		t.Fatalf("seed notifications: %v", err)
	}
	token := notificationToken(t, recipient)
	otherToken := notificationToken(t, other)

	res := notificationRequest(router, http.MethodGet, "/api/v1/notifications", "")
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("list without token: expected 401, got %d", res.Code)
	}
	res = notificationRequest(router, http.MethodGet, "/api/v1/notifications?unread=true&limit=1&category=interaction", token)
	if res.Code != http.StatusOK {
		t.Fatalf("list notifications: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var listed struct {
		Code int                      `json:"code"`
		Data dto.NotificationListData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode notification list: %v", err)
	}
	if listed.Data.Total != 1 || len(listed.Data.Items) != 1 || listed.Data.Items[0].Title == "foreign" {
		t.Fatalf("unexpected notification list: %+v", listed.Data)
	}
	item := listed.Data.Items[0]
	if item.EntityType == nil || *item.EntityType != "post" || item.EntityID == nil || *item.EntityID != 1 || item.TargetURL == nil || *item.TargetURL != "/posts/1" || item.IsRead {
		t.Fatalf("unexpected notification navigation/read data: %+v", item)
	}

	foreignPath := "/api/v1/notifications/" + strconv.FormatUint(uint64(created[2].ID), 10) + "/read"
	res = notificationRequest(router, http.MethodPatch, foreignPath, token)
	if res.Code != http.StatusNotFound {
		t.Fatalf("mark foreign notification: expected 404, got %d", res.Code)
	}
	ownPath := "/api/v1/notifications/" + strconv.FormatUint(uint64(created[0].ID), 10) + "/read"
	res = notificationRequest(router, http.MethodPatch, ownPath, token)
	if res.Code != http.StatusOK {
		t.Fatalf("mark own notification: expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	res = notificationRequest(router, http.MethodGet, "/api/v1/notifications/unread-count", token)
	if res.Code != http.StatusOK {
		t.Fatalf("unread count: expected 200, got %d", res.Code)
	}
	var unread struct {
		Data dto.UnreadCountData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &unread); err != nil || unread.Data.Count != 1 {
		t.Fatalf("expected one unread notification, response=%+v err=%v", unread, err)
	}
	res = notificationRequest(router, http.MethodPatch, "/api/v1/notifications/read-all", token)
	if res.Code != http.StatusOK {
		t.Fatalf("mark all read: expected 200, got %d", res.Code)
	}
	res = notificationRequest(router, http.MethodGet, "/api/v1/notifications/unread-count", token)
	if err := json.Unmarshal(res.Body.Bytes(), &unread); err != nil || unread.Data.Count != 0 {
		t.Fatalf("expected no unread notifications, response=%+v err=%v", unread, err)
	}
	res = notificationRequest(router, http.MethodGet, "/api/v1/notifications/unread-count", otherToken)
	if err := json.Unmarshal(res.Body.Bytes(), &unread); err != nil || unread.Data.Count != 1 {
		t.Fatalf("foreign unread notification changed, response=%+v err=%v", unread, err)
	}
}

func notificationRequest(router *gin.Engine, method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}
