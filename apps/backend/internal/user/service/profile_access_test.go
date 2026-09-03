package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"backend/internal/user/dto"
	"backend/internal/user/model"
)

func TestProfileAccess(t *testing.T) {
	service := NewProfileAccessService()
	ownerID := uint(1001)
	viewerID := uint(1002)
	base := model.PublicProfileRecord{
		ID: ownerID, ProfileVisibility: model.ProfileVisibilityPublic,
		ShowPosts: true, ShowComments: true, ShowRatings: true, ShowActivity: true,
	}

	tests := []struct {
		name    string
		profile model.PublicProfileRecord
		viewer  *uint
		want    dto.ProfileAccess
	}{
		{name: "public anonymous", profile: base, want: dto.ProfileAccess{CanViewProfile: true, CanViewPosts: true, CanViewComments: true, CanViewRatings: true, CanViewActivity: true}},
		{name: "registered anonymous", profile: withVisibility(base, model.ProfileVisibilityRegistered)},
		{name: "registered authenticated", profile: withVisibility(base, model.ProfileVisibilityRegistered), viewer: &viewerID, want: dto.ProfileAccess{CanViewProfile: true, CanViewPosts: true, CanViewComments: true, CanViewRatings: true, CanViewActivity: true}},
		{name: "private other", profile: withVisibility(base, model.ProfileVisibilityPrivate), viewer: &viewerID},
		{name: "owner bypasses private and hidden collections", profile: withVisibility(base, model.ProfileVisibilityPrivate), viewer: &ownerID, want: dto.ProfileAccess{CanViewProfile: true, CanViewPosts: true, CanViewComments: true, CanViewRatings: true, CanViewFavorites: true, CanViewActivity: true, CanViewLocation: true, CanViewBirthday: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			access := service.Resolve(&test.profile, test.viewer)
			if access != test.want {
				t.Fatalf("unexpected access: %+v", access)
			}
		})
	}
}

func TestDefaultPrivacy(t *testing.T) {
	settings := model.DefaultUserPrivacySettings(1001)
	if settings.ProfileVisibility != model.ProfileVisibilityPublic || !settings.ShowPosts || !settings.ShowComments || !settings.ShowRatings || !settings.ShowActivity {
		t.Fatalf("unexpected visible defaults: %+v", settings)
	}
	if settings.ShowLocation || settings.ShowBirthday || settings.ShowFavorites {
		t.Fatalf("location, birthday, and favorites must default hidden: %+v", settings)
	}
}

func TestProfileAccessHiddenCollections(t *testing.T) {
	profile := model.PublicProfileRecord{
		ID: 1001, ProfileVisibility: model.ProfileVisibilityPublic,
		ShowActivity: true, ShowPosts: true, ShowComments: true,
		ShowRatings: true, ShowFavorites: true,
	}
	tests := []struct {
		name   string
		hide   func(*model.PublicProfileRecord)
		access func(dto.ProfileAccess) bool
	}{
		{name: "posts", hide: func(p *model.PublicProfileRecord) { p.ShowPosts = false }, access: func(a dto.ProfileAccess) bool { return a.CanViewPosts }},
		{name: "comments", hide: func(p *model.PublicProfileRecord) { p.ShowComments = false }, access: func(a dto.ProfileAccess) bool { return a.CanViewComments }},
		{name: "ratings", hide: func(p *model.PublicProfileRecord) { p.ShowRatings = false }, access: func(a dto.ProfileAccess) bool { return a.CanViewRatings }},
		{name: "favorites", hide: func(p *model.PublicProfileRecord) { p.ShowFavorites = false }, access: func(a dto.ProfileAccess) bool { return a.CanViewFavorites }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := profile
			test.hide(&current)
			if test.access(NewProfileAccessService().Resolve(&current, nil)) {
				t.Fatalf("%s remained visible", test.name)
			}
		})
	}
}

func TestVisibleActivityTypesRespectCollectionPrivacy(t *testing.T) {
	types := visibleActivityTypes(NewProfileAccessService().Resolve(&model.PublicProfileRecord{
		ID: 1001, ProfileVisibility: model.ProfileVisibilityPublic,
		ShowActivity: true, ShowPosts: true, ShowComments: false,
		ShowRatings: false, ShowFavorites: false,
	}, nil))
	joined := strings.Join(types, ",")
	if !strings.Contains(joined, model.ActivityPostCreated) {
		t.Fatalf("post activity should be visible: %v", types)
	}
	for _, hidden := range []string{model.ActivityCommentCreated, model.ActivityRatingCreated, model.ActivityFavoriteCreated} {
		if strings.Contains(joined, hidden) {
			t.Fatalf("hidden collection activity %q leaked: %v", hidden, types)
		}
	}
}

func TestPublicDTOsDoNotLeakPrivateFields(t *testing.T) {
	registered := time.Now()
	birthday := registered.AddDate(-20, 0, 0)
	viewerID := uint(1002)
	service := &UserProfileService{access: NewProfileAccessService()}
	profile, err := service.buildPublicProfile(context.Background(), &model.PublicProfileRecord{
		ID: 1001, Username: "koyomi", DisplayName: "Koyomi", AvatarURL: "avatar",
		BannerURL: "banner", Bio: "private bio", Location: "private location", Birthday: &birthday,
		RegisteredAt: registered, ProfileVisibility: model.ProfileVisibilityPrivate,
		ShowLocation: true, ShowBirthday: true, ShowPosts: true, ShowFavorites: true,
	}, &viewerID)
	if err != nil {
		t.Fatalf("build restricted profile: %v", err)
	}
	encoded, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal profile: %v", err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"email", "roles", "is_banned", "password", "private bio", "private location", "registered_at", "post_count", "banner_url"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("public DTO leaked %q: %s", forbidden, text)
		}
	}
	for _, required := range []string{"\"id\":1001", "\"username\":\"koyomi\"", "\"is_private\":true", "\"is_restricted\":true"} {
		if !strings.Contains(text, required) {
			t.Fatalf("restricted DTO missing %s: %s", required, text)
		}
	}
}

func withVisibility(profile model.PublicProfileRecord, visibility string) model.PublicProfileRecord {
	profile.ProfileVisibility = visibility
	return profile
}
