package orchestrator

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/acheong08/hackeurope-spr/internal/aggregate"
	"github.com/acheong08/hackeurope-spr/internal/analysis"
	"github.com/acheong08/hackeurope-spr/internal/registry"
	"github.com/acheong08/hackeurope-spr/internal/tester"
	"github.com/acheong08/hackeurope-spr/pkg/models"
)

// ProgressCallback is called when a package's artifacts are successfully copied
type ProgressCallback func(pkgName, pkgVersion string, artifactCount int)

// LogCallback is an optional function for forwarding log messages (e.g. to WebSocket).
type LogCallback func(message, level string)

// Orchestrator manages GitHub Actions workflow runs for packages
type Orchestrator struct {
	client       *GitHubClient
	workflowFile string
	concurrency  int
	timeout      time.Duration
	progressCb   ProgressCallback
	logCb        LogCallback
	baselinePath string
	baseline     *aggregate.PerProcessStats
	apiKey       string // API key for AI analysis

	// Safe registry — nil means promotion is disabled
	safeUploader *registry.Uploader
	// Full dependency graph, needed for full-tree promotion
	graph *models.DependencyGraph
}

// PackageResult holds the result of analyzing a single package
type PackageResult struct {
	Package   models.Package
	Success   bool
	RunID     int64
	Artifacts []string
	Error     error
}

// NewOrchestrator creates a new orchestrator.
// safeUploader and graph are optional (nil disables safe-registry promotion).
func NewOrchestrator(token, owner, repo, workflowFile string, concurrency int, timeout time.Duration, progressCb ProgressCallback, baselinePath string, apiKey string, safeUploader *registry.Uploader, graph *models.DependencyGraph) *Orchestrator {
	o := &Orchestrator{
		client:       NewGitHubClient(token, owner, repo),
		workflowFile: workflowFile,
		concurrency:  concurrency,
		timeout:      timeout,
		progressCb:   progressCb,
		baselinePath: baselinePath,
		apiKey:       apiKey,
		safeUploader: safeUploader,
		graph:        graph,
	}

	// Load baseline if provided
	if baselinePath != "" {
		if baseline, err := aggregate.LoadPerProcessStats(baselinePath); err == nil {
			o.baseline = baseline
			o.logMsg(fmt.Sprintf("Loaded baseline from %s (%d processes)", baselinePath, baseline.CountProcesses), "info")
		} else {
			o.logMsg(fmt.Sprintf("Failed to load baseline from %s: %v", baselinePath, err), "warning")
		}
	}

	return o
}

// SetLogCallback sets an optional callback for forwarding log messages.
func (o *Orchestrator) SetLogCallback(cb LogCallback) {
	o.logCb = cb
}

// logMsg prints to console and optionally forwards via the log callback.
func (o *Orchestrator) logMsg(message, level string) {
	prefix := "[INFO]"
	switch level {
	case "success":
		prefix = "[SUCCESS]"
	case "warning":
		prefix = "[WARN]"
	case "error":
		prefix = "[ERROR]"
	}
	log.Printf("%s %s", prefix, message)
	if o.logCb != nil {
		o.logCb(message, level)
	}
}

// RunPackages triggers workflows for all packages and collects results
func (o *Orchestrator) RunPackages(ctx context.Context, packages []models.Package, tempDir string, outputDir string) ([]PackageResult, error) {
	if len(packages) == 0 {
		return nil, fmt.Errorf("no packages to analyze")
	}

	o.logMsg(fmt.Sprintf("Starting analysis of %d packages (max %d concurrent)", len(packages), o.concurrency), "info")

	// Create a cancellable context for early termination
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create channels for work distribution and result collection
	workChan := make(chan models.Package, len(packages))
	resultChan := make(chan PackageResult, len(packages))

	// Fill work queue
	for _, pkg := range packages {
		workChan <- pkg
	}
	close(workChan)

	// Create worker pool
	var wg sync.WaitGroup
	var copyWg sync.WaitGroup
	semaphore := make(chan struct{}, o.concurrency)

	for i := 0; i < o.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			o.worker(ctx, cancel, workerID, workChan, resultChan, semaphore, tempDir, outputDir, &copyWg)
		}(i)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []PackageResult
	completed := 0
	failed := 0
	hasFailure := false

	for result := range resultChan {
		completed++
		if result.Error != nil {
			failed++
			if !hasFailure {
				hasFailure = true
				// Cancel context on first failure (fail-fast)
				cancel()
			}
		}
		results = append(results, result)
		if result.Error != nil {
			o.logMsg(fmt.Sprintf("[%d/%d] %s@%s — FAILED: %v", completed, len(packages), result.Package.Name, result.Package.Version, result.Error), "error")
		} else {
			o.logMsg(fmt.Sprintf("[%d/%d] %s@%s — SUCCESS (%d artifacts)", completed, len(packages), result.Package.Name, result.Package.Version, len(result.Artifacts)), "success")
		}
	}

	// Check if we had any failures
	for _, result := range results {
		if result.Error != nil {
			return results, fmt.Errorf("analysis failed for %s@%s: %w", result.Package.Name, result.Package.Version, result.Error)
		}
	}

	o.logMsg(fmt.Sprintf("Completed analysis: %d/%d packages successful", len(packages)-failed, len(packages)), "info")

	// Wait for all artifact copy goroutines to complete
	o.logMsg("Waiting for artifact copies to complete...", "info")
	copyWg.Wait()
	o.logMsg("All artifacts copied successfully", "success")

	// Run AI security analysis if API key is provided
	if o.apiKey != "" && o.baseline != nil {
		if err := o.runAIAnalysis(ctx, packages, outputDir); err != nil {
			return results, fmt.Errorf("AI analysis failed: %w", err)
		}
	}

	// Promote full dependency tree to safe registry if all packages passed
	if err := o.promoteToSafeRegistry(ctx, packages, outputDir); err != nil {
		return results, fmt.Errorf("safe registry promotion failed: %w", err)
	}

	return results, nil
}

// worker processes packages from the work channel
func (o *Orchestrator) worker(ctx context.Context, cancel context.CancelFunc, workerID int, workChan <-chan models.Package, resultChan chan<- PackageResult, semaphore chan struct{}, tempDir string, outputDir string, copyWg *sync.WaitGroup) {
	for pkg := range workChan {
		// Check if context is cancelled before acquiring semaphore
		select {
		case <-ctx.Done():
			resultChan <- PackageResult{
				Package: pkg,
				Success: false,
				Error:   fmt.Errorf("cancelled due to previous error"),
			}
			continue
		default:
		}

		semaphore <- struct{}{} // Acquire
		result := o.analyzePackage(ctx, pkg, tempDir, outputDir, copyWg)
		<-semaphore // Release

		resultChan <- result
	}
}

// analyzePackage triggers workflow, polls for completion, and downloads artifacts
func (o *Orchestrator) analyzePackage(ctx context.Context, pkg models.Package, tempDir string, outputDir string, copyWg *sync.WaitGroup) PackageResult {
	result := PackageResult{
		Package: pkg,
	}

	// 1. Check for cached behavior.jsonl file
	normalizedPkgName := tester.NormalizePackageName(pkg.Name)
	cacheDir := filepath.Join("analysis-results", fmt.Sprintf("%s@%s", normalizedPkgName, pkg.Version))
	cachedBehaviorPath := filepath.Join(cacheDir, "behavior.jsonl")

	if _, err := os.Stat(cachedBehaviorPath); err == nil {
		// Cached file exists, use it instead of running workflow
		o.logMsg(fmt.Sprintf("Using cached behavior.jsonl for %s@%s", pkg.Name, pkg.Version), "info")

		// Copy cached file to tempDir for processing
		artifactDir := filepath.Join(tempDir, fmt.Sprintf("%s@%s", normalizedPkgName, pkg.Version))
		if err := os.MkdirAll(artifactDir, 0o755); err != nil {
			result.Error = fmt.Errorf("failed to create artifact directory: %w", err)
			return result
		}

		// Copy behavior.jsonl to artifact directory
		data, err := os.ReadFile(cachedBehaviorPath)
		if err != nil {
			result.Error = fmt.Errorf("failed to read cached behavior.jsonl: %w", err)
			return result
		}

		destPath := filepath.Join(artifactDir, "behavior.jsonl")
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			result.Error = fmt.Errorf("failed to write cached behavior.jsonl: %w", err)
			return result
		}

		// Generate diff.json if it doesn't exist in cache and baseline is available
		if o.baseline != nil {
			if _, err := os.Stat(filepath.Join(cacheDir, "diff.json")); os.IsNotExist(err) {
				if err := o.generateDiff(cachedBehaviorPath); err != nil {
					o.logMsg(fmt.Sprintf("Failed to generate diff for cached %s@%s: %v", pkg.Name, pkg.Version, err), "warning")
				}
			}
		}

		// Also copy diff.json if it exists
		cachedDiffPath := filepath.Join(cacheDir, "diff.json")
		if diffData, err := os.ReadFile(cachedDiffPath); err == nil {
			diffDestPath := filepath.Join(artifactDir, "diff.json")
			if err := os.WriteFile(diffDestPath, diffData, 0o644); err != nil {
				o.logMsg(fmt.Sprintf("Failed to copy cached diff.json: %v", err), "warning")
			}
		}

		// Copy cached files to outputDir so AI analysis and emitPackageResults can find them
		if outputDir != "" {
			pkgOutputDir := filepath.Join(outputDir, fmt.Sprintf("%s@%s", normalizedPkgName, pkg.Version))
			if err := os.MkdirAll(pkgOutputDir, 0o755); err != nil {
				o.logMsg(fmt.Sprintf("Failed to create output directory for cached %s@%s: %v", pkg.Name, pkg.Version, err), "warning")
			} else {
				// Copy behavior.jsonl
				if err := os.WriteFile(filepath.Join(pkgOutputDir, "behavior.jsonl"), data, 0o644); err != nil {
					o.logMsg(fmt.Sprintf("Failed to copy cached behavior.jsonl to output: %v", err), "warning")
				}
				// Copy diff.json if it exists
				if diffData, err := os.ReadFile(cachedDiffPath); err == nil {
					if err := os.WriteFile(filepath.Join(pkgOutputDir, "diff.json"), diffData, 0o644); err != nil {
						o.logMsg(fmt.Sprintf("Failed to copy cached diff.json to output: %v", err), "warning")
					}
				}
				// Copy ai-analysis.json if it exists in cache
				cachedAIPath := filepath.Join(cacheDir, "ai-analysis.json")
				if aiData, err := os.ReadFile(cachedAIPath); err == nil {
					if err := os.WriteFile(filepath.Join(pkgOutputDir, "ai-analysis.json"), aiData, 0o644); err != nil {
						o.logMsg(fmt.Sprintf("Failed to copy cached ai-analysis.json to output: %v", err), "warning")
					}
				}

				// Notify via callback if provided
				if o.progressCb != nil {
					o.progressCb(pkg.Name, pkg.Version, 1)
				}
			}
		}

		artifacts := []string{artifactDir}

		result.Success = true
		result.Artifacts = artifacts
		return result
	}

	// 2. Trigger workflow (no cache found)
	inputs := map[string]string{
		"package": pkg.Name,
		"version": pkg.Version,
	}

	triggerResp, err := o.client.TriggerWorkflow(ctx, o.workflowFile, inputs)
	if err != nil {
		result.Error = fmt.Errorf("failed to trigger workflow: %w", err)
		return result
	}

	result.RunID = triggerResp.RunID
	o.logMsg(fmt.Sprintf("Triggered workflow for %s@%s (run ID: %d)", pkg.Name, pkg.Version, triggerResp.RunID), "info")

	// 3. Poll for completion
	run, err := o.pollWorkflowCompletion(ctx, triggerResp.RunID)
	if err != nil {
		result.Error = fmt.Errorf("failed to wait for completion: %w", err)
		return result
	}

	// 4. Check conclusion
	if run.Conclusion != "success" {
		result.Error = fmt.Errorf("workflow failed with conclusion: %s", run.Conclusion)
		return result
	}

	// 5. Download artifacts
	artifacts, err := o.downloadArtifacts(ctx, run.ID, pkg, tempDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to download artifacts: %w", err)
		return result
	}

	// 5. Copy artifacts to output directory immediately (non-blocking, with context cancellation)
	if len(artifacts) > 0 && outputDir != "" {
		copyWg.Add(1)
		go func(ctx context.Context, artifactPaths []string, pkgName, pkgVersion string) {
			defer copyWg.Done()

			// Check if context is cancelled before starting
			select {
			case <-ctx.Done():
				o.logMsg(fmt.Sprintf("Skipping artifact copy for %s@%s: context cancelled", pkgName, pkgVersion), "warning")
				return
			default:
			}

			normalizedPkgName := tester.NormalizePackageName(pkgName)
			pkgOutputDir := filepath.Join(outputDir, fmt.Sprintf("%s@%s", normalizedPkgName, pkgVersion))
			if err := os.MkdirAll(pkgOutputDir, 0o755); err != nil {
				o.logMsg(fmt.Sprintf("Failed to create output directory for %s@%s: %v", pkgName, pkgVersion, err), "warning")
				return
			}

			for _, artifactPath := range artifactPaths {
				// Check context before each file copy
				select {
				case <-ctx.Done():
					o.logMsg(fmt.Sprintf("Aborting artifact copy for %s@%s: context cancelled", pkgName, pkgVersion), "warning")
					return
				default:
					// Copy contents of artifact directory directly into pkgOutputDir (flatten structure)
					if err := copyDirContents(artifactPath, pkgOutputDir); err != nil {
						o.logMsg(fmt.Sprintf("Failed to copy artifact %s: %v", artifactPath, err), "warning")
					}
				}
			}
			o.logMsg(fmt.Sprintf("Copied %d artifacts for %s@%s to output", len(artifactPaths), pkgName, pkgVersion), "info")

			// Generate diff.json if baseline is available
			if o.baseline != nil {
				behaviorPath := filepath.Join(pkgOutputDir, "behavior.jsonl")
				if _, err := os.Stat(behaviorPath); err == nil {
					if err := o.generateDiff(behaviorPath); err != nil {
						o.logMsg(fmt.Sprintf("Failed to generate diff for %s@%s: %v", pkgName, pkgVersion, err), "warning")
					}
				}
			}

			// Notify via callback if provided (sends to WebSocket)
			if o.progressCb != nil {
				o.progressCb(pkgName, pkgVersion, len(artifactPaths))
			}
		}(ctx, artifacts, pkg.Name, pkg.Version)
	}

	result.Success = true
	result.Artifacts = artifacts
	return result
}

// pollWorkflowCompletion polls the workflow status until completed or timeout
func (o *Orchestrator) pollWorkflowCompletion(ctx context.Context, runID int64) (*WorkflowRun, error) {
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	const pollInterval = 15 * time.Second
	attempt := 0

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return nil, fmt.Errorf("workflow polling cancelled")
			}
			return nil, fmt.Errorf("timeout waiting for workflow completion")
		default:
		}

		attempt++
		o.logMsg(fmt.Sprintf("Polling workflow run %d (attempt %d)", runID, attempt), "info")

		run, err := o.client.GetWorkflowRun(ctx, runID)
		if err != nil {
			return nil, fmt.Errorf("failed to get workflow status: %w", err)
		}

		if run.Status == "completed" {
			return run, nil
		}

		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return nil, fmt.Errorf("workflow polling cancelled")
			}
			return nil, fmt.Errorf("timeout waiting for workflow completion")
		case <-time.After(pollInterval):
			// Continue polling
		}
	}
}

// downloadArtifacts downloads and extracts all artifacts for a run
func (o *Orchestrator) downloadArtifacts(ctx context.Context, runID int64, pkg models.Package, tempDir string) ([]string, error) {
	artifacts, err := o.client.ListArtifacts(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}

	var downloaded []string

	for _, artifact := range artifacts {
		if artifact.Expired {
			continue
		}

		data, err := o.client.DownloadArtifact(ctx, artifact.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to download artifact %s: %w", artifact.Name, err)
		}

		// Extract zip
		extractDir := filepath.Join(tempDir, artifact.Name)
		if err := os.MkdirAll(extractDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}

		if err := extractZip(data, extractDir); err != nil {
			return nil, fmt.Errorf("failed to extract artifact: %w", err)
		}

		downloaded = append(downloaded, extractDir)
	}

	return downloaded, nil
}

// generateDiff creates a diff.json file from behavior.jsonl if it doesn't exist
func (o *Orchestrator) generateDiff(behaviorPath string) error {
	// Skip if no baseline loaded
	if o.baseline == nil {
		return nil
	}

	// Check if diff already exists
	diffPath := filepath.Join(filepath.Dir(behaviorPath), "diff.json")
	if _, err := os.Stat(diffPath); err == nil {
		// Diff already exists, skip
		return nil
	}

	// Process behavior.jsonl
	aggregator := aggregate.NewProcessAggregator()
	result, err := aggregator.ProcessFile(behaviorPath, filepath.Base(filepath.Dir(behaviorPath)))
	if err != nil {
		return fmt.Errorf("failed to process behavior.jsonl: %w", err)
	}

	// Apply deduplication
	deduped := aggregate.Dedup(result, o.baseline)

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(deduped, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal diff: %w", err)
	}

	// Write diff.json
	if err := os.WriteFile(diffPath, jsonBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write diff.json: %w", err)
	}

	return nil
}

// extractZip extracts a zip file to a directory
func extractZip(data []byte, destDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to read zip: %w", err)
	}

	for _, file := range reader.File {
		// Security check: prevent zip slip - validate BEFORE joining
		// filepath.IsLocal checks: not empty, not absolute, no .., no reserved names
		if !filepath.IsLocal(file.Name) {
			log.Printf("    [Worker] Warning: skipping dangerous path in zip: %s\n", file.Name)
			continue
		}

		path := filepath.Join(destDir, file.Name)

		// Double-check the resolved path is within destination
		if !isSubPath(path, destDir) {
			log.Printf("    [Worker] Warning: skipping path that escapes destination: %s\n", file.Name)
			continue
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

// isSubPath checks if path is a subdirectory of base
func isSubPath(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && rel != ".." && !filepath.HasPrefix(rel, "..")
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyDirContents copies the contents of src directory into dst directory
func copyDirContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

// runAIAnalysis runs AI security analysis on all packages with diffs
func (o *Orchestrator) runAIAnalysis(ctx context.Context, packages []models.Package, outputDir string) error {
	if o.apiKey == "" {
		return nil
	}

	// Create analyzer with concurrency limit of 5
	analyzer, err := analysis.NewAnalyzer(o.apiKey, 5)
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %w", err)
	}

	// Chain log callback so analyzer logs go to WebSocket too
	if o.logCb != nil {
		analyzer.SetLogCallback(func(message, level string) {
			o.logCb(message, level)
		})
	}

	// Build list of packages to analyze
	var packagesToAnalyze []analysis.PackageInfo
	for _, pkg := range packages {
		normalizedName := tester.NormalizePackageName(pkg.Name)
		pkgOutputDir := filepath.Join(outputDir, fmt.Sprintf("%s@%s", normalizedName, pkg.Version))
		diffPath := filepath.Join(pkgOutputDir, "diff.json")

		// Check if diff.json exists
		if _, err := os.Stat(diffPath); err == nil {
			packagesToAnalyze = append(packagesToAnalyze, analysis.PackageInfo{
				Name:      pkg.Name,
				Version:   pkg.Version,
				OutputDir: pkgOutputDir,
			})
		}
	}

	if len(packagesToAnalyze) == 0 {
		o.logMsg("No packages with diff.json found for AI analysis", "info")
		return nil
	}

	o.logMsg(fmt.Sprintf("Running AI security analysis on %d packages...", len(packagesToAnalyze)), "info")
	if err := analyzer.AnalyzePackages(ctx, packagesToAnalyze); err != nil {
		return err
	}

	return nil
}

// promoteToSafeRegistry promotes the full dependency graph to the safe registry
// after verifying that none of the analyzed packages were flagged as malicious.
// Packages with no ai-analysis.json (empty diff → no anomalies) are treated as safe.
func (o *Orchestrator) promoteToSafeRegistry(ctx context.Context, packages []models.Package, outputDir string) error {
	if o.safeUploader == nil || o.graph == nil {
		return nil
	}

	o.logMsg("Checking AI analysis results before promoting to safe registry...", "info")

	var blocked []string

	for _, pkg := range packages {
		normalizedName := tester.NormalizePackageName(pkg.Name)
		aiPath := filepath.Join(outputDir, fmt.Sprintf("%s@%s", normalizedName, pkg.Version), "ai-analysis.json")

		data, err := os.ReadFile(aiPath)
		if err != nil {
			if os.IsNotExist(err) {
				// No analysis file → no anomalies detected → treat as safe
				o.logMsg(fmt.Sprintf("%s@%s: no AI analysis (clean diff), treating as safe", pkg.Name, pkg.Version), "info")
				continue
			}
			return fmt.Errorf("failed to read ai-analysis.json for %s@%s: %w", pkg.Name, pkg.Version, err)
		}

		var assessment analysis.SecurityAssessment
		if err := json.Unmarshal(data, &assessment); err != nil {
			return fmt.Errorf("failed to parse ai-analysis.json for %s@%s: %w", pkg.Name, pkg.Version, err)
		}

		if assessment.IsMalicious {
			blocked = append(blocked, fmt.Sprintf("%s@%s (confidence=%.2f): %s",
				pkg.Name, pkg.Version, assessment.Confidence, assessment.Justification))
			o.logMsg(fmt.Sprintf("BLOCKED %s@%s — %s", pkg.Name, pkg.Version, assessment.Justification), "error")
		} else {
			o.logMsg(fmt.Sprintf("%s@%s: safe (confidence=%.2f)", pkg.Name, pkg.Version, assessment.Confidence), "success")
		}
	}

	if len(blocked) > 0 {
		o.logMsg(fmt.Sprintf("Promotion skipped — %d package(s) flagged as malicious:", len(blocked)), "warning")
		for _, b := range blocked {
			o.logMsg(fmt.Sprintf("  - %s", b), "warning")
		}
		// Don't return an error — let the caller continue so it can
		// emit results (e.g. red nodes in the frontend).
		return nil
	}

	o.logMsg("All packages passed analysis — promoting full dependency tree to safe registry...", "success")
	if err := o.safeUploader.UploadGraph(ctx, o.graph); err != nil {
		return fmt.Errorf("failed to promote packages to safe registry: %w", err)
	}

	o.logMsg("Successfully promoted dependency tree to safe registry", "success")
	return nil
}
