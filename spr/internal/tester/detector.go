package tester

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// PackageType represents the module system type
type PackageType string

const (
	TypeCommonJS PackageType = "commonjs"
	TypeESM      PackageType = "module"
	TypeDual     PackageType = "dual" // Supports both
	TypeUnknown  PackageType = "unknown"
)

// PackageVersionInfo represents a single version in npm registry
type PackageVersionInfo struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Type    string            `json:"type,omitempty"`
	Main    string            `json:"main,omitempty"`
	Module  string            `json:"module,omitempty"`
	Exports interface{}       `json:"exports,omitempty"`
	Bin     json.RawMessage   `json:"bin,omitempty"` // Can be string or object
	Scripts map[string]string `json:"scripts,omitempty"`
}

// PackageInfo holds metadata about an npm package
type PackageInfo struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Type       PackageType       `json:"type"`
	Main       string            `json:"main"`
	Module     string            `json:"module"`
	Exports    interface{}       `json:"exports"`
	Bin        map[string]string `json:"bin"`
	HasBin     bool              `json:"has_bin"`
	HasPrepare bool              `json:"has_prepare"`
	HasInstall bool              `json:"has_install"`
	Scripts    map[string]string `json:"scripts"`
}

// RegistryPackage represents npm registry metadata
type RegistryPackage struct {
	Name     string                        `json:"name"`
	Versions map[string]PackageVersionInfo `json:"versions"`
}

// Detector analyzes npm packages to determine their type
// and available attack surfaces
type Detector struct {
	HTTPClient *http.Client
}

// NewDetector creates a new package detector
func NewDetector() *Detector {
	return &Detector{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DetectPackage fetches and analyzes package metadata from npm registry
func (d *Detector) DetectPackage(name, version string) (*PackageInfo, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", name)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var registryPkg RegistryPackage
	if err := json.NewDecoder(resp.Body).Decode(&registryPkg); err != nil {
		return nil, fmt.Errorf("failed to decode registry response: %w", err)
	}

	versionInfo, exists := registryPkg.Versions[version]
	if !exists {
		return nil, fmt.Errorf("version %s not found for package %s", version, name)
	}

	info := &PackageInfo{
		Name:    versionInfo.Name,
		Version: versionInfo.Version,
		Main:    versionInfo.Main,
		Module:  versionInfo.Module,
		Exports: versionInfo.Exports,
		Scripts: versionInfo.Scripts,
	}

	// Detect module type
	info.Type = d.detectModuleType(&versionInfo)

	// Parse bin field (can be string or object)
	info.Bin, info.HasBin = d.parseBin(versionInfo.Bin)

	// Detect install scripts
	info.HasPrepare = d.hasScript(versionInfo.Scripts, "prepare")
	info.HasInstall = d.hasScript(versionInfo.Scripts, "preinstall") ||
		d.hasScript(versionInfo.Scripts, "postinstall") ||
		d.hasScript(versionInfo.Scripts, "install")

	return info, nil
}

// detectModuleType determines the module system type
func (d *Detector) detectModuleType(v *PackageVersionInfo) PackageType {
	// Check explicit type field
	if v.Type == "module" {
		return TypeESM
	}

	// Check for ESM indicators
	if v.Module != "" {
		return TypeESM
	}

	// Check exports field for dual support
	if v.Exports != nil {
		// If exports exists, it might support both
		return TypeDual
	}

	// Default to CommonJS
	return TypeCommonJS
}

// parseBin handles the bin field which can be string or object
func (d *Detector) parseBin(bin json.RawMessage) (map[string]string, bool) {
	if len(bin) == 0 {
		return nil, false
	}

	// Try parsing as string first
	var binStr string
	if err := json.Unmarshal(bin, &binStr); err == nil {
		// Single binary with package name
		return map[string]string{"default": binStr}, true
	}

	// Try parsing as object
	var binMap map[string]string
	if err := json.Unmarshal(bin, &binMap); err == nil {
		return binMap, len(binMap) > 0
	}

	return nil, false
}

// hasScript checks if a script exists
func (d *Detector) hasScript(scripts map[string]string, name string) bool {
	if scripts == nil {
		return false
	}
	_, exists := scripts[name]
	return exists
}

// GetImportStatement returns the appropriate import statement for the package type
func (d *Detector) GetImportStatement(info *PackageInfo) string {
	switch info.Type {
	case TypeESM:
		return fmt.Sprintf("import * as target from '%s';", info.Name)
	case TypeCommonJS, TypeDual, TypeUnknown:
		return fmt.Sprintf("const target = require('%s');", info.Name)
	default:
		return fmt.Sprintf("const target = require('%s');", info.Name)
	}
}

// GetPackageJSONType returns the type field value for package.json
func (d *Detector) GetPackageJSONType(info *PackageInfo) string {
	switch info.Type {
	case TypeESM:
		return "module"
	default:
		return "commonjs"
	}
}

// NormalizePackageName normalizes package name for file paths
func NormalizePackageName(name string) string {
	// Replace @scope/name with scope__name
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			return parts[0][1:] + "__" + parts[1]
		}
	}
	return name
}
