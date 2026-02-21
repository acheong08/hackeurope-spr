package aggregate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Aggregator processes Tracee events and generates statistics
type Aggregator struct {
	totalEvents      int
	syscallProfile   map[string]int
	fileAccess       map[string]int
	executedCommands map[string]int
	ips              map[string]int
	dnsRecords       map[string]int
}

// NewAggregator creates a new Aggregator instance
func NewAggregator() *Aggregator {
	return &Aggregator{
		syscallProfile:   make(map[string]int),
		fileAccess:       make(map[string]int),
		executedCommands: make(map[string]int),
		ips:              make(map[string]int),
		dnsRecords:       make(map[string]int),
	}
}

// ProcessFile reads a JSONL file and aggregates statistics
func (a *Aggregator) ProcessFile(filename string, collection string) (*Stats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return a.ProcessReader(file, collection)
}

// ProcessReader reads from an io.Reader and aggregates statistics
func (a *Aggregator) ProcessReader(reader io.Reader, collection string) (*Stats, error) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event TraceeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip invalid JSON lines (matching Python behavior)
			continue
		}

		a.processEvent(&event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	return a.buildStats(collection), nil
}

func (a *Aggregator) processEvent(event *TraceeEvent) {
	a.totalEvents++
	a.syscallProfile[event.EventName]++

	switch event.EventName {
	case "openat":
		a.processOpenat(event)
	case "execve":
		a.processExecve(event)
	case "connect":
		a.processConnect(event)
	case "net_packet_dns_request":
		a.processDNS(event)
	}
}

func (a *Aggregator) processOpenat(event *TraceeEvent) {
	for _, arg := range event.Args {
		if arg.Name == "pathname" {
			var pathname string
			if err := json.Unmarshal(arg.Value, &pathname); err == nil {
				// Filter out node_modules paths
				if !strings.Contains(pathname, "node_modules") {
					a.fileAccess[pathname]++
				}
			}
			break
		}
	}
}

func (a *Aggregator) processExecve(event *TraceeEvent) {
	for _, arg := range event.Args {
		if arg.Name == "pathname" {
			var pathname string
			if err := json.Unmarshal(arg.Value, &pathname); err == nil {
				a.executedCommands[pathname]++
			}
			break
		}
	}
}

func (a *Aggregator) processConnect(event *TraceeEvent) {
	for _, arg := range event.Args {
		if arg.Name == "addr" {
			var addr struct {
				IP   string `json:"ip"`
				Port int    `json:"port"`
			}
			if err := json.Unmarshal(arg.Value, &addr); err == nil {
				key := addr.IP
				if addr.Port != 0 {
					key = fmt.Sprintf("%s:%d", addr.IP, addr.Port)
				}
				a.ips[key]++
			} else {
				// Handle case where addr is a string
				var addrStr string
				if err := json.Unmarshal(arg.Value, &addrStr); err == nil {
					a.ips[addrStr]++
				}
			}
			break
		}
	}
}

func (a *Aggregator) processDNS(event *TraceeEvent) {
	for _, arg := range event.Args {
		if arg.Name == "dns_questions" {
			var questions []struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(arg.Value, &questions); err == nil {
				for _, q := range questions {
					a.dnsRecords[q.Query]++
				}
			}
			break
		}
	}
}

func (a *Aggregator) buildStats(collection string) *Stats {
	stats := &Stats{
		Collection:       collection,
		TotalEvents:      a.totalEvents,
		SyscallProfile:   a.syscallProfile,
		FileAccess:       a.fileAccess,
		ExecutedCommands: a.executedCommands,
		NetworkActivity: NetworkActivity{
			IPs:        a.ips,
			DNSRecords: a.dnsRecords,
		},
		RiskFlags: a.detectRiskFlags(),
	}

	return stats
}

func (a *Aggregator) detectRiskFlags() []string {
	flags := make(map[string]bool)

	sensitivePaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/root",
		".ssh",
	}

	shellBinaries := []string{
		"/bin/sh",
		"/bin/bash",
		"sh",
		"bash",
	}

	// Check for sensitive file access
	sensitiveAccessFound := false
	for path := range a.fileAccess {
		for _, sensitive := range sensitivePaths {
			if strings.Contains(path, sensitive) {
				sensitiveAccessFound = true
				break
			}
		}
		if sensitiveAccessFound {
			break
		}
	}
	if sensitiveAccessFound {
		flags["sensitive_file_access"] = true
	}

	// Check for shell execution
	shellSpawned := false
	for cmd := range a.executedCommands {
		for _, shell := range shellBinaries {
			if strings.Contains(cmd, shell) {
				shellSpawned = true
				break
			}
		}
		if shellSpawned {
			break
		}
	}
	if shellSpawned {
		flags["shell_spawned"] = true
	}

	// Check for network activity
	if len(a.ips) > 0 {
		flags["network_activity"] = true
	}

	// Check for procfs access
	procfsAccess := false
	for path := range a.fileAccess {
		if strings.Contains(path, "/proc") {
			procfsAccess = true
			break
		}
	}
	if procfsAccess {
		flags["procfs_access"] = true
	}

	// Convert map to slice
	result := make([]string, 0, len(flags))
	for flag := range flags {
		result = append(result, flag)
	}
	return result
}
