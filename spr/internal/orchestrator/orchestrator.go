package orchestrator

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/acheong08/hackeurope-spr/internal/tester"
	"github.com/acheong08/hackeurope-spr/pkg/models"
)

// ProgressCallback is called when a package's artifacts are successfully copied
type ProgressCallback func(pkgName, pkgVersion string, artifactCount int)

// Orchestrator manages GitHub Actions workflow runs for packages
type Orchestrator struct {
	client       *GitHubClient
	workflowFile string
	concurrency  int
	timeout      time.Duration
	progressCb   ProgressCallback
}

// PackageResult holds the result of analyzing a single package
type PackageResult struct {
	Package   models.Package
	Success   bool
	RunID     int64
	Artifacts []string
	Error     error
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(token, owner, repo, workflowFile string, concurrency int, timeout time.Duration, progressCb ProgressCallback) *Orchestrator {
	return &Orchestrator{
		client:       NewGitHubClient(token, owner, repo),
		workflowFile: workflowFile,
		concurrency:  concurrency,
		timeout:      timeout,
		progressCb:   progressCb,
	}
}

// RunPackages triggers workflows for all packages and collects results
func (o *Orchestrator) RunPackages(ctx context.Context, packages []models.Package, tempDir string, outputDir string) ([]PackageResult, error) {
	if len(packages) == 0 {
		return nil, fmt.Errorf("no packages to analyze")
	}

	fmt.Printf("Starting analysis of %d packages (max %d concurrent)\n", len(packages), o.concurrency)

	// Create a cancellable context for early termination
	ctx, cancel := context.WithCancel(ctx)

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
		fmt.Printf("  [%d/%d] %s@%s - ", completed, len(packages), result.Package.Name, result.Package.Version)
		if result.Error != nil {
			fmt.Printf("FAILED: %v\n", result.Error)
		} else {
			fmt.Printf("SUCCESS (%d artifacts)\n", len(result.Artifacts))
		}
	}

	// Check if we had any failures
	for _, result := range results {
		if result.Error != nil {
			return results, fmt.Errorf("analysis failed for %s@%s: %w", result.Package.Name, result.Package.Version, result.Error)
		}
	}

	fmt.Printf("\nCompleted analysis: %d/%d packages successful\n", len(packages)-failed, len(packages))

	// Wait for all artifact copy goroutines to complete
	fmt.Printf("Waiting for artifact copies to complete...\n")
	copyWg.Wait()
	fmt.Printf("All artifacts copied successfully\n")

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

	// 1. Trigger workflow
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
	fmt.Printf("    [Worker] Triggered workflow for %s@%s (run ID: %d)\n", pkg.Name, pkg.Version, triggerResp.RunID)

	// 2. Poll for completion
	run, err := o.pollWorkflowCompletion(ctx, triggerResp.RunID)
	if err != nil {
		result.Error = fmt.Errorf("failed to wait for completion: %w", err)
		return result
	}

	// 3. Check conclusion
	if run.Conclusion != "success" {
		result.Error = fmt.Errorf("workflow failed with conclusion: %s", run.Conclusion)
		return result
	}

	// 4. Download artifacts
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
				log.Printf("    [Worker] Skipping artifact copy for %s@%s: context cancelled\n", pkgName, pkgVersion)
				return
			default:
			}

			normalizedPkgName := tester.NormalizePackageName(pkgName)
			pkgOutputDir := filepath.Join(outputDir, fmt.Sprintf("%s@%s", normalizedPkgName, pkgVersion))
			if err := os.MkdirAll(pkgOutputDir, 0o755); err != nil {
				log.Printf("    [Worker] Warning: failed to create output directory for %s@%s: %v\n", pkgName, pkgVersion, err)
				return
			}

			for _, artifactPath := range artifactPaths {
				// Check context before each file copy
				select {
				case <-ctx.Done():
					log.Printf("    [Worker] Aborting artifact copy for %s@%s: context cancelled\n", pkgName, pkgVersion)
					return
				default:
					// Copy contents of artifact directory directly into pkgOutputDir (flatten structure)
					if err := copyDirContents(artifactPath, pkgOutputDir); err != nil {
						log.Printf("    [Worker] Warning: failed to copy artifact %s: %v\n", artifactPath, err)
					}
				}
			}
			log.Printf("    [Worker] Copied %d artifacts for %s@%s to output\n", len(artifactPaths), pkgName, pkgVersion)

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
		log.Printf("    [Worker] Polling workflow run %d (attempt %d)\n", runID, attempt)

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
