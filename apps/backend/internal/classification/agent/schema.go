package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"backend/internal/classification/model"
)

// ClassificationEvidence is one piece of source-backed evidence the model
// collected during research.
type ClassificationEvidence struct {
	SourceType string `json:"source_type" example:"official"`
	Title      string `json:"title" example:"サクラノ刻 公式サイト"`
	URL        string `json:"url" example:"https://example.com"`
	Evidence   string `json:"evidence" example:"18歳以上対象"`
}

// ClassificationResult is the only payload the agent may produce. The model
// must answer in this exact JSON shape; the service never parses prose.
type ClassificationResult struct {
	Classification model.ClassificationValue `json:"classification"`
	Confidence     float64                   `json:"confidence"`
	Reason         string                    `json:"reason"`
	Conflict       bool                      `json:"conflict"`
	Evidence       []ClassificationEvidence  `json:"evidence"`
}

const (
	// MaxResultEvidence caps the evidence list persisted per classification.
	MaxResultEvidence = 20
	maxReasonLength   = 4000
	maxEvidenceLength = 2000
)

var errNotJSON = errors.New("agent output is not valid structured JSON")

// ExtractResult parses the model's final message into a ClassificationResult.
// Markdown code fences and surrounding prose are tolerated, but the payload
// itself must be JSON matching the schema.
func ExtractResult(content string) (*ClassificationResult, error) {
	trimmed := strings.TrimSpace(content)
	lower := strings.ToLower(trimmed)
	if start := strings.Index(lower, "```"); start >= 0 {
		trimmed = trimmed[start+3:]
		if end := strings.Index(trimmed, "```"); end >= 0 {
			trimmed = trimmed[:end]
		}
		trimmed = strings.TrimSpace(trimmed)
		if jsonStart := strings.IndexByte(trimmed, '{'); jsonStart >= 0 {
			trimmed = trimmed[jsonStart:]
		}
	}

	start := strings.IndexByte(trimmed, '{')
	end := strings.LastIndexByte(trimmed, '}')
	if start < 0 || end <= start {
		return nil, fmt.Errorf("%w: no JSON object found", errNotJSON)
	}

	var result ClassificationResult
	if err := json.Unmarshal([]byte(trimmed[start:end+1]), &result); err != nil {
		return nil, fmt.Errorf("%w: %v", errNotJSON, err)
	}
	return &result, nil
}

// NormalizeResult clamps values that violate hard rules so a slightly off
// model answer still yields a usable record:
//   - unknown answers never carry confidence above 0.8
//   - high-confidence answers without evidence are downgraded
func NormalizeResult(result *ClassificationResult) {
	if result.Classification == model.ClassificationUnknown && result.Confidence > 0.8 {
		result.Confidence = 0.8
	}
	if result.Classification != model.ClassificationUnknown &&
		result.Confidence > 0.9 && len(result.Evidence) == 0 {
		result.Confidence = 0.9
	}
	if result.Confidence < 0 {
		result.Confidence = 0
	}
	if result.Confidence > 1 {
		result.Confidence = 1
	}
}

// ValidateResult enforces the structural contract agreed with the model.
func ValidateResult(result *ClassificationResult) error {
	if result == nil {
		return errors.New("agent returned an empty result")
	}
	if !result.Classification.Valid() {
		return fmt.Errorf("classification %q is not one of r18/r17/r15/r12/non_r18/unknown", result.Classification)
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		return fmt.Errorf("confidence %v is outside [0,1]", result.Confidence)
	}
	if strings.TrimSpace(result.Reason) == "" {
		return errors.New("reason is required")
	}
	if len(result.Reason) > maxReasonLength {
		return fmt.Errorf("reason exceeds %d characters", maxReasonLength)
	}
	if result.Classification == model.ClassificationUnknown && result.Confidence > 0.8 {
		return fmt.Errorf("unknown classification cannot carry confidence above 0.8")
	}
	if result.Classification != model.ClassificationUnknown &&
		result.Confidence > 0.9 && len(result.Evidence) == 0 {
		return errors.New("confidence above 0.9 requires at least one evidence entry")
	}
	if len(result.Evidence) > MaxResultEvidence {
		return fmt.Errorf("evidence exceeds %d entries", MaxResultEvidence)
	}
	for i, evidence := range result.Evidence {
		sourceType := strings.ToLower(strings.TrimSpace(evidence.SourceType))
		if !model.ValidSourceType(sourceType) {
			return fmt.Errorf("evidence[%d] has unknown source_type %q", i, evidence.SourceType)
		}
		if evidence.URL != "" {
			parsed, err := url.Parse(evidence.URL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return fmt.Errorf("evidence[%d] has invalid URL %q", i, evidence.URL)
			}
		}
		if len(evidence.Evidence) > maxEvidenceLength {
			return fmt.Errorf("evidence[%d] text exceeds %d characters", i, maxEvidenceLength)
		}
		if strings.TrimSpace(evidence.Title) == "" && strings.TrimSpace(evidence.Evidence) == "" {
			return fmt.Errorf("evidence[%d] has neither a title nor a quote", i)
		}
	}
	return nil
}
