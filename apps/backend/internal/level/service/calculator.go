package service

import levelModel "backend/internal/level/model"

// resolveLevel returns the highest enabled level whose min_exp <= totalExp.
// When nothing matches (e.g. no LV1 with min_exp 0 exists) the lowest level is
// returned so the profile always renders. The next level is nil at max level.
func resolveLevel(
	configs []levelModel.LevelConfig,
	totalExp int64,
) (current *levelModel.LevelConfig, next *levelModel.LevelConfig) {
	if len(configs) == 0 {
		return nil, nil
	}
	// configs are expected sorted by level ascending; keep it defensive.
	sorted := make([]levelModel.LevelConfig, len(configs))
	copy(sorted, configs)
	for i := 1; i < len(sorted); i++ {
		if sorted[i].MinExp < sorted[i-1].MinExp {
			// fall back to an insertion sort for tiny config lists
			for j := i; j > 0 && sorted[j].MinExp < sorted[j-1].MinExp; j-- {
				sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			}
		}
	}
	current = &sorted[0]
	for i := range sorted {
		if totalExp >= sorted[i].MinExp {
			current = &sorted[i]
			if i+1 < len(sorted) {
				next = &sorted[i+1]
			} else {
				next = nil
			}
		}
	}
	return current, next
}

// levelProgress computes progress in [0,1] relative to the current level span
// and the remaining exp to the next level. Max level reports progress 1.
func levelProgress(current, next *levelModel.LevelConfig, totalExp int64) (progress float64, remaining int64) {
	if next == nil {
		return 1, 0
	}
	span := next.MinExp - current.MinExp
	if span <= 0 {
		// Degenerate configs (two levels with the same min_exp): report full
		// progress so division by zero never happens.
		return 1, next.MinExp - totalExp
	}
	gained := totalExp - current.MinExp
	if gained < 0 {
		gained = 0
	}
	if gained > span {
		gained = span
	}
	progress = float64(gained) / float64(span)
	remaining = next.MinExp - totalExp
	if remaining < 0 {
		remaining = 0
	}
	return progress, remaining
}
