package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend/config"
	"backend/internal/classification/tools"
	importerProvider "backend/internal/importer/provider"
	"backend/pkg/logger"

	einoOpenAI "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// GameAgentInput carries the game identity handed to the agent. Title plus
// original title plus developer keep searches on target.
type GameAgentInput struct {
	GameID        uint   `json:"game_id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	Developer     string `json:"developer"`
	Publisher     string `json:"publisher"`
	VNDBID        string `json:"vndb_id"`
	BangumiID     string `json:"bangumi_id"`
}

// RunStats describes one agent execution; consumed by logs only.
type RunStats struct {
	Searches       int
	Fetches        int
	VNDBLookups    int
	BangumiLookups int
	LLMLatency     time.Duration
}

// Agent wraps the Eino ReAct agent and its four read-only tools.
type Agent struct {
	react *react.Agent
}

// Per-run budget limits. The system prompt repeats these numbers.
const (
	MaxSearchPerRun = 5
	MaxFetchPerRun  = 8
)

var (
	// ErrAgentDisabled is returned when CLASSIFICATION_AGENT_ENABLED=false.
	ErrAgentDisabled = errors.New("classification agent is disabled")
	// ErrInvalidModelOutput is returned when the model answer cannot be
	// parsed into the structured result.
	ErrInvalidModelOutput = errors.New("agent produced an invalid structured result")
)

type runStatsKey struct{}

// New builds the agent from config. Tools stay strictly read-only: web search,
// webpage fetching, VNDB and Bangumi lookups. No database tool exists, so the
// agent can never mutate game data.
func New(
	ctx context.Context,
	cfg *config.Classification,
	cache *tools.Cache,
	httpClient *http.Client,
) (*Agent, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, ErrAgentDisabled
	}
	if strings.TrimSpace(cfg.LLMAPIKey) == "" || strings.TrimSpace(cfg.LLMModel) == "" {
		return nil, errors.New("classification agent requires LLM_API_KEY and LLM_MODEL")
	}

	chatModel, err := einoOpenAI.NewChatModel(ctx, &einoOpenAI.ChatModelConfig{
		APIKey:  cfg.LLMAPIKey,
		BaseURL: cfg.LLMBaseURL,
		Model:   cfg.LLMModel,
		Timeout: 3 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("create classification chat model: %w", err)
	}

	searchTool := tools.NewSearchTool(newSearchProvider(cfg), cache)
	fetchTool := tools.NewFetchPageTool(cache)
	vndbTool := tools.NewVNDBTool(importerProvider.NewVNDBProvider(httpClient))
	bangumiTool := tools.NewBangumiTool(importerProvider.NewBangumiProvider(httpClient, "", ""))

	searchFn := func(ctx context.Context, input tools.SearchToolInput) (string, error) {
		stats := statsFrom(ctx)
		if stats != nil {
			if stats.Searches >= MaxSearchPerRun {
				return `{"results":[],"note":"search budget exhausted, conclude with the evidence you have"}`, nil
			}
			stats.Searches++
		}
		output, err := searchTool.Search(ctx, input)
		if err != nil {
			return softToolError(ctx, "search_web", err)
		}
		return output, nil
	}
	fetchFn := func(ctx context.Context, input tools.FetchPageToolInput) (string, error) {
		stats := statsFrom(ctx)
		if stats != nil {
			if stats.Fetches >= MaxFetchPerRun {
				return `{"title":"","content":"fetch budget exhausted, stop reading pages"}`, nil
			}
			stats.Fetches++
		}
		output, err := fetchTool.Fetch(ctx, input)
		if err != nil {
			return softToolError(ctx, "fetch_web_page", err)
		}
		return output, nil
	}
	vndbFn := func(ctx context.Context, input tools.VNDBToolInput) (string, error) {
		if stats := statsFrom(ctx); stats != nil {
			stats.VNDBLookups++
		}
		output, err := vndbTool.Lookup(ctx, input)
		if err != nil {
			return softToolError(ctx, "lookup_vndb", err)
		}
		return output, nil
	}
	bangumiFn := func(ctx context.Context, input tools.BangumiToolInput) (string, error) {
		if stats := statsFrom(ctx); stats != nil {
			stats.BangumiLookups++
		}
		output, err := bangumiTool.Lookup(ctx, input)
		if err != nil {
			return softToolError(ctx, "lookup_bangumi", err)
		}
		return output, nil
	}

	searchInvokable, err := utils.InferTool("search_web",
		"Web search. Returns up to 5 hits with title, url and snippet. Input: {query: string, limit?: int}.", searchFn)
	if err != nil {
		return nil, fmt.Errorf("register search tool: %w", err)
	}
	fetchInvokable, err := utils.InferTool("fetch_web_page",
		"Fetch one webpage and return its readable text. Input: {url: string}. Fails for blocked or non-HTML pages.", fetchFn)
	if err != nil {
		return nil, fmt.Errorf("register fetch tool: %w", err)
	}
	vndbInvokable, err := utils.InferTool("lookup_vndb",
		"Look up a game on VNDB by vndb_id (e.g. v20431) or by title. Evidence only, never a final verdict.", vndbFn)
	if err != nil {
		return nil, fmt.Errorf("register vndb tool: %w", err)
	}
	bangumiInvokable, err := utils.InferTool("lookup_bangumi",
		"Look up a game on Bangumi by subject_id or title. Auxiliary source, never a final verdict.", bangumiFn)
	if err != nil {
		return nil, fmt.Errorf("register bangumi tool: %w", err)
	}

	registered := []tool.BaseTool{searchInvokable, fetchInvokable, vndbInvokable, bangumiInvokable}
	// Eino's MaxStep counts every model AND tool node activation, so the
	// configured iteration budget maps to roughly twice the tool rounds.
	maxStep := cfg.MaxIterations*2 + 2
	if maxStep < 6 {
		maxStep = 6
	}
	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: registered,
		},
		MaxStep:         maxStep,
		MessageModifier: react.NewPersonaModifier(SystemPrompt),
	})
	if err != nil {
		return nil, fmt.Errorf("create react agent: %w", err)
	}
	return &Agent{react: reactAgent}, nil
}

// Run executes one classification research pass and returns the structured
// result. The agent has no database access; persisting happens in the service.
func (a *Agent) Run(ctx context.Context, input GameAgentInput) (*ClassificationResult, RunStats, error) {
	var stats RunStats
	ctx = context.WithValue(ctx, runStatsKey{}, &stats)

	started := time.Now()
	message, err := a.react.Generate(ctx, []*schema.Message{{
		Role:    schema.User,
		Content: buildUserPrompt(input),
	}})
	stats.LLMLatency = time.Since(started)
	if err != nil {
		return nil, stats, fmt.Errorf("classification agent failed: %w", err)
	}

	content := strings.TrimSpace(message.Content)
	if content == "" {
		return nil, stats, fmt.Errorf("%w: empty final answer", ErrInvalidModelOutput)
	}
	result, err := ExtractResult(content)
	if err != nil {
		logger.Warn("classification agent produced unparsable output",
			zap.Uint("game_id", input.GameID),
			zap.Int("searches", stats.Searches),
			zap.Int("fetches", stats.Fetches),
			zap.Int("vndb_lookups", stats.VNDBLookups),
			zap.Int("bangumi_lookups", stats.BangumiLookups),
			zap.String("output", truncateOutput(content)),
		)
		return nil, stats, fmt.Errorf("%w: %v", ErrInvalidModelOutput, err)
	}
	NormalizeResult(result)
	if err := ValidateResult(result); err != nil {
		return nil, stats, fmt.Errorf("%w: %v", ErrInvalidModelOutput, err)
	}
	return result, stats, nil
}

func statsFrom(ctx context.Context) *RunStats {
	stats, _ := ctx.Value(runStatsKey{}).(*RunStats)
	return stats
}

// softToolError converts a tool failure into a result the model can read, so
// one blocked page or failing search never aborts the whole research pass.
// Only context cancellation is kept fatal so Asynq can retry the task.
func softToolError(ctx context.Context, toolName string, err error) (string, error) {
	if ctx.Err() != nil {
		return "", err
	}
	summary := err.Error()
	if len(summary) > 300 {
		summary = summary[:300] + "…"
	}
	return fmt.Sprintf("{\"note\":\"%s failed: %s — skip this source and continue with others\"}",
		toolName, summary), nil
}

func newSearchProvider(cfg *config.Classification) tools.SearchProvider {
	switch strings.ToLower(cfg.SearchProvider) {
	case "tavily":
		return tools.NewTavilyProvider(cfg.SearchAPIKey)
	default:
		return nil
	}
}

func buildUserPrompt(input GameAgentInput) string {
	lines := []string{
		"Research the age rating of this visual novel:",
		fmt.Sprintf("- game_id: %d", input.GameID),
	}
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			lines = append(lines, fmt.Sprintf("- %s: %s", label, value))
		}
	}
	add("title", input.Title)
	add("original title", input.OriginalTitle)
	add("developer", input.Developer)
	add("publisher", input.Publisher)
	add("vndb_id", input.VNDBID)
	add("bangumi_id", input.BangumiID)
	lines = append(lines,
		"",
		"Determine whether this release is an R18 adult title. When the game "+
			"has a known VNDB or Bangumi id, start there; otherwise search the web.",
		"Reply with ONLY the structured JSON result.",
	)
	return strings.Join(lines, "\n")
}

func truncateOutput(content string) string {
	runes := []rune(content)
	if len(runes) > 600 {
		return string(runes[:600]) + "…"
	}
	return content
}
