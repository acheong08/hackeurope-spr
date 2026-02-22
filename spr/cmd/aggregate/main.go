package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/acheong08/hackeurope-spr/internal/aggregate"
)

func main() {
	var (
		inputFile   = flag.String("input", "", "Path to behavior.jsonl file (required if -dir not used)")
		dirPath     = flag.String("dir", "", "Path to directory containing package subdirectories with behavior.jsonl files")
		collection  = flag.String("collection", "default", "Collection name (used when -input specified)")
		outputFile  = flag.String("output", "", "Output JSON file (optional, defaults to stdout; used with -input)")
		dedupSource = flag.String("dedup-source", "", "Path to safe baseline JSON file for deduplication (required for batch mode)")
		help        = flag.Bool("help", false, "Show help")
	)

	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	// Load dedup source if provided
	var baseline *aggregate.PerProcessStats
	if *dedupSource != "" {
		if _, err := os.Stat(*dedupSource); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Dedup source file not found: %s\n", *dedupSource)
			os.Exit(1)
		}
		var err error
		baseline, err = aggregate.LoadPerProcessStats(*dedupSource)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading dedup source: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Loaded baseline from %s (%d processes)\n", *dedupSource, baseline.CountProcesses)
	}

	// Batch mode: process directory
	if *dirPath != "" {
		if err := processDirectory(*dirPath, baseline); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing directory: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Single file mode
	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Either -input or -dir must be specified\n")
		printUsage()
		os.Exit(1)
	}

	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file not found: %s\n", *inputFile)
		os.Exit(1)
	}

	processSingleFile(*inputFile, *collection, *outputFile, baseline)
}

func processSingleFile(inputFile, collection, outputFile string, baseline *aggregate.PerProcessStats) {
	startTime := time.Now()
	fmt.Fprintf(os.Stderr, "Processing %s...\n", inputFile)

	// Always use per-process aggregation
	aggregator := aggregate.NewProcessAggregator()
	result, err := aggregator.ProcessFile(inputFile, collection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	fmt.Fprintf(os.Stderr, "Aggregation completed in %v\n", duration)

	// Apply deduplication if baseline provided
	var output interface{} = result
	if baseline != nil {
		dedupStart := time.Now()
		deduped := aggregate.Dedup(result, baseline)
		dedupDuration := time.Since(dedupStart)
		fmt.Fprintf(os.Stderr, "Dedup completed in %v\n", dedupDuration)
		fmt.Fprintf(os.Stderr, "Removed: %d processes, %d files, %d commands, %d syscalls\n",
			deduped.RemovedProcesses,
			deduped.RemovedFiles,
			deduped.RemovedCommands,
			deduped.RemovedSyscalls)
		output = deduped
	}

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, jsonBytes, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Output written to: %s\n", outputFile)
	} else {
		fmt.Println(string(jsonBytes))
	}
}

func processDirectory(dirPath string, baseline *aggregate.PerProcessStats) error {
	if baseline == nil {
		return fmt.Errorf("-dedup-source is required for batch directory processing")
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	processed := 0
	errors := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packageName := entry.Name()
		packageDir := filepath.Join(dirPath, packageName)
		behaviorFile := filepath.Join(packageDir, "behavior.jsonl")
		diffFile := filepath.Join(packageDir, "diff.json")

		// Check if behavior.jsonl exists
		if _, err := os.Stat(behaviorFile); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Skipping %s: no behavior.jsonl found\n", packageName)
			continue
		}

		fmt.Fprintf(os.Stderr, "\n=== Processing %s ===\n", packageName)
		startTime := time.Now()

		// Process the file
		aggregator := aggregate.NewProcessAggregator()
		result, err := aggregator.ProcessFile(behaviorFile, packageName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", packageName, err)
			errors++
			continue
		}

		// Apply deduplication
		deduped := aggregate.Dedup(result, baseline)

		// Marshal to JSON
		jsonBytes, err := json.MarshalIndent(deduped, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON for %s: %v\n", packageName, err)
			errors++
			continue
		}

		// Write diff.json
		if err := os.WriteFile(diffFile, jsonBytes, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing diff.json for %s: %v\n", packageName, err)
			errors++
			continue
		}

		duration := time.Since(startTime)
		fmt.Fprintf(os.Stderr, "Created %s in %v (removed %d processes, %d files, %d commands, %d syscalls)\n",
			diffFile,
			duration,
			deduped.RemovedProcesses,
			deduped.RemovedFiles,
			deduped.RemovedCommands,
			deduped.RemovedSyscalls)
		processed++
	}

	fmt.Fprintf(os.Stderr, "\n=== Summary ===\n")
	fmt.Fprintf(os.Stderr, "Processed: %d packages\n", processed)
	fmt.Fprintf(os.Stderr, "Errors: %d\n", errors)

	return nil
}

func printUsage() {
	fmt.Println("Usage: aggregate [options]")
	fmt.Println()
	fmt.Println("Aggregate Tracee behavior.jsonl files with per-process analysis")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -input string         Path to behavior.jsonl file (either)")
	fmt.Println("  -input string         Directory to scan for behavior files (either)")
	fmt.Println("  -collection string    Collection name (default: \"default\")")
	fmt.Println("  -output string        Output JSON file (optional, defaults to stdout)")
	fmt.Println("  -dedup-source string  Path to safe baseline JSON for deduplication (optional)")
	fmt.Println("  -help                 Show this help message")
}
