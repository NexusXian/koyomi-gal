package service

import (
	"errors"
	"testing"
	"time"

	"backend/internal/banner/model"
)

func TestValidateBanner(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	tests := []struct {
		name   string
		banner model.Banner
		err    error
	}{
		{name: "none", banner: model.Banner{Title: "banner", ImageURL: "/banner.jpg", LinkType: model.LinkTypeNone}},
		{name: "absolute url", banner: model.Banner{Title: "banner", ImageURL: "/banner.jpg", LinkType: model.LinkTypeURL, LinkValue: "https://example.com/page"}},
		{name: "entity id", banner: model.Banner{Title: "banner", ImageURL: "/banner.jpg", LinkType: model.LinkTypeGalgame, LinkValue: "10"}},
		{name: "empty title", banner: model.Banner{ImageURL: "/banner.jpg", LinkType: model.LinkTypeNone}, err: ErrInvalidBannerInput},
		{name: "empty image", banner: model.Banner{Title: "banner", LinkType: model.LinkTypeNone}, err: ErrInvalidBannerInput},
		{name: "none with value", banner: model.Banner{Title: "banner", ImageURL: "/banner.jpg", LinkType: model.LinkTypeNone, LinkValue: "1"}, err: ErrInvalidBannerLink},
		{name: "relative url", banner: model.Banner{Title: "banner", ImageURL: "/banner.jpg", LinkType: model.LinkTypeURL, LinkValue: "/page"}, err: ErrInvalidBannerLink},
		{name: "invalid entity id", banner: model.Banner{Title: "banner", ImageURL: "/banner.jpg", LinkType: model.LinkTypePost, LinkValue: "zero"}, err: ErrInvalidBannerLink},
		{name: "invalid schedule", banner: model.Banner{Title: "banner", ImageURL: "/banner.jpg", LinkType: model.LinkTypeNone, StartAt: &later, EndAt: &now}, err: ErrInvalidSchedule},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateBanner(&test.banner)
			if !errors.Is(err, test.err) {
				t.Fatalf("expected %v, got %v", test.err, err)
			}
		})
	}
}
