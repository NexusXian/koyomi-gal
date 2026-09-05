package agent

import (
	"strings"
	"testing"

	"backend/internal/classification/model"
)

func TestExtractResultFromFencedJSON(t *testing.T) {
	content := "Here is my finding:\n```json\n{\"classification\":\"r18\",\"confidence\":0.98,\"reason\":\"官网标注 18 岁以上对象\",\"conflict\":false,\"evidence\":[{\"source_type\":\"official\",\"title\":\"官网\",\"url\":\"https://official.example.com\",\"evidence\":\"18禁\"}]}\n```"
	result, err := ExtractResult(content)
	if err != nil {
		t.Fatalf("extract fenced json: %v", err)
	}
	if result.Classification != model.ClassificationR18 {
		t.Fatalf("unexpected classification: %s", result.Classification)
	}
	if len(result.Evidence) != 1 || result.Evidence[0].URL != "https://official.example.com" {
		t.Fatalf("unexpected evidence: %+v", result.Evidence)
	}
}

func TestExtractResultFromPlainJSON(t *testing.T) {
	content := `{"classification":"non_r18","confidence":0.96,"reason":"官方全年齢","conflict":false,"evidence":[]}`
	result, err := ExtractResult(content)
	if err != nil {
		t.Fatalf("extract plain json: %v", err)
	}
	if result.Classification != model.ClassificationNonR18 {
		t.Fatalf("unexpected classification: %s", result.Classification)
	}
}

func TestExtractResultRejectsProse(t *testing.T) {
	// The business logic must never parse natural language guesses.
	if _, err := ExtractResult("我认为这大概是一款 R18 游戏……"); err == nil {
		t.Fatal("expected error for prose-only answer")
	}
	if _, err := ExtractResult(""); err == nil {
		t.Fatal("expected error for empty answer")
	}
	if _, err := ExtractResult("{broken json"); err == nil {
		t.Fatal("expected error for broken json")
	}
}

func TestNormalizeResultClampsUnknownConfidence(t *testing.T) {
	result := &ClassificationResult{Classification: model.ClassificationUnknown, Confidence: 0.99}
	NormalizeResult(result)
	if result.Confidence != 0.8 {
		t.Fatalf("unknown confidence should clamp to 0.8, got %v", result.Confidence)
	}
}

func TestNormalizeResultDowngradesUnsupportedHighConfidence(t *testing.T) {
	result := &ClassificationResult{
		Classification: model.ClassificationR18,
		Confidence:     0.99,
		Evidence:       []ClassificationEvidence{{SourceType: model.SourceTypeOfficial, Title: "x", URL: "https://a.example", Evidence: "18禁"}},
	}
	NormalizeResult(result)
	if result.Confidence != 0.99 {
		t.Fatalf("supported high confidence must stay, got %v", result.Confidence)
	}

	noEvidence := &ClassificationResult{Classification: model.ClassificationR18, Confidence: 0.99}
	NormalizeResult(noEvidence)
	if noEvidence.Confidence != 0.9 {
		t.Fatalf("high confidence without evidence should downgrade to 0.9, got %v", noEvidence.Confidence)
	}
}

func TestValidateResultRules(t *testing.T) {
	valid := &ClassificationResult{
		Classification: model.ClassificationR18,
		Confidence:     0.95,
		Reason:         "官网 18禁",
		Evidence: []ClassificationEvidence{{
			SourceType: model.SourceTypeOfficial,
			Title:      "公式サイト",
			URL:        "https://example.com",
			Evidence:   "18禁",
		}},
	}
	if err := ValidateResult(valid); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}

	// Sakura no Toki style strong evidence stays valid.
	if err := ValidateResult(&ClassificationResult{
		Classification: model.ClassificationR18,
		Confidence:     0.99,
		Reason:         "公式サイトに 18歳以上対象 と記載",
		Evidence: []ClassificationEvidence{{
			SourceType: model.SourceTypeOfficial,
			URL:        "https://official.example.com",
			Evidence:   "18歳以上対象",
		}},
	}); err != nil {
		t.Fatalf("strong evidence rejected: %v", err)
	}

	cases := []struct {
		name   string
		result *ClassificationResult
	}{
		{"invalid enum", &ClassificationResult{Classification: model.ClassificationValue("mature"), Confidence: 0.5, Reason: "x"}},
		{"confidence negative", &ClassificationResult{Classification: model.ClassificationR18, Confidence: -0.1, Reason: "x"}},
		{"confidence too high", &ClassificationResult{Classification: model.ClassificationR18, Confidence: 1.2, Reason: "x"}},
		{"empty reason", &ClassificationResult{Classification: model.ClassificationUnknown, Confidence: 0.5}},
		{"unknown too confident", &ClassificationResult{Classification: model.ClassificationUnknown, Confidence: 0.81, Reason: "x"}},
		{"high confidence no evidence", &ClassificationResult{Classification: model.ClassificationNonR18, Confidence: 0.95, Reason: "x"}},
		{"bad source type", &ClassificationResult{
			Classification: model.ClassificationR18, Confidence: 0.95, Reason: "x",
			Evidence: []ClassificationEvidence{{SourceType: "fanblog", URL: "https://a.example", Evidence: "q"}},
		}},
		{"bad url", &ClassificationResult{
			Classification: model.ClassificationR18, Confidence: 0.95, Reason: "x",
			Evidence: []ClassificationEvidence{{SourceType: "official", URL: "ftp://x", Evidence: "q"}},
		}},
		{"empty evidence entry", &ClassificationResult{
			Classification: model.ClassificationR18, Confidence: 0.95, Reason: "x",
			Evidence: []ClassificationEvidence{{SourceType: "official", URL: "", Evidence: ""}},
		}},
	}
	for _, tc := range cases {
		if err := ValidateResult(tc.result); err == nil {
			t.Errorf("%s: expected validation error", tc.name)
		}
	}
}

func TestEvidenceMaxBound(t *testing.T) {
	evidence := make([]ClassificationEvidence, MaxResultEvidence+1)
	for i := range evidence {
		evidence[i] = ClassificationEvidence{SourceType: "other", URL: "https://e.example", Evidence: "x"}
	}
	err := ValidateResult(&ClassificationResult{
		Classification: model.ClassificationUnknown,
		Confidence:     0.5,
		Reason:         "too many entries",
		Evidence:       evidence,
	})
	if err == nil || !strings.Contains(err.Error(), "evidence") {
		t.Fatalf("expected evidence cap error, got %v", err)
	}
}
