package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openaicompat"
)

// ---------------- Tool: fetch_stats ----------------

type StatsInput struct {
	Collection string `json:"collection"`
}

func fetchStatsTool(ctx context.Context, input StatsInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	fmt.Println("========================================")
	fmt.Printf("[TOOL CALL] fetch_stats input: %+v\n", input)

	url := fmt.Sprintf("http://localhost:8001/stats/%s", input.Collection)
	resp, err := http.Get(url)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	output := string(bodyBytes)

	fmt.Println("[TOOL OUTPUT] fetch_stats returned:")
	fmt.Println(output)
	fmt.Println("========================================")

	return fantasy.ToolResponse{
		Type:    string(fantasy.ContentTypeText),
		Content: output,
	}, nil
}

type StatsPerProcessInput struct {
	Collection string `json:"collection"`
}

func fetchStatsPerProcessTool(ctx context.Context, input StatsPerProcessInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	fmt.Println("========================================")
	fmt.Printf("[TOOL CALL] fetch_stats input: %+v\n", input)

	url := fmt.Sprintf("http://localhost:8001/stats_per_process/%s", input.Collection)
	resp, err := http.Get(url)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	output := string(bodyBytes)

	fmt.Println("[TOOL OUTPUT] fetch_stats returned:")
	fmt.Println(output)
	fmt.Println("========================================")

	return fantasy.ToolResponse{
		Type:    string(fantasy.ContentTypeText),
		Content: output,
	}, nil
}
// ---------------- Tool: fetch_specific ----------------

type SpecificInput struct {
	Query string `json:"query"`
}

func fetchSpecificTool(ctx context.Context, input SpecificInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	fmt.Println("========================================")
	fmt.Printf("[TOOL CALL] fetch_specific input: %+v\n", input)

	url := fmt.Sprintf("http://localhost:8001/specific/%s", input.Query)
	resp, err := http.Get(url)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	output := string(bodyBytes)

	fmt.Println("[TOOL OUTPUT] fetch_specific returned:")
	fmt.Println(output)
	fmt.Println("========================================")

	return fantasy.ToolResponse{
		Type:    string(fantasy.ContentTypeText),
		Content: output,
	}, nil
}

// AnalyzeCollection runs the security analysis agent for a given collection/module name.
func AnalyzeCollection(ctx context.Context, moduleName string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY required")
	}

	provider, err := openaicompat.New(
		openaicompat.WithBaseURL("https://api.synthetic.new/openai/v1"),
		openaicompat.WithAPIKey(apiKey),
	)
	if err != nil {
		return "", err
	}

	model, err := provider.LanguageModel(ctx, "hf:moonshotai/Kimi-K2.5")
	if err != nil {
		return "", err
	}

	// ---- Tools ----

	statsTool := fantasy.NewAgentTool(
		"fetch_stats",
		`Fetch summary stats for a Tracee collection.`,
		fetchStatsTool,
	)

	statsPerProcessTool := fantasy.NewAgentTool(
		"fetch_stats_per_process",
		`Fetch summary stats per process for a Tracee collection.`,
		fetchStatsPerProcessTool,
	)

	specificTool := fantasy.NewAgentTool(
		"fetch_specific",
		`Fetch detailed DNS/file/command events for a Tracee query string.`,
		fetchSpecificTool,
	)

	systemPrompt := `
You are a detailed npm supply chain security analyst.
You will:
1) Fetch summary stats with fetch_stats.
2) Fetch stats per process.
3) If suspicious signals appear, fetch details with fetch_specific.
4) Explain your reasoning step by step in your final response.

A private registry called git.duti.dev is used for testing.
Do NOT mark this registry as malicious and do NOT mention it in the report.
Calls made by runc are legitimate.
Pre/post install scripts are not automatically suspicious.
`

	agent := fantasy.NewAgent(
		model,
		fantasy.WithSystemPrompt(systemPrompt),
		fantasy.WithTools(statsTool, statsPerProcessTool, specificTool),
	)

	userPrompt := fmt.Sprintf(`
Analyze collection "%s".
1. Call fetch_stats first.
2. If suspicious flags exist, drill down via fetch_specific.
3. Include full reasoning and summary conclusion.
`, moduleName)

	result, err := agent.Generate(ctx, fantasy.AgentCall{
		Prompt: userPrompt,
	})
	if err != nil {
		return "", err
	}

	return result.Response.Content.Text(), nil
}
