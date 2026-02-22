package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/acheong08/hackeurope-spr/internal/orchestrator"
	"github.com/acheong08/hackeurope-spr/internal/parser"
	"github.com/acheong08/hackeurope-spr/internal/registry"
	"github.com/acheong08/hackeurope-spr/pkg/models"
	"github.com/joho/godotenv"
)

// Config holds all environment/flag configuration for the spr CLI.
type Config struct {
	PackageJSONPath string
	LockfilePath    string
	OutputDir       string
	RegistryURL     string
	RegistryOwner   string
	RegistryToken   string
	GitHubToken     string
	RepoOwner       string
	RepoName        string
	WorkflowFile    string
	Concurrency     int
	TimeoutMinutes  int
	BaselinePath    string
	OpenAIAPIKey    string

	// Safe registry — packages are promoted here after passing AI analysis.
	// Leave SAFE_REGISTRY_TOKEN empty to disable promotion.
	SafeRegistryURL   string
	SafeRegistryToken string
	SafeRegistryOwner string
}

func loadConfig() *Config {
	// Load .env file if present; ignore error (file is optional).
	_ = godotenv.Load()

	return &Config{
		OutputDir:      getEnv("OUTPUT_DIR", "./analysis-results"),
		RegistryURL:    getEnv("REGISTRY_URL", "https://git.duti.dev"),
		RegistryOwner:  getEnv("REGISTRY_OWNER", "acheong08"),
		RegistryToken:  getEnv("REGISTRY_TOKEN", ""),
		GitHubToken:    getEnv("GITHUB_TOKEN", ""),
		RepoOwner:      getEnv("REPO_OWNER", "acheong08"),
		RepoName:       getEnv("REPO_NAME", "hackeurope-spr"),
		WorkflowFile:   getEnv("WORKFLOW_FILE", "analyze-package.yml"),
		Concurrency:    getEnvInt("CONCURRENCY", 5),
		TimeoutMinutes: getEnvInt("TIMEOUT_MINUTES", 5),
		BaselinePath:   getEnv("BASELINE_PATH", "safe-sample.json"),
		OpenAIAPIKey:   getEnv("OPENAI_API_KEY", ""),

		SafeRegistryURL:   getEnv("SAFE_REGISTRY_URL", "https://git.duti.dev"),
		SafeRegistryToken: getEnv("SAFE_REGISTRY_TOKEN", ""),
		SafeRegistryOwner: getEnv("SAFE_REGISTRY_OWNER", "secure"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultValue
}

func main() {
	// Check for subcommands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg := loadConfig()
	subcommand := os.Args[1]

	switch subcommand {
	case "check":
		runCheckCommand(cfg, os.Args[2:])
	case "test":
		runTestCommand(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("spr - Supply chain Package Runtime analyzer")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  spr check [options]     Analyze package.json, upload to registry, trigger workflows")
	fmt.Println("  spr test <command>      Generate test packages for behavioral analysis")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  check                   Full analysis pipeline")
	fmt.Println("  test generate           Generate test packages for a specific dependency")
	fmt.Println("  test list               List all generated test packages")
	fmt.Println("")
	fmt.Println("Run 'spr <command> -help' for more information on a command.")
}

func runCheckCommand(cfg *Config, args []string) {
	// Flag values start from config (env / .env defaults); CLI flags override.
	packageJSONPath := cfg.PackageJSONPath
	lockfilePath := cfg.LockfilePath

	// Parse flags manually (single dash); flags override env/config.
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-package":
			if i+1 < len(args) {
				packageJSONPath = args[i+1]
				i++
			}
		case "-lockfile":
			if i+1 < len(args) {
				lockfilePath = args[i+1]
				i++
			}
		case "-output":
			if i+1 < len(args) {
				cfg.OutputDir = args[i+1]
				i++
			}
		case "-registry-url":
			if i+1 < len(args) {
				cfg.RegistryURL = args[i+1]
				i++
			}
		case "-registry-owner":
			if i+1 < len(args) {
				cfg.RegistryOwner = args[i+1]
				i++
			}
		case "-registry-token":
			if i+1 < len(args) {
				cfg.RegistryToken = args[i+1]
				i++
			}
		case "-github-token":
			if i+1 < len(args) {
				cfg.GitHubToken = args[i+1]
				i++
			}
		case "-repo-owner":
			if i+1 < len(args) {
				cfg.RepoOwner = args[i+1]
				i++
			}
		case "-repo-name":
			if i+1 < len(args) {
				cfg.RepoName = args[i+1]
				i++
			}
		case "-workflow":
			if i+1 < len(args) {
				cfg.WorkflowFile = args[i+1]
				i++
			}
		case "-concurrency":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					cfg.Concurrency = n
				}
				i++
			}
		case "-timeout":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					cfg.TimeoutMinutes = n
				}
				i++
			}
		case "-baseline":
			if i+1 < len(args) {
				cfg.BaselinePath = args[i+1]
				i++
			}
		case "-help":
			printCheckUsage()
			os.Exit(0)
		}
	}

	// Validate required tokens early
	if cfg.RegistryToken == "" {
		fmt.Fprintln(os.Stderr, "Error: -registry-token is required (or set REGISTRY_TOKEN in environment / .env)")
		printCheckUsage()
		os.Exit(1)
	}

	if cfg.GitHubToken == "" {
		fmt.Fprintln(os.Stderr, "Error: -github-token is required (or set GITHUB_TOKEN in environment / .env)")
		printCheckUsage()
		os.Exit(1)
	}

	// Need either package.json or lockfile
	if packageJSONPath == "" && lockfilePath == "" {
		// Auto-detect in current directory
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
			os.Exit(1)
		}

		// Try package-lock.json first, then package.json
		if _, err := os.Stat(filepath.Join(cwd, "package-lock.json")); err == nil {
			lockfilePath = filepath.Join(cwd, "package-lock.json")
		} else {
			path, err := parser.FindPackageJSON(cwd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			packageJSONPath = path
		}
	}

	var pkgJSON *parser.PackageJSON
	var graph *models.DependencyGraph

	if lockfilePath != "" {
		// Using lockfile directly
		fmt.Printf("Using lockfile: %s\n", lockfilePath)

		// Extract root package from lockfile
		lm := parser.NewLockfileManager()
		rootPackage, err := lm.ExtractRootPackage(lockfilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting root from lockfile: %v\n", err)
			os.Exit(1)
		}

		// Parse lockfile to get full graph
		graph, err = lm.ParseLockfile(lockfilePath, rootPackage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing lockfile: %v\n", err)
			os.Exit(1)
		}

		// Create a synthetic pkgJSON for display purposes
		pkgJSON = &parser.PackageJSON{
			Name:    "package",
			Version: rootPackage.Version,
		}
	} else {
		// Using package.json
		// Validate package.json
		if err := parser.ValidatePackageJSON(packageJSONPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Parse package.json
		var err error
		pkgJSON, err = parser.ParsePackageJSON(packageJSONPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing package.json: %v\n", err)
			os.Exit(1)
		}

		// Build dependency graph
		if lockfilePath != "" {
			// Use provided lockfile
			lm := parser.NewLockfileManager()
			graph, err = lm.ParseLockfile(lockfilePath, pkgJSON.ToPackage())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing lockfile: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Generate and parse lockfile
			fmt.Println("Generating lockfile...")
			graph, err = parser.BuildGraphFromPackageJSON(packageJSONPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error building dependency graph: %v\n", err)
				os.Exit(1)
			}
		}
	}

	fmt.Printf("Analyzing: %s@%s\n", pkgJSON.Name, pkgJSON.Version)

	// Print summary
	fmt.Printf("\nDependency Graph Summary:\n")
	fmt.Printf("   Root: %s@%s\n", graph.RootPackage.Name, graph.RootPackage.Version)
	fmt.Printf("   Total packages: %d\n", len(graph.Nodes))

	directDeps := graph.GetDirectDependencies()
	fmt.Printf("   Direct dependencies: %d\n\n", len(directDeps))

	if len(directDeps) > 0 {
		fmt.Println("Direct Dependencies:")
		for _, dep := range directDeps {
			depCount := len(dep.Dependencies)
			fmt.Printf("   - %s@%s (%d sub-dependencies)\n", dep.Name, dep.Version, depCount)
		}
	}

	// Step 1: Upload all packages to registry
	fmt.Println("\nUploading packages to registry...")
	uploader := registry.NewUploader(cfg.RegistryURL, cfg.RegistryOwner, cfg.RegistryToken)

	ctx := context.Background()
	if err := uploader.UploadGraph(ctx, graph); err != nil {
		fmt.Fprintf(os.Stderr, "Error uploading to registry: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Successfully uploaded all packages")

	// Step 2: Trigger GitHub Actions for direct dependencies only
	if len(directDeps) == 0 {
		fmt.Println("\nNo direct dependencies to analyze")
		return
	}

	// Convert direct dependencies to []models.Package
	packagesToAnalyze := make([]models.Package, len(directDeps))
	for i, dep := range directDeps {
		packagesToAnalyze[i] = models.Package{
			Name:    dep.Name,
			Version: dep.Version,
		}
	}

	// Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Create temp directory for artifacts
	tempDir, err := os.MkdirTemp("", "spr-analysis-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	// Run analysis workflows
	fmt.Printf("\nTriggering analysis workflows for %d direct dependencies (max %d concurrent)...\n", len(packagesToAnalyze), cfg.Concurrency)

	// Build safe registry uploader (nil when token not configured → promotion disabled)
	var safeUploader *registry.Uploader
	if cfg.SafeRegistryToken != "" {
		safeUploader = registry.NewUploader(cfg.SafeRegistryURL, cfg.SafeRegistryOwner, cfg.SafeRegistryToken)
		fmt.Printf("Safe registry promotion enabled (%s / %s)\n", cfg.SafeRegistryURL, cfg.SafeRegistryOwner)
	} else {
		fmt.Println("Safe registry promotion disabled (SAFE_REGISTRY_TOKEN not set)")
	}

	orch := orchestrator.NewOrchestrator(
		cfg.GitHubToken,
		cfg.RepoOwner,
		cfg.RepoName,
		cfg.WorkflowFile,
		cfg.Concurrency,
		time.Duration(cfg.TimeoutMinutes)*time.Minute,
		nil, // No progress callback for CLI
		cfg.BaselinePath,
		cfg.OpenAIAPIKey,
		safeUploader,
		graph,
	)

	_, err = orch.RunPackages(ctx, packagesToAnalyze, tempDir, cfg.OutputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nAnalysis failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nAnalysis complete. Artifacts saved to: %s\n", cfg.OutputDir)
}

func printCheckUsage() {
	fmt.Println("Usage: spr check [options]")
	fmt.Println("")
	fmt.Println("Analyzes npm packages by uploading to registry and running behavioral tests.")
	fmt.Println("Requires either -package or -lockfile (auto-detects if neither specified).")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -package <path>        Path to package.json (generates lockfile if needed)")
	fmt.Println("  -lockfile <path>       Path to package-lock.json (uses existing lockfile)")
	fmt.Println("  -output <dir>          Output directory for artifacts (default: ./analysis-results)")
	fmt.Println("  -registry-url <url>    Gitea registry URL (default: https://git.duti.dev)")
	fmt.Println("  -registry-owner <own>  Gitea registry owner (default: acheong08)")
	fmt.Println("  -registry-token <tok>  Gitea registry token (required)")
	fmt.Println("  -github-token <tok>    GitHub token for workflow triggers (required)")
	fmt.Println("  -repo-owner <owner>    GitHub repo owner (default: acheong08)")
	fmt.Println("  -repo-name <name>      GitHub repo name (default: hackeurope)")
	fmt.Println("  -workflow <file>       Workflow file name (default: analyze-package.yml)")
	fmt.Println("  -concurrency <n>       Max concurrent workflows (default: 5)")
	fmt.Println("  -timeout <minutes>     Timeout per workflow in minutes (default: 5)")
	fmt.Println("  -baseline <path>       Path to baseline JSON for diff generation (default: safe-sample.json)")
	fmt.Println("  -help                  Show this help message")
}

func runTestCommand(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: spr test <command>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  generate    Generate test packages for a dependency")
		fmt.Fprintln(os.Stderr, "  list        List all generated test packages")
		os.Exit(1)
	}

	testSubcommand := args[0]

	switch testSubcommand {
	case "generate":
		TestGenerateCommand(args[1:])
	case "list":
		TestListCommand(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown test command: %s\n\n", testSubcommand)
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  generate    Generate test packages for a dependency")
		fmt.Fprintln(os.Stderr, "  list        List all generated test packages")
		os.Exit(1)
	}
}
