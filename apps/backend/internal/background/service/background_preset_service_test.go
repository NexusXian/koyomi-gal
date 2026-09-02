package service

import (
	"errors"
	"testing"

	"backend/internal/background/model"
)

func TestValidatePreset(t *testing.T) {
	tests := []struct {
		name   string
		preset model.BackgroundPreset
		err    error
	}{
		{name: "valid", preset: model.BackgroundPreset{Key: "default-01", Name: "暮色花海", ImageURL: "presets/backgrounds/default-01.webp"}},
		{name: "empty name", preset: model.BackgroundPreset{Key: "default-01", ImageURL: "presets/backgrounds/default-01.webp"}, err: ErrInvalidPresetInput},
		{name: "empty image", preset: model.BackgroundPreset{Key: "default-01", Name: "暮色花海"}, err: ErrInvalidPresetInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validatePreset(&test.preset); !errors.Is(err, test.err) {
				t.Fatalf("expected %v, got %v", test.err, err)
			}
		})
	}
}

func TestResolveImageURL(t *testing.T) {
	svc := NewBackgroundPresetService(nil, "https://img.example.com/")
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "object key", value: "presets/backgrounds/default-01.webp", want: "https://img.example.com/presets/backgrounds/default-01.webp"},
		{name: "leading slash key", value: "/presets/bg.webp", want: "https://img.example.com/presets/bg.webp"},
		{name: "http url untouched", value: "http://cdn.example.com/bg.webp", want: "http://cdn.example.com/bg.webp"},
		{name: "https url untouched", value: "https://cdn.example.com/bg.webp", want: "https://cdn.example.com/bg.webp"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := svc.resolveImageURL(test.value); got != test.want {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
		})
	}
}

func TestGeneratePresetKey(t *testing.T) {
	first, err := generatePresetKey()
	if err != nil {
		t.Fatalf("generate preset key: %v", err)
	}
	second, err := generatePresetKey()
	if err != nil {
		t.Fatalf("generate preset key: %v", err)
	}
	if first == second {
		t.Fatalf("preset keys must be unique: %s", first)
	}
	if len(first) != len("preset-")+8 {
		t.Fatalf("unexpected preset key format: %s", first)
	}
}
