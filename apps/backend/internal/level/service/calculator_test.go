package service

import (
	"testing"

	levelModel "backend/internal/level/model"
)

func levelConfigs() []levelModel.LevelConfig {
	return []levelModel.LevelConfig{
		{Level: 1, Name: "初见", MinExp: 0},
		{Level: 2, Name: "读者", MinExp: 100},
		{Level: 3, Name: "爱好者", MinExp: 500},
		{Level: 4, Name: "鉴赏家", MinExp: 1500},
		{Level: 5, Name: "资深鉴赏家", MinExp: 4000},
	}
}

func TestResolveLevel(t *testing.T) {
	tests := []struct {
		name     string
		totalExp int64
		current  int
		next     int // 0 means max level
	}{
		{name: "zero exp", totalExp: 0, current: 1, next: 2},
		{name: "level boundary low", totalExp: 100, current: 2, next: 3},
		{name: "mid level", totalExp: 3500, current: 4, next: 5},
		{name: "boundary exact", totalExp: 4000, current: 5, next: 0},
		{name: "beyond max", totalExp: 999999, current: 5, next: 0},
	}
	configs := levelConfigs()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current, next := resolveLevel(configs, test.totalExp)
			if current == nil {
				t.Fatalf("expected a level for %d exp", test.totalExp)
			}
			if current.Level != test.current {
				t.Fatalf("expected level %d, got %d", test.current, current.Level)
			}
			if test.next == 0 && next != nil {
				t.Fatalf("expected max level, got next %d", next.Level)
			}
			if test.next != 0 && (next == nil || next.Level != test.next) {
				t.Fatalf("expected next level %d, got %v", test.next, next)
			}
		})
	}
}

func TestResolveLevelEmptyConfigs(t *testing.T) {
	if current, next := resolveLevel(nil, 100); current != nil || next != nil {
		t.Fatalf("expected nil levels for empty configs, got %v %v", current, next)
	}
}

func TestLevelProgress(t *testing.T) {
	configs := levelConfigs()
	tests := []struct {
		name      string
		totalExp  int64
		progress  float64
		remaining int64
	}{
		{name: "level start", totalExp: 1500, progress: 0, remaining: 2500},
		{name: "mid progress", totalExp: 3480, progress: 0.792, remaining: 520},
		{name: "level end", totalExp: 4000, progress: 1, remaining: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current, next := resolveLevel(configs, test.totalExp)
			progress, remaining := levelProgress(current, next, test.totalExp)
			if diff := progress - test.progress; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("expected progress %v, got %v", test.progress, progress)
			}
			if remaining != test.remaining {
				t.Fatalf("expected remaining %d, got %d", test.remaining, remaining)
			}
		})
	}
}

func TestLevelProgressMaxLevel(t *testing.T) {
	configs := levelConfigs()
	current, next := resolveLevel(configs, 100000)
	if next != nil {
		t.Fatalf("expected max level")
	}
	progress, remaining := levelProgress(current, next, 100000)
	if progress != 1 || remaining != 0 {
		t.Fatalf("expected progress 1 and remaining 0, got %v %d", progress, remaining)
	}
}

func TestLevelProgressDegenerateSpan(t *testing.T) {
	configs := []levelModel.LevelConfig{
		{Level: 1, Name: "A", MinExp: 0},
		{Level: 2, Name: "B", MinExp: 0},
	}
	current, next := resolveLevel(configs, 0)
	progress, remaining := levelProgress(current, next, 0)
	if progress != 1 {
		t.Fatalf("expected progress 1 for zero span, got %v", progress)
	}
	if remaining != 0 {
		t.Fatalf("expected remaining 0 for zero span, got %d", remaining)
	}
}
