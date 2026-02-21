package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hackeurope/spr/pkg/models"
)

// PackageLockV3 represents package-lock.json version 3 structure
type PackageLockV3 struct {
	LockfileVersion int                           `json:"lockfileVersion"`
	Packages        map[string]PackageLockPackage `json:"packages"`
}

// PackageLockPackage represents a single package entry in lockfile
type PackageLockPackage struct {
	Version         string            `json:"version"`
	Resolved        string            `json:"resolved"`
	Integrity       string            `json:"integrity"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Dev             bool              `json:"dev"`
}

// LockfileManager handles generation and parsing of lockfiles
type LockfileManager struct {
	TempDir string
}

// NewLockfileManager creates a new lockfile manager
func NewLockfileManager() *LockfileManager {
	return &LockfileManager{}
}

// GenerateLockfile creates a package-lock.json from package.json in a temp directory
// Returns the path to the generated lockfile
func (lm *LockfileManager) GenerateLockfile(packageJSONPath string) (string, error) {
	// Check if npm is available
	if _, err := exec.LookPath("npm"); err != nil {
		return "", fmt.Errorf("npm not found in PATH: %w", err)
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "spr-lockfile-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	lm.TempDir = tempDir

	// Copy package.json to temp directory
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to read package.json: %w", err)
	}

	destPath := filepath.Join(tempDir, "package.json")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write package.json to temp: %w", err)
	}

	// Run npm install --package-lock-only
	cmd := exec.Command("npm", "install", "--package-lock-only", "--silent")
	cmd.Dir = tempDir
	// Discard npm output
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("npm install --package-lock-only failed: %w", err)
	}

	lockfilePath := filepath.Join(tempDir, "package-lock.json")
	if _, err := os.Stat(lockfilePath); os.IsNotExist(err) {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("package-lock.json was not generated")
	}

	return lockfilePath, nil
}

// ExtractRootPackage extracts the root package info from a lockfile
func (lm *LockfileManager) ExtractRootPackage(lockfilePath string) (*models.Package, error) {
	data, err := os.ReadFile(lockfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	var lockfile PackageLockV3
	if err := json.Unmarshal(data, &lockfile); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	if lockfile.LockfileVersion != 3 {
		return nil, fmt.Errorf("unsupported lockfile version: %d (expected 3)", lockfile.LockfileVersion)
	}

	// Root package is at path ""
	if rootPkg, exists := lockfile.Packages[""]; exists {
		return &models.Package{
			ID:      "root@" + rootPkg.Version,
			Name:    "root", // Package name not available in lockfile
			Version: rootPkg.Version,
		}, nil
	}

	return nil, fmt.Errorf("root package not found in lockfile")
}

// ParseLockfile parses a package-lock.json file into a DependencyGraph
func (lm *LockfileManager) ParseLockfile(lockfilePath string, rootPackage *models.Package) (*models.DependencyGraph, error) {
	data, err := os.ReadFile(lockfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	var lockfile PackageLockV3
	if err := json.Unmarshal(data, &lockfile); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	if lockfile.LockfileVersion != 3 {
		return nil, fmt.Errorf("unsupported lockfile version: %d (expected 3)", lockfile.LockfileVersion)
	}

	graph := models.NewDependencyGraph()
	graph.RootPackage = rootPackage

	// First pass: collect all packages
	for path, pkg := range lockfile.Packages {
		// Skip the root package (path is "")
		if path == "" {
			continue
		}

		// Extract name from path (node_modules/foo or node_modules/@scope/name)
		name := extractPackageName(path)
		if name == "" {
			continue
		}

		node := &models.PackageNode{
			Package: models.Package{
				ID:      name + "@" + pkg.Version,
				Name:    name,
				Version: pkg.Version,
			},
			ResolvedURL:  pkg.Resolved,
			Integrity:    pkg.Integrity,
			Dependencies: pkg.Dependencies,
		}

		graph.AddNode(node)
	}

	// Second pass: extract root dependencies
	if rootPkg, exists := lockfile.Packages[""]; exists {
		// Combine devDependencies and dependencies from root
		allRootDeps := make(map[string]string)
		for name, version := range rootPkg.Dependencies {
			allRootDeps[name] = version
		}
		for name, version := range rootPkg.DevDependencies {
			allRootDeps[name] = version
		}

		// Add root node with its dependencies
		rootNode := &models.PackageNode{
			Package:      *rootPackage,
			Dependencies: allRootDeps,
		}
		graph.AddNode(rootNode)
	}

	return graph, nil
}

// Cleanup removes the temporary directory
func (lm *LockfileManager) Cleanup() error {
	if lm.TempDir != "" {
		return os.RemoveAll(lm.TempDir)
	}
	return nil
}

// extractPackageName extracts the package name from a node_modules path
func extractPackageName(path string) string {
	// Handle scoped packages: node_modules/@scope/name
	parts := strings.Split(path, "node_modules/")
	if len(parts) < 2 {
		return ""
	}

	// Get the last part after node_modules/
	name := parts[len(parts)-1]

	// Remove any trailing node_modules references
	if idx := strings.Index(name, "/node_modules/"); idx != -1 {
		name = name[:idx]
	}

	return name
}

// BuildGraphFromPackageJSON is a convenience function that generates lockfile and builds graph
func BuildGraphFromPackageJSON(packageJSONPath string) (*models.DependencyGraph, error) {
	// Parse package.json first
	pkgJSON, err := ParsePackageJSON(packageJSONPath)
	if err != nil {
		return nil, err
	}

	rootPackage := pkgJSON.ToPackage()

	// Check if lockfile exists alongside package.json
	dir := filepath.Dir(packageJSONPath)
	existingLockfile := filepath.Join(dir, "package-lock.json")

	lm := NewLockfileManager()
	defer lm.Cleanup()

	var lockfilePath string
	if _, err := os.Stat(existingLockfile); err == nil {
		// Use existing lockfile
		lockfilePath = existingLockfile
	} else {
		// Generate new lockfile
		lockfilePath, err = lm.GenerateLockfile(packageJSONPath)
		if err != nil {
			return nil, fmt.Errorf("failed to generate lockfile: %w", err)
		}
	}

	// Parse lockfile into graph
	graph, err := lm.ParseLockfile(lockfilePath, rootPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	return graph, nil
}
