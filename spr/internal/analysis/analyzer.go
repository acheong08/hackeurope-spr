package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"github.com/acheong08/hackeurope-spr/internal/aggregate"
)

const systemPrompt = `You are a security analyst specializing in software supply chain security. Your task is to analyze behavioral data from npm package installations and determine if the package exhibits malicious behavior.

CONTEXT:
You are analyzing "deduped" behavioral data, which means:
- The data has been compared against a baseline (normal npm installation behavior)
- Only ANOMALOUS behavior unique to this package is shown
- Common npm operations have been filtered out

WHAT TO LOOK FOR:
1. Network activity to suspicious IPs or domains (especially crypto pools, C2 servers)
2. File system access to sensitive locations (/etc, ~/.ssh, browser data, crypto wallets)
3. Execution of system commands (especially encoded/obfuscated commands)
4. Syscall patterns indicating process injection or privilege escalation
5. Unusual process spawning patterns
6. Access to environment variables containing secrets

JUDGMENT CRITERIA:
- Consider the package's stated purpose vs its behavior
- False positives are common for build tools that legitimately compile code
- Multiple suspicious indicators increase confidence
- Look for patterns typical of: cryptominers, data stealers, backdoors, ransomware

Provide a thorough justification explaining your reasoning.`

// Analyzer handles AI-powered security analysis of packages
type Analyzer struct {
	model     fantasy.LanguageModel
	semaphore chan struct{} // Limits concurrent analysis
}

// NewAnalyzer creates a new analyzer with the specified concurrency limit
func NewAnalyzer(apiKey string, concurrencyLimit int) (*Analyzer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required for AI analysis")
	}

	provider, err := openai.New(
		openai.WithBaseURL("https://cope.duti.dev"),
		openai.WithAPIKey(apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI provider: %w", err)
	}

	ctx := context.Background()
	model, err := provider.LanguageModel(ctx, "gpt-5-mini")
	if err != nil {
		return nil, fmt.Errorf("failed to create language model: %w", err)
	}

	return &Analyzer{
		model:     model,
		semaphore: make(chan struct{}, concurrencyLimit),
	}, nil
}

// AnalyzePackages performs AI security analysis on multiple packages in parallel
func (a *Analyzer) AnalyzePackages(ctx context.Context, packages []PackageInfo) error {
	if len(packages) == 0 {
		return nil
	}

	log.Printf("Starting AI security analysis for %d packages (max %d concurrent)", len(packages), cap(a.semaphore))

	var wg sync.WaitGroup
	errChan := make(chan error, len(packages))

	for _, pkg := range packages {
		wg.Add(1)
		go func(p PackageInfo) {
			defer wg.Done()

			// Check if context is cancelled
			select {
			case <-ctx.Done():
				errChan <- fmt.Errorf("analysis cancelled for %s@%s", p.Name, p.Version)
				return
			default:
			}

			// Acquire semaphore
			select {
			case a.semaphore <- struct{}{}:
			case <-ctx.Done():
				errChan <- fmt.Errorf("analysis cancelled for %s@%s", p.Name, p.Version)
				return
			}

			err := a.analyzePackage(ctx, p)
			<-a.semaphore // Release semaphore

			if err != nil {
				errChan <- fmt.Errorf("AI analysis failed for %s@%s: %w", p.Name, p.Version, err)
			}
		}(pkg)
	}

	// Wait for all analyses to complete
	wg.Wait()
	close(errChan)

	// Check for any errors (fail fast)
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0] // Return first error (fail fast)
	}

	log.Printf("Completed AI security analysis for %d packages", len(packages))
	return nil
}

// PackageInfo holds information about a package to analyze
type PackageInfo struct {
	Name      string
	Version   string
	OutputDir string // Directory containing diff.json
}

// analyzePackage performs AI analysis on a single package
func (a *Analyzer) analyzePackage(ctx context.Context, pkg PackageInfo) error {
	// Check if analysis already exists (caching)
	analysisPath := filepath.Join(pkg.OutputDir, "ai-analysis.json")
	if _, err := os.Stat(analysisPath); err == nil {
		log.Printf("  [AI] Using cached analysis for %s@%s", pkg.Name, pkg.Version)
		return nil
	}

	// Load diff.json
	diffPath := filepath.Join(pkg.OutputDir, "diff.json")
	diffData, err := os.ReadFile(diffPath)
	if err != nil {
		return fmt.Errorf("failed to read diff.json: %w", err)
	}

	// Parse diff to get structured data
	var deduped aggregate.DedupedProcessStats
	if err := json.Unmarshal(diffData, &deduped); err != nil {
		return fmt.Errorf("failed to parse diff.json: %w", err)
	}

	// Skip analysis if no anomalous behavior
	if len(deduped.PerProcess) == 0 {
		log.Printf("  [AI] No anomalous behavior for %s@%s, skipping analysis", pkg.Name, pkg.Version)
		assessment := SecurityAssessment{
			IsMalicious:   false,
			Confidence:    1.0,
			Justification: "No anomalous behavior detected. All activity matched baseline patterns.",
		}
		return a.saveAnalysis(pkg.OutputDir, assessment)
	}

	// Format diff data for the prompt
	prompt := formatAnalysisPrompt(pkg.Name, pkg.Version, &deduped)

	report := SecurityAssessment{}
	// Tool
	submitReportTool := fantasy.NewAgentTool(
		"submit_assessment",
		"Submit your security assessment for this package", func(
			_ context.Context,
			input SecurityAssessment,
			_ fantasy.ToolCall,
		) (fantasy.ToolResponse, error) {
			report = input
			return fantasy.ToolResponse{
				Content: "Command received",
			}, nil
		})

	// Call the agent
	agent := fantasy.NewAgent(a.model, fantasy.WithSystemPrompt(systemPrompt), fantasy.WithTools(submitReportTool))
	result, err := agent.Generate(ctx, fantasy.AgentCall{
		Prompt: prompt,
	})
	if err != nil {
		return fmt.Errorf("agent generation failed: %w", err)
	}

	log.Printf("  [AI] Agent response for %s@%s:\n%s", pkg.Name, pkg.Version,
		result.Response.Content.Text())

	// Save the analysis
	if err := a.saveAnalysis(pkg.OutputDir, report); err != nil {
		return fmt.Errorf("failed to save analysis: %w", err)
	}

	log.Printf("  [AI] Completed analysis for %s@%s - Malicious: %v (confidence: %.2f)",
		pkg.Name, pkg.Version, report.IsMalicious, report.Confidence)

	return nil
}

// formatAnalysisPrompt creates a detailed prompt from the deduped stats
func formatAnalysisPrompt(name, version string, stats *aggregate.DedupedProcessStats) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Analyze the security of npm package: %s@%s\n\n", name, version))
	sb.WriteString("DEDUPED BEHAVIORAL DATA (anomalous activity only):\n")
	sb.WriteString(fmt.Sprintf("Total unique processes: %d\n", stats.CountProcesses))
	sb.WriteString(fmt.Sprintf("Filtered from baseline: %d processes, %d files, %d commands, %d syscalls\n\n",
		stats.RemovedProcesses, stats.RemovedFiles, stats.RemovedCommands, stats.RemovedSyscalls))

	for procName, proc := range stats.PerProcess {
		sb.WriteString(fmt.Sprintf("\n=== PROCESS: %s ===\n", procName))

		if len(proc.SyscallProfile) > 0 {
			sb.WriteString("\nSyscalls:\n")
			for syscall, count := range proc.SyscallProfile {
				sb.WriteString(fmt.Sprintf("  - %s: %d calls\n", syscall, count))
			}
		}

		if len(proc.FileAccess) > 0 {
			sb.WriteString("\nFile Access:\n")
			for file, count := range proc.FileAccess {
				sb.WriteString(fmt.Sprintf("  - %s: %d accesses\n", file, count))
			}
		}

		if len(proc.ExecutedCommands) > 0 {
			sb.WriteString("\nExecuted Commands:\n")
			for cmd, count := range proc.ExecutedCommands {
				sb.WriteString(fmt.Sprintf("  - %s: %d executions\n", cmd, count))
			}
		}

		if len(proc.NetworkActivity.IPs) > 0 {
			sb.WriteString("\nNetwork Connections:\n")
			for ip, count := range proc.NetworkActivity.IPs {
				sb.WriteString(fmt.Sprintf("  - %s: %d connections\n", ip, count))
			}
		}

		if len(proc.NetworkActivity.DNSRecords) > 0 {
			sb.WriteString("\nDNS Lookups:\n")
			for domain, count := range proc.NetworkActivity.DNSRecords {
				sb.WriteString(fmt.Sprintf("  - %s: %d lookups\n", domain, count))
			}
		}
	}

	sb.WriteString("\n\nUse the submit_assessment tool to provide your security assessment.")

	return sb.String()
}

// saveAnalysis saves the assessment to ai-analysis.json
func (a *Analyzer) saveAnalysis(outputDir string, assessment SecurityAssessment) error {
	analysisPath := filepath.Join(outputDir, "ai-analysis.json")

	jsonBytes, err := json.MarshalIndent(assessment, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal assessment: %w", err)
	}

	if err := os.WriteFile(analysisPath, jsonBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write analysis file: %w", err)
	}

	return nil
}
