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
		inputFile  = flag.String("input", "", "Path to behavior.jsonl file (required)")
		collection = flag.String("collection", "default", "Collection name")
		outputFile = flag.String("output", "", "Output JSON file (optional, defaults to stdout)")
		perProcess = flag.Bool("per-process", false, "Generate per-process statistics")
		help       = flag.Bool("help", false, "Show help")
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

	startTime := time.Now()
	fmt.Fprintf(os.Stderr, "Processing %s...\n", *inputFile)

	var result interface{}
	var err error

	if *perProcess {
		aggregator := aggregate.NewProcessAggregator()
		result, err = aggregator.ProcessFile(*inputFile, *collection)
	} else {
		aggregator := aggregate.NewAggregator()
		result, err = aggregator.ProcessFile(*inputFile, *collection)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	fmt.Fprintf(os.Stderr, "Completed in %v\n", duration)

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
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
	fmt.Println("Aggregate Tracee behavior.jsonl files")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -input string       Path to behavior.jsonl file (required)")
	fmt.Println("  -collection string  Collection name (default: \"default\")")
	fmt.Println("  -output string      Output JSON file (optional, defaults to stdout)")
	fmt.Println("  -per-process        Generate per-process statistics")
	fmt.Println("  -help               Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  aggregate -input behavior.jsonl")
	fmt.Println("  aggregate -input behavior.jsonl -collection module2 -output stats.json")
	fmt.Println("  aggregate -input behavior.jsonl -per-process -output per_process.json")
}
