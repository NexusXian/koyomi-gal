package service

import (
	"backend/internal/user/dto"
	"backend/internal/user/model"
)

type ProfileAccessService struct{}

func NewProfileAccessService() *ProfileAccessService { return &ProfileAccessService{} }

func (s *ProfileAccessService) Resolve(profile *model.PublicProfileRecord, viewerID *uint) dto.ProfileAccess {
	if viewerID != nil && *viewerID == profile.ID {
		return dto.ProfileAccess{
			CanViewProfile: true, CanViewPosts: true, CanViewComments: true,
			CanViewRatings: true, CanViewFavorites: true, CanViewActivity: true,
			CanViewLocation: true, CanViewBirthday: true,
		}
	}
	canViewProfile := profile.ProfileVisibility == model.ProfileVisibilityPublic ||
		(profile.ProfileVisibility == model.ProfileVisibilityRegistered && viewerID != nil)
	if !canViewProfile {
		return dto.ProfileAccess{}
	}
	return dto.ProfileAccess{
		CanViewProfile: true,
		CanViewPosts:   profile.ShowPosts, CanViewComments: profile.ShowComments,
		CanViewRatings: profile.ShowRatings, CanViewFavorites: profile.ShowFavorites,
		CanViewActivity: profile.ShowActivity, CanViewLocation: profile.ShowLocation,
		CanViewBirthday: profile.ShowBirthday,
	}
}
