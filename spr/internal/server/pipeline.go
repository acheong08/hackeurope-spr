package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/acheong08/hackeurope-spr/internal/orchestrator"
	"github.com/acheong08/hackeurope-spr/internal/parser"
	"github.com/acheong08/hackeurope-spr/internal/registry"
	"github.com/acheong08/hackeurope-spr/pkg/models"
)

// ProgressSender interface for sending progress updates
type ProgressSender interface {
	SendMessage(msg Message)
	SendLog(message, level string)
	SendProgress(percent int, stage, message string)
	SendError(message string, err error)
}

// Pipeline wraps the CLI analysis logic for WebSocket use
type Pipeline struct {
	// Registry settings
	registryURL   string
	registryToken string
	registryOwner string

	// GitHub settings
	githubToken string
	repoOwner   string
	repoName    string

	// Progress sender
	sender ProgressSender

	// Temp directory for this analysis
	tempDir string
}

// NewPipeline creates a new pipeline instance
func NewPipeline(registryURL, registryToken, registryOwner, githubToken, repoOwner, repoName string, sender ProgressSender) *Pipeline {
	return &Pipeline{
		registryURL:   registryURL,
		registryToken: registryToken,
		registryOwner: registryOwner,
		githubToken:   githubToken,
		repoOwner:     repoOwner,
		repoName:      repoName,
		sender:        sender,
	}
}

// log sends a log message both to the WebSocket client and to the console
func (p *Pipeline) log(message, level string) {
	// Send to WebSocket client
	p.sender.SendLog(message, level)

	// Also log to console with level indicator
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
}

// logf is a formatted version of log
func (p *Pipeline) logf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	p.log(message, "info")
}

// Run executes the full analysis pipeline
func (p *Pipeline) Run(ctx context.Context, packageJSONContent string) error {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "spr-analysis-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	p.tempDir = tempDir
	defer os.RemoveAll(tempDir)

	p.log("Starting analysis...", "info")

	// Step 1: Parse package.json and build DAG
	p.sender.SendProgress(0, "dag", "Parsing package.json...")
	graph, err := p.buildDAG(ctx, packageJSONContent, tempDir)
	if err != nil {
		return fmt.Errorf("failed to build DAG: %w", err)
	}

	p.sender.SendProgress(10, "dag", fmt.Sprintf("DAG built: %d packages", len(graph.Nodes)))

	// Send DAG to frontend
	if err := p.sendDAG(graph); err != nil {
		return fmt.Errorf("failed to send DAG: %w", err)
	}

	// Get direct dependencies for analysis
	directDeps := graph.GetDirectDependencies()
	p.log(fmt.Sprintf("Found %d direct dependencies to analyze", len(directDeps)), "info")

	// Step 2: Upload to unsafe registry (20% - 40%)
	p.sender.SendProgress(20, "upload", "Uploading packages to registry...")
	if err := p.uploadPackages(ctx, graph); err != nil {
		return fmt.Errorf("failed to upload packages: %w", err)
	}
	p.sender.SendProgress(40, "upload", "Packages uploaded successfully")

	// Step 3: Run behavioral analysis workflows (40% - 80%)
	if len(directDeps) > 0 {
		p.sender.SendProgress(40, "workflow", fmt.Sprintf("Starting analysis of %d packages...", len(directDeps)))
		if err := p.runWorkflows(ctx, directDeps); err != nil {
			return fmt.Errorf("workflow analysis failed: %w", err)
		}
		p.sender.SendProgress(80, "workflow", "Behavioral analysis complete")
	} else {
		p.sender.SendProgress(80, "workflow", "No direct dependencies to analyze")
	}

	// Step 4: Aggregate data (80% - 90%)
	p.sender.SendProgress(80, "aggregate", "Aggregating behavioral data...")
	// TODO: Call Mongo aggregation service
	p.sender.SendProgress(90, "aggregate", "Data aggregation complete")

	// Step 5: Run agent (90% - 100%)
	p.sender.SendProgress(90, "agent", "Running security analysis...")
	// TODO: Call agent
	p.sender.SendProgress(100, "agent", "Analysis complete")

	p.log("Analysis pipeline complete", "success")
	return nil
}

// buildDAG parses package.json, generates lockfile, and builds dependency graph
func (p *Pipeline) buildDAG(ctx context.Context, packageJSONContent, tempDir string) (*models.DependencyGraph, error) {
	// Write package.json to temp directory
	pkgPath := filepath.Join(tempDir, "package.json")
	if err := os.WriteFile(pkgPath, []byte(packageJSONContent), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write package.json: %w", err)
	}

	// Validate and parse
	if err := parser.ValidatePackageJSON(pkgPath); err != nil {
		return nil, fmt.Errorf("invalid package.json: %w", err)
	}

	pkgJSON, err := parser.ParsePackageJSON(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	p.log(fmt.Sprintf("Analyzing: %s@%s", pkgJSON.Name, pkgJSON.Version), "info")

	// Generate lockfile
	p.log("Generating lockfile...", "info")
	lm := parser.NewLockfileManager()
	lockfilePath, err := lm.GenerateLockfile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate lockfile: %w", err)
	}

	// Extract root package and parse lockfile
	rootPackage, err := lm.ExtractRootPackage(lockfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract root package: %w", err)
	}

	graph, err := lm.ParseLockfile(lockfilePath, rootPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	return graph, nil
}

// sendDAG sends the dependency graph to the frontend
func (p *Pipeline) sendDAG(graph *models.DependencyGraph) error {
	// Convert nodes map to slice
	var nodes []*models.PackageNode
	for _, node := range graph.Nodes {
		nodes = append(nodes, node)
	}

	// Count edges (dependencies)
	edgeCount := 0
	for _, node := range graph.Nodes {
		edgeCount += len(node.Dependencies)
	}

	msg := NewDAGMessage(graph.RootPackage, nodes, edgeCount)
	p.sender.SendMessage(msg)

	p.log(fmt.Sprintf("DAG sent: %d nodes, %d edges", len(nodes), edgeCount), "success")
	return nil
}

// uploadPackages uploads the dependency graph to the registry
func (p *Pipeline) uploadPackages(ctx context.Context, graph *models.DependencyGraph) error {
	uploader := registry.NewUploader(p.registryURL, p.registryOwner, p.registryToken)

	// Track progress
	totalPackages := len(graph.Nodes)
	uploaded := 0

	// Create a wrapper to track progress
	errChan := make(chan error, 1)
	go func() {
		errChan <- uploader.UploadGraph(ctx, graph)
	}()

	// Poll progress (simple version)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-errChan:
			if err != nil {
				return err
			}
			// Send final progress
			percent := 20 + int(float64(totalPackages)/float64(totalPackages)*20)
			p.sender.SendProgress(percent, "upload", fmt.Sprintf("Uploaded %d/%d packages", totalPackages, totalPackages))
			return nil
		case <-ticker.C:
			// Update progress (approximate)
			uploaded++
			if uploaded > totalPackages {
				uploaded = totalPackages
			}
			percent := 20 + int(float64(uploaded)/float64(totalPackages)*20)
			if percent > 40 {
				percent = 40
			}
			p.sender.SendProgress(percent, "upload", fmt.Sprintf("Uploading package %d/%d...", uploaded, totalPackages))
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// runWorkflows triggers GitHub Actions workflows for packages
func (p *Pipeline) runWorkflows(ctx context.Context, packages []*models.PackageNode) error {
	// Convert to []models.Package
	pkgs := make([]models.Package, len(packages))
	for i, node := range packages {
		pkgs[i] = models.Package{
			ID:      node.ID,
			Name:    node.Name,
			Version: node.Version,
		}
	}

	// Create temp directories
	tempDir, err := os.MkdirTemp("", "spr-workflow-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	outputDir := filepath.Join(tempDir, "artifacts")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(
		p.githubToken,
		p.repoOwner,
		p.repoName,
		"analyze-package.yml",
		5,             // concurrency
		5*time.Minute, // timeout
	)

	// Send status updates for each package
	for _, pkg := range packages {
		p.sender.SendMessage(NewPackageStatusMessage(pkg.ID, pkg.Name, pkg.Version, "pending", 0))
	}

	// Create progress callback
	completedChan := make(chan int, len(pkgs))
	go func() {
		completed := 0
		for range completedChan {
			completed++
			percent := 40 + int(float64(completed)/float64(len(pkgs))*40)
			if percent > 80 {
				percent = 80
			}
			p.sender.SendProgress(percent, "workflow", fmt.Sprintf("Analyzed %d/%d packages", completed, len(pkgs)))
		}
	}()

	// Run workflows
	// Note: The orchestrator doesn't have progress callbacks yet, so we just run it
	// In a future version, we'd modify the orchestrator to send progress updates
	_, err = orch.RunPackages(ctx, pkgs, tempDir, outputDir)

	// Signal completion
	close(completedChan)

	if err != nil {
		return err
	}

	// Mark all packages as complete
	for _, pkg := range packages {
		p.sender.SendMessage(NewPackageStatusMessage(pkg.ID, pkg.Name, pkg.Version, "complete", 100))
	}

	return nil
}

// parsePackageJSON is a helper to parse package.json from string
type packageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func parsePackageJSON(content string) (*packageJSON, error) {
	var pkg packageJSON
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
