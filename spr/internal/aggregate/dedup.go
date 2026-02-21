package aggregate

import (
	"encoding/json"
	"fmt"
	"os"
)

// DedupedProcessStats represents the result after deduplication
type DedupedProcessStats struct {
	Collection       string                     `json:"collection"`
	PerProcess       map[string]*ProcessSummary `json:"per_process"`
	CountProcesses   int                        `json:"count_processes"`
	BaselineSource   string                     `json:"baseline_source"`
	RemovedProcesses int                        `json:"removed_processes"`
	RemovedFiles     int                        `json:"removed_files"`
	RemovedCommands  int                        `json:"removed_commands"`
	RemovedSyscalls  int                        `json:"removed_syscalls"`
}

// LoadPerProcessStats loads per-process stats from a JSON file
func LoadPerProcessStats(filename string) (*PerProcessStats, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var stats PerProcessStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &stats, nil
}

// Dedup subtracts baseline data from target data
func Dedup(target *PerProcessStats, baseline *PerProcessStats) *DedupedProcessStats {
	result := &DedupedProcessStats{
		Collection:     target.Collection,
		BaselineSource: baseline.Collection,
		PerProcess:     make(map[string]*ProcessSummary),
	}

	removedProcesses := 0
	removedFiles := 0
	removedCommands := 0
	removedSyscalls := 0

	for procName, targetProc := range target.PerProcess {
		// Check if this process exists in baseline
		baselineProc, exists := baseline.PerProcess[procName]
		if !exists {
			// Process doesn't exist in baseline, keep it entirely
			result.PerProcess[procName] = targetProc
			continue
		}

		// Process exists, need to dedup
		dedupedProc := &ProcessSummary{
			SyscallProfile:   make(map[string]int),
			FileAccess:       make(map[string]int),
			ExecutedCommands: make(map[string]int),
			NetworkActivity: NetworkActivity{
				IPs:        make(map[string]int),
				DNSRecords: make(map[string]int),
			},
		}

		// Dedup syscalls (only include if count differs significantly)
		for syscall, count := range targetProc.SyscallProfile {
			if baselineCount, exists := baselineProc.SyscallProfile[syscall]; !exists || count > baselineCount {
				// Keep the difference if count is higher
				if exists && count > baselineCount {
					dedupedProc.SyscallProfile[syscall] = count - baselineCount
				} else {
					dedupedProc.SyscallProfile[syscall] = count
				}
			} else {
				removedSyscalls++
			}
		}

		// Dedup file access
		for file, count := range targetProc.FileAccess {
			if _, exists := baselineProc.FileAccess[file]; !exists {
				dedupedProc.FileAccess[file] = count
			} else {
				removedFiles++
			}
		}

		// Dedup executed commands
		for cmd, count := range targetProc.ExecutedCommands {
			if _, exists := baselineProc.ExecutedCommands[cmd]; !exists {
				dedupedProc.ExecutedCommands[cmd] = count
			} else {
				removedCommands++
			}
		}

		// Dedup network activity (IPs)
		for ip, count := range targetProc.NetworkActivity.IPs {
			if _, exists := baselineProc.NetworkActivity.IPs[ip]; !exists {
				dedupedProc.NetworkActivity.IPs[ip] = count
			}
		}

		// Dedup DNS records
		for dns, count := range targetProc.NetworkActivity.DNSRecords {
			if _, exists := baselineProc.NetworkActivity.DNSRecords[dns]; !exists {
				dedupedProc.NetworkActivity.DNSRecords[dns] = count
			}
		}

		// Only keep process if it has unique activity
		if len(dedupedProc.SyscallProfile) > 0 ||
			len(dedupedProc.FileAccess) > 0 ||
			len(dedupedProc.ExecutedCommands) > 0 ||
			len(dedupedProc.NetworkActivity.IPs) > 0 ||
			len(dedupedProc.NetworkActivity.DNSRecords) > 0 {
			result.PerProcess[procName] = dedupedProc
		} else {
			removedProcesses++
		}
	}

	result.CountProcesses = len(result.PerProcess)
	result.RemovedProcesses = removedProcesses
	result.RemovedFiles = removedFiles
	result.RemovedCommands = removedCommands
	result.RemovedSyscalls = removedSyscalls

	return result
}
