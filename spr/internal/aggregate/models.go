package aggregate

import "encoding/json"

// TraceeEvent represents a single Tracee JSON event
type TraceeEvent struct {
	Timestamp       int64         `json:"timestamp"`
	ProcessID       int           `json:"processId"`
	ProcessName     string        `json:"processName"`
	ParentProcessID int           `json:"parentProcessId"`
	EventName       string        `json:"eventName"`
	Args            []TraceeArg   `json:"args"`
	Container       ContainerInfo `json:"container"`
}

// TraceeArg represents an argument in a Tracee event
type TraceeArg struct {
	Name  string          `json:"name"`
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

// ContainerInfo represents container metadata
type ContainerInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
}

// Stats represents the aggregated statistics
type Stats struct {
	Collection       string          `json:"collection"`
	TotalEvents      int             `json:"total_events"`
	SyscallProfile   map[string]int  `json:"syscall_profile"`
	FileAccess       map[string]int  `json:"file_access"`
	ExecutedCommands map[string]int  `json:"executed_commands"`
	NetworkActivity  NetworkActivity `json:"network_activity"`
	RiskFlags        []string        `json:"risk_flags"`
}

// NetworkActivity contains network-related aggregations
type NetworkActivity struct {
	IPs        map[string]int `json:"ips"`
	DNSRecords map[string]int `json:"dns_records"`
}

// PerProcessStats contains stats grouped by process
type PerProcessStats struct {
	Collection     string                     `json:"collection"`
	PerProcess     map[string]*ProcessSummary `json:"per_process"`
	CountProcesses int                        `json:"count_processes"`
}

// ProcessSummary contains summary for a single process
type ProcessSummary struct {
	SyscallProfile   map[string]int  `json:"syscall_profile"`
	FileAccess       map[string]int  `json:"file_access"`
	ExecutedCommands map[string]int  `json:"executed_commands"`
	NetworkActivity  NetworkActivity `json:"network_activity"`
}
