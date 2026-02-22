package server

import (
	"encoding/json"
	"fmt"

	"github.com/acheong08/hackeurope-spr/internal/aggregate"
	"github.com/acheong08/hackeurope-spr/internal/analysis"
	"github.com/acheong08/hackeurope-spr/pkg/models"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Client -> Server
	TypeAnalyze MessageType = "analyze" // Client sends package.json to analyze
	TypePing    MessageType = "ping"    // Keep-alive

	// Server -> Client
	TypeDAG                   MessageType = "dag"                     // Dependency graph data
	TypeProgress              MessageType = "progress"                // Progress updates
	TypeLog                   MessageType = "log"                     // Log messages for terminal
	TypePackageStatus         MessageType = "package_status"          // Individual package status update
	TypePackageBehavioralData MessageType = "package_behavioral_data" // Per-package deduped diff data
	TypePackageAnalysis       MessageType = "package_analysis"        // Per-package AI security assessment
	TypeComplete              MessageType = "complete"                // Analysis complete
	TypeError                 MessageType = "error"                   // Error message
)

// Message is the base WebSocket message structure
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// AnalyzePayload sent by client to start analysis
type AnalyzePayload struct {
	PackageJSON string `json:"package_json"` // Raw package.json content
}

// DAGPayload contains the dependency graph for visualization
type DAGPayload struct {
	RootPackage *models.Package       `json:"root_package"`
	Nodes       []*models.PackageNode `json:"nodes"`
	EdgeCount   int                   `json:"edge_count"`
}

// ProgressPayload for progress bar updates
type ProgressPayload struct {
	Percent int    `json:"percent"` // 0-100
	Stage   string `json:"stage"`   // "dag", "upload", "workflow", "aggregate", "agent"
	Message string `json:"message"` // Human-readable status
}

// LogPayload for terminal output
type LogPayload struct {
	Message string `json:"message"`         // Log message
	Level   string `json:"level,omitempty"` // "info", "success", "warning", "error"
}

// PackageStatusPayload for individual package updates
type PackageStatusPayload struct {
	PackageID string `json:"package_id"` // "name@version"
	Name      string `json:"name"`
	Version   string `json:"version"`
	Status    string `json:"status"`   // "pending", "uploading", "analyzing", "complete", "failed"
	Progress  int    `json:"progress"` // 0-100 for this package
}

// CompletePayload sent when analysis is done
type CompletePayload struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ErrorPayload for error messages
type ErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// Helper functions to create messages

func NewDAGMessage(root *models.Package, nodes []*models.PackageNode, edgeCount int) Message {
	payload := DAGPayload{
		RootPackage: root,
		Nodes:       nodes,
		EdgeCount:   edgeCount,
	}
	payloadBytes, _ := json.Marshal(payload)
	return Message{Type: TypeDAG, Payload: payloadBytes}
}

func NewProgressMessage(percent int, stage, message string) Message {
	payload := ProgressPayload{
		Percent: percent,
		Stage:   stage,
		Message: message,
	}
	payloadBytes, _ := json.Marshal(payload)
	return Message{Type: TypeProgress, Payload: payloadBytes}
}

func NewLogMessage(message, level string) Message {
	payload := LogPayload{
		Message: message,
		Level:   level,
	}
	payloadBytes, _ := json.Marshal(payload)
	return Message{Type: TypeLog, Payload: payloadBytes}
}

func NewPackageStatusMessage(pkgID, name, version, status string, progress int) Message {
	payload := PackageStatusPayload{
		PackageID: pkgID,
		Name:      name,
		Version:   version,
		Status:    status,
		Progress:  progress,
	}
	payloadBytes, _ := json.Marshal(payload)
	return Message{Type: TypePackageStatus, Payload: payloadBytes}
}

func NewCompleteMessage(success bool, message string) Message {
	payload := CompletePayload{
		Success: success,
		Message: message,
	}
	payloadBytes, _ := json.Marshal(payload)
	return Message{Type: TypeComplete, Payload: payloadBytes}
}

func NewErrorMessage(message string, err error) Message {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}
	payload := ErrorPayload{Message: errMsg}
	payloadBytes, _ := json.Marshal(payload)
	return Message{Type: TypeError, Payload: payloadBytes}
}

// ParseAnalyzePayload extracts the analyze payload from a message
func ParseAnalyzePayload(msg Message) (*AnalyzePayload, error) {
	var payload AnalyzePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse analyze payload: %w", err)
	}
	return &payload, nil
}

// PackageBehavioralDataPayload contains the deduped behavioral diff for a package
type PackageBehavioralDataPayload struct {
	PackageID string                         `json:"package_id"`
	Name      string                         `json:"name"`
	Version   string                         `json:"version"`
	Data      *aggregate.DedupedProcessStats `json:"data"`
}

// PackageAnalysisPayload contains the AI security assessment for a package
type PackageAnalysisPayload struct {
	PackageID  string                       `json:"package_id"`
	Name       string                       `json:"name"`
	Version    string                       `json:"version"`
	Assessment *analysis.SecurityAssessment `json:"assessment"`
}

func NewPackageBehavioralDataMessage(pkgID, name, version string, data *aggregate.DedupedProcessStats) Message {
	payload := PackageBehavioralDataPayload{
		PackageID: pkgID,
		Name:      name,
		Version:   version,
		Data:      data,
	}
	payloadBytes, _ := json.Marshal(payload)
	return Message{Type: TypePackageBehavioralData, Payload: payloadBytes}
}

func NewPackageAnalysisMessage(pkgID, name, version string, assessment *analysis.SecurityAssessment) Message {
	payload := PackageAnalysisPayload{
		PackageID:  pkgID,
		Name:       name,
		Version:    version,
		Assessment: assessment,
	}
	payloadBytes, _ := json.Marshal(payload)
	return Message{Type: TypePackageAnalysis, Payload: payloadBytes}
}
