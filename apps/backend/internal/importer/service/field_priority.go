package service

import (
	"strings"

	galgameModel "backend/internal/galgame/model"
)

// descriptionSourcePriority ranks description sources for automatic
// enrichment. Unknown sources are treated as the lowest priority.
var descriptionSourcePriority = map[string]int{
	galgameModel.DescriptionSourceUnknown: 0,
	galgameModel.DescriptionSourceVNDB:    10,
	galgameModel.DescriptionSourceBangumi: 20,
	galgameModel.DescriptionSourceManual:  100,
}

// normalizeDescriptionSource maps a provider/model source value onto the
// description-source constants; anything unrecognized counts as unknown.
func normalizeDescriptionSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case galgameModel.DescriptionSourceVNDB:
		return galgameModel.DescriptionSourceVNDB
	case galgameModel.DescriptionSourceBangumi:
		return galgameModel.DescriptionSourceBangumi
	case galgameModel.DescriptionSourceManual:
		return galgameModel.DescriptionSourceManual
	default:
		return galgameModel.DescriptionSourceUnknown
	}
}

// descriptionSourceForImport records where an imported description came
// from. An empty description must never claim a provider source.
func descriptionSourceForImport(source, description string) string {
	if normalizeDescription(description) == "" {
		return galgameModel.DescriptionSourceUnknown
	}
	return normalizeDescriptionSource(source)
}

// shouldReplaceDescription decides whether an incoming description may
// replace the current one:
//
//  1. an empty incoming value never overwrites anything;
//  2. force (admin-requested) overwrites any non-empty incoming value;
//  3. an empty current description is always fillable;
//  4. manually maintained descriptions are never touched without force;
//  5. otherwise the incoming source must strictly outrank the current one,
//     so repeated syncs of the same source do not churn the row.
func shouldReplaceDescription(
	current string,
	currentSource string,
	incoming string,
	incomingSource string,
	force bool,
) bool {
	if incoming == "" {
		return false
	}
	if force {
		return true
	}
	if current == "" {
		return true
	}
	if normalizeDescriptionSource(currentSource) == galgameModel.DescriptionSourceManual {
		return false
	}
	currentPriority := descriptionSourcePriority[normalizeDescriptionSource(currentSource)]
	return descriptionSourcePriority[normalizeDescriptionSource(incomingSource)] > currentPriority
}
