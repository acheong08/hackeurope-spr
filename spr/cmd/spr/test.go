package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/acheong08/hackeurope-spr/internal/tester"
)

// TestGenerateCommand generates test packages for behavioral analysis
func TestGenerateCommand(args []string) {
	var (
		packageName    = ""
		packageVersion = ""
		outputDir      = "./test-packages"
		templatesDir   = ""
	)

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--package", "-p":
			if i+1 < len(args) {
				packageName = args[i+1]
				i++
			}
		case "--version", "-v":
			if i+1 < len(args) {
				packageVersion = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				outputDir = args[i+1]
				i++
			}
		case "--templates", "-t":
			if i+1 < len(args) {
				templatesDir = args[i+1]
				i++
			}
		}
	}

	// Get executable directory for default templates
	if templatesDir == "" {
		execPath, err := os.Executable()
		if err == nil {
			// Binary is in spr/, templates are in spr/templates/
			templatesDir = filepath.Join(filepath.Dir(execPath), "templates")
		} else {
			// Fallback to current working directory
			cwd, _ := os.Getwd()
			templatesDir = filepath.Join(cwd, "templates")
		}
	}

	// Validate required args
	if packageName == "" || packageVersion == "" {
		fmt.Fprintln(os.Stderr, "Usage: spr test generate --package <name> --version <version> [options]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  -p, --package <name>      Package name (required)")
		fmt.Fprintln(os.Stderr, "  -v, --version <version>   Package version (required)")
		fmt.Fprintln(os.Stderr, "  -o, --output <dir>        Output directory (default: ./test-packages)")
		fmt.Fprintln(os.Stderr, "  -t, --templates <dir>     Templates directory (default: ./templates)")
		os.Exit(1)
	}

	fmt.Printf("🔍 Detecting package type for %s@%s...\n", packageName, packageVersion)

	// Create generator
	generator := tester.NewGenerator(templatesDir)

	// Generate all test packages
	fmt.Printf("📝 Generating test packages...\n")
	dirs, err := generator.GenerateAll(packageName, packageVersion, outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error generating tests: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Generated %d test packages:\n", len(dirs))
	for _, dir := range dirs {
		fmt.Printf("   📦 %s\n", dir)
	}

	fmt.Println("\n🚀 Ready for GitHub Actions workflow!")
	fmt.Println("   Run: gh workflow run test-packages.yml -f package=" + packageName + " -f version=" + packageVersion)
}

// TestListCommand lists all generated test packages
func TestListCommand(args []string) {
	outputDir := "./test-packages"

	// Parse flags
	for i := 0; i < len(args); i++ {
		if args[i] == "--output" || args[i] == "-o" {
			if i+1 < len(args) {
				outputDir = args[i+1]
				i++
			}
		}
	}

	// Check if directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ Test packages directory not found: %s\n", outputDir)
		fmt.Fprintln(os.Stderr, "   Run 'spr test generate' first to create test packages")
		os.Exit(1)
	}

	generator := tester.NewGenerator("")
	packages, err := generator.ListGenerated(outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error listing packages: %v\n", err)
		os.Exit(1)
	}

	if len(packages) == 0 {
		fmt.Println("📭 No test packages found")
		return
	}

	fmt.Printf("📦 Found %d test packages:\n\n", len(packages))
	for _, pkg := range packages {
		fmt.Printf("   %s@%s\n", pkg.PackageName, pkg.PackageVersion)
	}
}
