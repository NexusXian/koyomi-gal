package model

import "testing"

func TestNotificationTypesAndCategories(t *testing.T) {
	types := []NotificationType{
		TypePostCommented,
		TypeCommentReplied,
		TypePostLiked,
		TypeCommentLiked,
		TypeGalgameSubmitted,
		TypeGalgameApproved,
		TypeGalgameRejected,
		TypeResourceSubmitted,
		TypeResourceApproved,
		TypeResourceRejected,
		TypeResourceHidden,
		TypeResourceReported,
		TypeReportResolved,
		TypeReportRejected,
		TypePostModerated,
		TypeCommentModerated,
		TypeSystem,
	}
	for _, notificationType := range types {
		if !IsValidType(notificationType) {
			t.Fatalf("expected valid notification type %q", notificationType)
		}
	}
	if IsValidType("unknown") {
		t.Fatal("unknown notification type should be invalid")
	}

	categories := []NotificationCategory{CategoryInteraction, CategoryReview, CategoryModeration, CategorySystem}
	for _, category := range categories {
		if !IsValidCategory(category) {
			t.Fatalf("expected valid notification category %q", category)
		}
	}
	if IsValidCategory("unknown") {
		t.Fatal("unknown notification category should be invalid")
	}
}
