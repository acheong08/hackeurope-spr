package aggregate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// ProcessAggregator aggregates statistics per process
type ProcessAggregator struct {
	processes map[string]*processData
}

type processData struct {
	syscallProfile   map[string]int
	fileAccess       map[string]int
	executedCommands map[string]int
	ips              map[string]int
	dnsRecords       map[string]int
}

// NewProcessAggregator creates a new ProcessAggregator
func NewProcessAggregator() *ProcessAggregator {
	return &ProcessAggregator{
		processes: make(map[string]*processData),
	}
}

// ProcessFile reads a JSONL file and aggregates per-process statistics
func (pa *ProcessAggregator) ProcessFile(filename string, collection string) (*PerProcessStats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return pa.ProcessReader(file, collection)
}

// ProcessReader reads from an io.Reader and aggregates per-process statistics
func (pa *ProcessAggregator) ProcessReader(reader io.Reader, collection string) (*PerProcessStats, error) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event TraceeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		pa.processEvent(&event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	return pa.buildStats(collection), nil
}

func (pa *ProcessAggregator) processEvent(event *TraceeEvent) {
	procName := event.ProcessName
	if procName == "" {
		procName = fmt.Sprintf("pid_%d", event.ProcessID)
	}

	data, exists := pa.processes[procName]
	if !exists {
		data = &processData{
			syscallProfile:   make(map[string]int),
			fileAccess:       make(map[string]int),
			executedCommands: make(map[string]int),
			ips:              make(map[string]int),
			dnsRecords:       make(map[string]int),
		}
		pa.processes[procName] = data
	}

	data.syscallProfile[event.EventName]++

	switch event.EventName {
	case "openat":
		pa.processOpenat(data, event)
	case "execve":
		pa.processExecve(data, event)
	case "connect":
		pa.processConnect(data, event)
	case "net_packet_dns_request":
		pa.processDNS(data, event)
	}
}

func (pa *ProcessAggregator) processOpenat(data *processData, event *TraceeEvent) {
	for _, arg := range event.Args {
		if arg.Name == "pathname" {
			var pathname string
			if err := json.Unmarshal(arg.Value, &pathname); err == nil {
				// Filter out node_modules paths
				if !strings.Contains(pathname, "node_modules") {
					data.fileAccess[pathname]++
				}
			}
			break
		}
	}
}

func (pa *ProcessAggregator) processExecve(data *processData, event *TraceeEvent) {
	for _, arg := range event.Args {
		if arg.Name == "pathname" {
			var pathname string
			if err := json.Unmarshal(arg.Value, &pathname); err == nil {
				data.executedCommands[pathname]++
			}
			break
		}
	}
}

func (pa *ProcessAggregator) processConnect(data *processData, event *TraceeEvent) {
	for _, arg := range event.Args {
		if arg.Name == "addr" {
			var sockAddr struct {
				Family  string `json:"sa_family"`
				SinAddr string `json:"sin_addr"`
				SinPort string `json:"sin_port"`
				SunPath string `json:"sun_path"`
			}
			if err := json.Unmarshal(arg.Value, &sockAddr); err == nil {
				// Skip local Unix sockets (AF_UNIX) - these are IPC, not network
				if sockAddr.Family == "AF_UNIX" {
					return
				}
				// Handle IPv4/IPv6 (AF_INET)
				if sockAddr.SinAddr != "" {
					key := sockAddr.SinAddr
					if sockAddr.SinPort != "" && sockAddr.SinPort != "0" {
						key = fmt.Sprintf("%s:%s", sockAddr.SinAddr, sockAddr.SinPort)
					}
					data.ips[key]++
				}
			}
			break
		}
	}
}

func (pa *ProcessAggregator) processDNS(data *processData, event *TraceeEvent) {
	for _, arg := range event.Args {
		if arg.Name == "dns_questions" {
			var questions []struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(arg.Value, &questions); err == nil {
				for _, q := range questions {
					data.dnsRecords[q.Query]++
				}
			}
			break
		}
	}
}

func (pa *ProcessAggregator) buildStats(collection string) *PerProcessStats {
	perProcess := make(map[string]*ProcessSummary)

	for procName, data := range pa.processes {
		perProcess[procName] = &ProcessSummary{
			SyscallProfile:   data.syscallProfile,
			FileAccess:       data.fileAccess,
			ExecutedCommands: data.executedCommands,
			NetworkActivity: NetworkActivity{
				IPs:        data.ips,
				DNSRecords: data.dnsRecords,
			},
		}
	}

	return &PerProcessStats{
		Collection:     collection,
		PerProcess:     perProcess,
		CountProcesses: len(perProcess),
	}
}
