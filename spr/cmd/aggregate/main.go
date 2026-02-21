package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/acheong08/hackeurope-spr/internal/aggregate"
)

func main() {
	var (
		inputFile   = flag.String("input", "", "Path to behavior.jsonl file (required)")
		collection  = flag.String("collection", "default", "Collection name")
		outputFile  = flag.String("output", "", "Output JSON file (optional, defaults to stdout)")
		dedupSource = flag.String("dedup-source", "", "Path to safe baseline JSON file for deduplication")
		help        = flag.Bool("help", false, "Show help")
	)

	flag.Parse()

	if *help || *inputFile == "" {
		printUsage()
		os.Exit(0)
	}

	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file not found: %s\n", *inputFile)
		os.Exit(1)
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

	startTime := time.Now()
	fmt.Fprintf(os.Stderr, "Processing %s...\n", *inputFile)

	// Always use per-process aggregation
	aggregator := aggregate.NewProcessAggregator()
	result, err := aggregator.ProcessFile(*inputFile, *collection)
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
	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, jsonBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Output written to: %s\n", *outputFile)
	} else {
		fmt.Println(string(jsonBytes))
	}
}

func printUsage() {
	fmt.Println("Usage: aggregate [options]")
	fmt.Println()
	fmt.Println("Aggregate Tracee behavior.jsonl files with per-process analysis")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -input string         Path to behavior.jsonl file (required)")
	fmt.Println("  -collection string    Collection name (default: \"default\")")
	fmt.Println("  -output string        Output JSON file (optional, defaults to stdout)")
	fmt.Println("  -dedup-source string  Path to safe baseline JSON for deduplication (optional)")
	fmt.Println("  -help                 Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Basic per-process aggregation")
	fmt.Println("  aggregate -input behavior.jsonl -collection module2 -output stats.json")
	fmt.Println()
	fmt.Println("  # With deduplication against known safe baseline")
	fmt.Println("  aggregate -input behavior.jsonl -collection suspicious -dedup-source safe.json -output diff.json")
	fmt.Println()
	fmt.Println("Workflow:")
	fmt.Println("  1. Create safe baseline: aggregate -input safe-sample.jsonl -collection safe -output safe.json")
	fmt.Println("  2. Analyze target: aggregate -input target.jsonl -collection target -dedup-source safe.json -output diff.json")
	fmt.Println("  3. Send diff.json to LLM for security analysis")
}
