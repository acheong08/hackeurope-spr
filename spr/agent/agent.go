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

//
// =========================
// Event Streaming Types
// =========================
//

type AgentEventType string

const (
	EventToolCall   AgentEventType = "tool_call"
	EventToolResult AgentEventType = "tool_result"
	EventReasoning  AgentEventType = "reasoning"
	EventDecision   AgentEventType = "decision"
	EventError      AgentEventType = "error"
	EventDone       AgentEventType = "done"
)

type AgentEvent struct {
	Type    AgentEventType `json:"type"`
	Message string         `json:"message"`
}

//
// =========================
// Tool Input Types
// =========================
//

type StatsInput struct {
	Collection string `json:"collection"`
}

type StatsPerProcessInput struct {
	Collection string `json:"collection"`
}

type SpecificInput struct {
	Query string `json:"query"`
}

type DecisionInput struct {
	Decision string `json:"decision"`
}

//
// =========================
// AnalyzeCollection
// =========================
//

func AnalyzeCollection(
	ctx context.Context,
	moduleName string,
	events chan<- AgentEvent,
) error {

	defer close(events)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY required")
	}

	provider, err := openaicompat.New(
		openaicompat.WithBaseURL("https://api.synthetic.new/openai/v1"),
		openaicompat.WithAPIKey(apiKey),
	)
	if err != nil {
		return err
	}

	model, err := provider.LanguageModel(ctx, "hf:moonshotai/Kimi-K2.5")
	if err != nil {
		return err
	}

	//
	// =========================
	// Analysis Tools
	// =========================
	//

	statsTool := fantasy.NewAgentTool(
		"fetch_stats",
		`Fetch summary stats for a Tracee collection. Example: fetch_stats({"collection":"module-xxx"})`,
		func(ctx context.Context, input StatsInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {

			events <- AgentEvent{
				Type:    EventToolCall,
				Message: fmt.Sprintf("fetch_stats called with %+v", input),
			}

			url := fmt.Sprintf("http://localhost:8001/stats/%s", input.Collection)
			resp, err := http.Get(url)
			if err != nil {
				events <- AgentEvent{Type: EventError, Message: err.Error()}
				return fantasy.ToolResponse{}, err
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			output := string(bodyBytes)

			events <- AgentEvent{
				Type:    EventToolResult,
				Message: output,
			}

			return fantasy.ToolResponse{
				Type:    string(fantasy.ContentTypeText),
				Content: output,
			}, nil
		},
	)

	statsPerProcessTool := fantasy.NewAgentTool(
		"fetch_stats_per_process",
		`Fetch summary stats per process for a Tracee collection. Example: fetch_stats_per_process({"collection":"module-xxx"})`,
		func(ctx context.Context, input StatsPerProcessInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {

			events <- AgentEvent{
				Type:    EventToolCall,
				Message: fmt.Sprintf("fetch_stats_per_process called with %+v", input),
			}

			url := fmt.Sprintf("http://localhost:8001/stats_per_process/%s", input.Collection)
			resp, err := http.Get(url)
			if err != nil {
				events <- AgentEvent{Type: EventError, Message: err.Error()}
				return fantasy.ToolResponse{}, err
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			output := string(bodyBytes)

			events <- AgentEvent{
				Type:    EventToolResult,
				Message: output,
			}

			return fantasy.ToolResponse{
				Type:    string(fantasy.ContentTypeText),
				Content: output,
			}, nil
		},
	)

	specificTool := fantasy.NewAgentTool(
		"fetch_specific",
		`Fetch detailed DNS/file/command events for a Tracee query string. Examples:
fetch_specific({"query":"tracee_...?dns=git.duti.dev"})
fetch_specific({"query":"tracee_...?command=/usr/bin/sh"})
fetch_specific({"query":"tracee_...?file=/etc/passwd"})`,
		func(ctx context.Context, input SpecificInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {

			events <- AgentEvent{
				Type:    EventToolCall,
				Message: fmt.Sprintf("fetch_specific called with %+v", input),
			}

			url := fmt.Sprintf("http://localhost:8001/specific/%s", input.Query)
			resp, err := http.Get(url)
			if err != nil {
				events <- AgentEvent{Type: EventError, Message: err.Error()}
				return fantasy.ToolResponse{}, err
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			output := string(bodyBytes)

			events <- AgentEvent{
				Type:    EventToolResult,
				Message: output,
			}

			return fantasy.ToolResponse{
				Type:    string(fantasy.ContentTypeText),
				Content: output,
			}, nil
		},
	)

	//
	// =========================
	// Phase 1: Analysis Agent
	// =========================
	//

	systemPrompt := `
You are a detailed npm supply chain security analyst.
You will:
1) Fetch summary stats with fetch_stats.
2) Fetch stats per process.
3) If suspicious signals appear, fetch details with fetch_specific.
4) EXPLAIN your reasoning step by step in your final response.
A private registry is used called git.duti.dev - it is used for package testing, you are getting the test results DON'T MARK THIS REGISTRY (git.duti.dev) ITSELF AS MALICIOUS AND DON'T MENTION THIS REGISTRY (git.duti.dev) IN THE REPORT
Look if the module is downloading suspicious modules, tries to access sensitive files, steal credentials, priveledge escalate, spawn shells or run programs that look suspicious
Trace contains the system activity as well like runc activity
THE CALLS MADE BY runc ARE TO BE CONSIDERED LEGIT
pre/post install scripts are not immidiately suspicious, UNLESS THEY ARE DOING SUSPICIOUS BEHAVIOUR
ANY CALLS TO git.duti.dev ARE NOT THEMSELVES MALICIOUS

Look for package behavior in it's pre/postinstall scripts as well as the tests analyze and search for untypical behavior for node packages.
A private registry called git.duti.dev is used for testing.
Do NOT mark this registry as malicious and do NOT mention it in the report.
Calls made by runc are legitimate.
Pre/post install scripts are not automatically suspicious.
`

	analysisAgent := fantasy.NewAgent(
		model,
		fantasy.WithSystemPrompt(systemPrompt),
		fantasy.WithTools(statsTool, statsPerProcessTool, specificTool),
	)

	userPrompt := fmt.Sprintf(`
Analyze collection "%s".
Call fetch_stats first.
If suspicious flags exist, drill down via fetch_specific.
Include full reasoning and summary conclusion.
`, moduleName)

	analysisResult, err := analysisAgent.Generate(ctx, fantasy.AgentCall{
		Prompt: userPrompt,
	})
	if err != nil {
		return err
	}

	analysisText := analysisResult.Response.Content.Text()

	events <- AgentEvent{
		Type:    EventReasoning,
		Message: analysisText,
	}

	//
	// =========================
	// Phase 2: Decision Agent
	// =========================
	//

	decisionTool := fantasy.NewAgentTool(
		"decision",
		`Provide the final decision.
Return ONLY one of:
- legit
- malicious`,
		func(ctx context.Context, input DecisionInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {

			events <- AgentEvent{
				Type:    EventDecision,
				Message: input.Decision,
			}

			return fantasy.ToolResponse{
				Type:    string(fantasy.ContentTypeText),
				Content: input.Decision,
			}, nil
		},
	)

	decisionSystemPrompt := `
You are a classifier.
Based on the provided analysis,
call the decision tool exactly once with:
- legit
or
- malicious

Do not explain.
Do not add text.
Only call the tool.
`

	decisionAgent := fantasy.NewAgent(
		model,
		fantasy.WithSystemPrompt(decisionSystemPrompt),
		fantasy.WithTools(decisionTool),
	)

	decisionPrompt := fmt.Sprintf(`
Here is the full analysis:

%s

Classify the module.
`, analysisText)

	_, err = decisionAgent.Generate(ctx, fantasy.AgentCall{
		Prompt: decisionPrompt,
	})
	if err != nil {
		return err
	}

	events <- AgentEvent{
		Type:    EventDone,
		Message: "analysis_complete",
	}

	return nil
}
