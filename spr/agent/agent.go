package main

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

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY required")
		return
	}

	provider, err := openaicompat.New(openaicompat.WithBaseURL("https://api.synthetic.new/openai/v1"), openaicompat.WithAPIKey(apiKey))
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	model, err := provider.LanguageModel(ctx, "hf:moonshotai/Kimi-K2.5")
	if err != nil {
		panic(err)
	}

	// ---- Tools ----

	statsTool := fantasy.NewAgentTool(
		"fetch_stats",
		`Fetch summary stats for a Tracee collection.
Example: fetch_stats({"collection":"tracee_20260221_140853_51fac1"})`,
		fetchStatsTool,
	)
	
	statsPerProcessTool := fantasy.NewAgentTool(
		"fetch_stats_per_process",
		`Fetch summary stats per process for a Tracee collection.
Example: fetch_stats_per_process({"collection":"tracee_20260221_140853_51fac1"})`,
		fetchStatsPerProcessTool,
	)

	specificTool := fantasy.NewAgentTool(
		"fetch_specific",
		`Fetch detailed DNS/file/command events for a Tracee query string.
Examples:
fetch_specific({"query":"tracee_...?...dns=git.duti.dev"})
fetch_specific({"query":"tracee_...?...command=/usr/bin/sh"})
fetch_specific({"query":"tracee_...?...file=/etc/passwd"})`,
		fetchSpecificTool,
	)

	// ---- System Prompt ----

	systemPrompt := `
You are a detailed npm supply chain security analyst.
You will:
1) Fetch summary stats with fetch_stats.
2) Fetch stats per process to see what processes are being active
3) If suspicious signals appear, fetch details with fetch_specific.
4) EXPLAIN your reasoning step by step in your final response.

Always show which tools you called (with examples) in your reasoning.
`

	agent := fantasy.NewAgent(
		model,
		fantasy.WithSystemPrompt(systemPrompt),
		fantasy.WithTools(statsTool, statsPerProcessTool, specificTool),
	)

	// ---- User Prompt ----

	userPrompt := `
Analyze collection "tracee_20260221_140853_51fac1".
1. Call fetch_stats first.
2. If there are suspicious flags, drill down via fetch_specific.
3. Include full reasoning and summary conclusion.
`

	// ---- Run Agent ----

	result, err := agent.Generate(ctx, fantasy.AgentCall{Prompt: userPrompt})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("========== MODEL REASONING + CONCLUSION ==========")
	fmt.Println(result.Response.Content.Text())
}
