package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hackeurope/spr/internal/parser"
	"github.com/hackeurope/spr/internal/registry"
	"github.com/hackeurope/spr/pkg/models"
)

func main() {
	// Check for subcommands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "check":
		runCheckCommand(os.Args[2:])
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
	fmt.Println("  spr check [options]     Analyze package.json and upload to registry")
	fmt.Println("  spr test <command>      Generate test packages for behavioral analysis")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  check                   Parse package.json, build dependency graph, upload to registry")
	fmt.Println("  test generate           Generate test packages for a specific dependency")
	fmt.Println("  test list               List all generated test packages")
	fmt.Println("")
	fmt.Println("Run 'spr <command> --help' for more information on a command.")
}

func runCheckCommand(args []string) {
	// Create flag set for check command
	checkFlags := flag.NewFlagSet("check", flag.ExitOnError)

	var (
		packageJSONPath = checkFlags.String("package", "", "Path to package.json")
		lockfilePath    = checkFlags.String("lockfile", "", "Path to package-lock.json (optional)")
		outputPath      = checkFlags.String("output", "", "Output path for dependency graph JSON (optional)")

		// Registry upload flags
		uploadRegistry = checkFlags.Bool("upload", false, "Upload packages to registry")
		registryURL    = checkFlags.String("registry-url", "https://git.duti.dev", "Gitea registry URL")
		registryOwner  = checkFlags.String("registry-owner", "acheong08", "Gitea registry owner")
		registryToken  = checkFlags.String("registry-token", "", "Gitea registry token (required when --upload is set)")
	)

	checkFlags.Parse(args)

	// If no package.json specified, look in current directory
	if *packageJSONPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
			os.Exit(1)
		}

		path, err := parser.FindPackageJSON(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		*packageJSONPath = path
	}

	// Validate package.json
	if err := parser.ValidatePackageJSON(*packageJSONPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse package.json
	pkgJSON, err := parser.ParsePackageJSON(*packageJSONPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing package.json: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📦 Analyzing: %s@%s\n", pkgJSON.Name, pkgJSON.Version)

	// Build dependency graph
	var graph *models.DependencyGraph

	if *lockfilePath != "" {
		// Use provided lockfile
		lm := parser.NewLockfileManager()
		graph, err = lm.ParseLockfile(*lockfilePath, pkgJSON.ToPackage())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing lockfile: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Generate and parse lockfile
		fmt.Println("🔧 Generating lockfile...")
		graph, err = parser.BuildGraphFromPackageJSON(*packageJSONPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building dependency graph: %v\n", err)
			os.Exit(1)
		}
	}

	// Print summary
	fmt.Printf("\n📊 Dependency Graph Summary:\n")
	fmt.Printf("   Root: %s@%s\n", graph.RootPackage.Name, graph.RootPackage.Version)
	fmt.Printf("   Total packages: %d\n", len(graph.Nodes))

	// Get and display direct dependencies
	directDeps := graph.GetDirectDependencies()
	fmt.Printf("   Direct dependencies: %d\n\n", len(directDeps))

	if len(directDeps) > 0 {
		fmt.Println("🔗 Direct Dependencies:")
		for _, dep := range directDeps {
			depCount := len(dep.Dependencies)
			fmt.Printf("   - %s@%s (%d sub-dependencies)\n", dep.Name, dep.Version, depCount)
		}
	}

	// Upload to registry if requested
	if *uploadRegistry {
		if *registryToken == "" {
			fmt.Fprintf(os.Stderr, "Error: --registry-token is required for upload\n")
			os.Exit(1)
		}

		fmt.Println("\n🚀 Uploading to registry...")
		uploader := registry.NewUploader(*registryURL, *registryOwner, *registryToken)

		ctx := context.Background()
		if err := uploader.UploadGraph(ctx, graph); err != nil {
			fmt.Fprintf(os.Stderr, "Error uploading to registry: %v\n", err)
			os.Exit(1)
		}
	}

	// Output to file if requested
	if *outputPath != "" {
		data, err := json.MarshalIndent(graph, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling graph: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(*outputPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n💾 Dependency graph saved to: %s\n", *outputPath)
	}
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
