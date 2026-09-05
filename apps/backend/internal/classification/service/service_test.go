package service

import (
	"testing"

	"backend/internal/classification/model"
)

func TestClassificationToAgeRatingMapping(t *testing.T) {
	cases := []struct {
		value model.ClassificationValue
		want  int16
		ok    bool
	}{
		{model.ClassificationR18, 3, true},
		{model.ClassificationNonR18, 1, true},
		{model.ClassificationUnknown, 0, false},
		{model.ClassificationValue("invalid"), 0, false},
	}
	for _, tc := range cases {
		got := classificationToAgeRating(tc.value)
		if tc.ok {
			if got == nil || *got != tc.want {
				t.Errorf("%s: want %d, got %+v", tc.value, tc.want, got)
			}
		} else if got != nil {
			t.Errorf("%s: expected nil mapping", tc.value)
		}
	}
}

func TestEvidenceWeightsCoverValidSources(t *testing.T) {
	for _, sourceType := range []string{
		model.SourceTypeOfficial, model.SourceTypeSteam, model.SourceTypeVNDB,
		model.SourceTypeBangumi, model.SourceTypeCERO, model.SourceTypeESRB,
		model.SourceTypePEGI, model.SourceTypeWikipedia, model.SourceTypeOther,
	} {
		if _, ok := evidenceWeights[sourceType]; !ok {
			t.Errorf("evidence weight missing for %s", sourceType)
		}
	}
}
