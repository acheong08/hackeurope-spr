package tester

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// PackageJSON represents the structure of a package.json file
type PackageJSON struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Private      bool              `json:"private"`
	Type         string            `json:"type,omitempty"`
	Main         string            `json:"main,omitempty"`
	Bin          map[string]string `json:"bin,omitempty"`
	Scripts      map[string]string `json:"scripts,omitempty"`
	Dependencies map[string]string `json:"dependencies"`
}

// TestPackage represents a generated test package
type TestPackage struct {
	Name            string
	Version         string
	PackageName     string
	PackageVersion  string
	ModuleType      string
	ImportStatement string
	CLIBinary       string
	HasCLI          bool
	OutputDir       string
}

// Generator creates test packages for behavioral analysis
type Generator struct {
	templatesDir  string
	detector      *Detector
	registryURL   string
	registryOwner string
	registryToken string
}

// NewGenerator creates a new test package generator
func NewGenerator(templatesDir string) *Generator {
	return &Generator{
		templatesDir: templatesDir,
		detector:     NewDetector(),
	}
}

// NewGeneratorWithRegistry creates a new test package generator with custom registry
func NewGeneratorWithRegistry(templatesDir, registryURL, registryOwner, registryToken string) *Generator {
	return &Generator{
		templatesDir:  templatesDir,
		detector:      NewDetectorWithRegistry(registryURL, registryOwner, registryToken),
		registryURL:   registryURL,
		registryOwner: registryOwner,
		registryToken: registryToken,
	}
}

// GenerateAll creates all test variants for a package
func (g *Generator) GenerateAll(name, version, outputDir string) ([]string, error) {
	// Detect package info
	info, err := g.detector.DetectPackage(name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package: %w", err)
	}

	// Create normalized directory name
	normalizedName := NormalizePackageName(name)
	pkgDir := filepath.Join(outputDir, fmt.Sprintf("%s@%s", normalizedName, version))

	// Generate each test type
	var generatedDirs []string

	// 1. Install test (always generated)
	installDir := filepath.Join(pkgDir, "install")
	if err := g.generateInstallTest(info, installDir); err != nil {
		return nil, fmt.Errorf("failed to generate install test: %w", err)
	}
	generatedDirs = append(generatedDirs, installDir)

	// 2. Import test (always generated)
	importDir := filepath.Join(pkgDir, "import")
	if err := g.generateImportTest(info, importDir); err != nil {
		return nil, fmt.Errorf("failed to generate import test: %w", err)
	}
	generatedDirs = append(generatedDirs, importDir)

	// 3. Prototype pollution test (always generated)
	protoDir := filepath.Join(pkgDir, "prototype")
	if err := g.generatePrototypeTest(info, protoDir); err != nil {
		return nil, fmt.Errorf("failed to generate prototype test: %w", err)
	}
	generatedDirs = append(generatedDirs, protoDir)

	// 4. CLI test (only if package has bin entries)
	if info.HasBin {
		cliDir := filepath.Join(pkgDir, "cli")
		if err := g.generateCLITest(info, cliDir); err != nil {
			return nil, fmt.Errorf("failed to generate CLI test: %w", err)
		}
		generatedDirs = append(generatedDirs, cliDir)
	}

	return generatedDirs, nil
}

// generateInstallTest creates the install-time test package
func (g *Generator) generateInstallTest(info *PackageInfo, outputDir string) error {
	data := TestPackage{
		Name:           fmt.Sprintf("test-install-%s", NormalizePackageName(info.Name)),
		Version:        "1.0.0",
		PackageName:    info.Name,
		PackageVersion: info.Version,
		ModuleType:     g.detector.GetPackageJSONType(info),
		OutputDir:      outputDir,
	}

	// Generate package.json using proper JSON encoding
	pkgJSON := PackageJSON{
		Name:         data.Name,
		Version:      data.Version,
		Description:  fmt.Sprintf("Install-time behavior test for %s@%s", info.Name, info.Version),
		Private:      true,
		Type:         data.ModuleType,
		Dependencies: map[string]string{info.Name: info.Version},
	}

	return g.generateTestPackage("install-test", data, outputDir, pkgJSON, nil)
}

// generateImportTest creates the import-time test package
func (g *Generator) generateImportTest(info *PackageInfo, outputDir string) error {
	data := TestPackage{
		Name:            fmt.Sprintf("test-import-%s", NormalizePackageName(info.Name)),
		Version:         "1.0.0",
		PackageName:     info.Name,
		PackageVersion:  info.Version,
		ModuleType:      g.detector.GetPackageJSONType(info),
		ImportStatement: g.detector.GetImportStatement(info),
		OutputDir:       outputDir,
	}

	// Generate package.json using proper JSON encoding
	pkgJSON := PackageJSON{
		Name:         data.Name,
		Version:      data.Version,
		Description:  fmt.Sprintf("Import-time behavior test for %s@%s", info.Name, info.Version),
		Private:      true,
		Type:         data.ModuleType,
		Dependencies: map[string]string{info.Name: info.Version},
	}

	return g.generateTestPackage("import-test", data, outputDir, pkgJSON, nil)
}

// generatePrototypeTest creates the prototype pollution test package
func (g *Generator) generatePrototypeTest(info *PackageInfo, outputDir string) error {
	data := TestPackage{
		Name:            fmt.Sprintf("test-prototype-%s", NormalizePackageName(info.Name)),
		Version:         "1.0.0",
		PackageName:     info.Name,
		PackageVersion:  info.Version,
		ModuleType:      g.detector.GetPackageJSONType(info),
		ImportStatement: g.detector.GetImportStatement(info),
		OutputDir:       outputDir,
	}

	// Generate package.json using proper JSON encoding
	pkgJSON := PackageJSON{
		Name:         data.Name,
		Version:      data.Version,
		Description:  fmt.Sprintf("Prototype pollution test for %s@%s", info.Name, info.Version),
		Private:      true,
		Type:         data.ModuleType,
		Dependencies: map[string]string{info.Name: info.Version},
	}

	return g.generateTestPackage("prototype-test", data, outputDir, pkgJSON, nil)
}

// generateCLITest creates a marker for CLI test (uses npx in workflow)
func (g *Generator) generateCLITest(info *PackageInfo, outputDir string) error {
	// Create directory as marker - actual test uses npx in workflow
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create CLI marker directory: %w", err)
	}

	// Get first binary entry for documentation
	var firstBinName string
	for name := range info.Bin {
		firstBinName = name
		break
	}

	// Create marker file with binary info
	markerContent := fmt.Sprintf("# CLI Test Marker\nPackage: %s@%s\nBinary: %s\n\nCLI test runs via: npx %s\n",
		info.Name, info.Version, firstBinName, info.Name)
	markerPath := filepath.Join(outputDir, "HAS_CLI")
	if err := os.WriteFile(markerPath, []byte(markerContent), 0644); err != nil {
		return fmt.Errorf("failed to write CLI marker: %w", err)
	}

	return nil
}

// generateTestPackage creates a test package with proper JSON encoding
type additionalFile struct {
	name    string
	content []byte
}

func (g *Generator) generateTestPackage(templateName string, data TestPackage, outputDir string, pkgJSON PackageJSON, extraFiles []additionalFile) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write package.json with proper JSON encoding
	pkgJSONPath := filepath.Join(outputDir, "package.json")
	pkgJSONData, err := json.MarshalIndent(pkgJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal package.json: %w", err)
	}
	if err := os.WriteFile(pkgJSONPath, pkgJSONData, 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}

	// Process template directory
	templateDir := filepath.Join(g.templatesDir, templateName)

	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return fmt.Errorf("failed to read template directory: %w", err)
	}

	for _, entry := range entries {
		// Skip package.json - we already generated it with proper JSON
		if entry.Name() == "package.json" {
			continue
		}

		srcPath := filepath.Join(templateDir, entry.Name())
		dstPath := filepath.Join(outputDir, entry.Name())

		if entry.IsDir() {
			// Recursively copy directories
			if err := g.copyDir(srcPath, dstPath, data); err != nil {
				return fmt.Errorf("failed to copy directory %s: %w", entry.Name(), err)
			}
		} else {
			// Process file as template
			if err := g.processTemplateFile(srcPath, dstPath, data, templateName); err != nil {
				return fmt.Errorf("failed to process template %s: %w", entry.Name(), err)
			}
		}
	}

	// Write any additional files
	for _, file := range extraFiles {
		filePath := filepath.Join(outputDir, file.name)
		if err := os.WriteFile(filePath, file.content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.name, err)
		}
	}

	return nil
}

// processTemplateFile processes a single template file
func (g *Generator) processTemplateFile(srcPath, dstPath string, data TestPackage, templateContext string) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("template %s: failed to read: %w", srcPath, err)
	}

	// Parse and execute template
	tmpl, err := template.New(filepath.Base(srcPath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("template %s: failed to parse: %w", srcPath, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template %s: failed to execute: %w", srcPath, err)
	}

	// Write output
	if err := os.WriteFile(dstPath, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("template %s: failed to write: %w", dstPath, err)
	}

	return nil
}

// copyDir recursively copies a directory, processing templates
func (g *Generator) copyDir(srcPath, dstPath string, data TestPackage) error {
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		srcChild := filepath.Join(srcPath, entry.Name())
		dstChild := filepath.Join(dstPath, entry.Name())

		if entry.IsDir() {
			if err := g.copyDir(srcChild, dstChild, data); err != nil {
				return err
			}
		} else {
			if err := g.processTemplateFile(srcChild, dstChild, data, ""); err != nil {
				return err
			}
		}
	}

	return nil
}

// ListGenerated returns all generated test packages in a directory
func (g *Generator) ListGenerated(baseDir string) ([]TestPackage, error) {
	var packages []TestPackage

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse directory name (format: name@version)
		dirName := entry.Name()
		parts := strings.Split(dirName, "@")
		if len(parts) != 2 {
			continue
		}

		pkg := TestPackage{
			PackageName:    parts[0],
			PackageVersion: parts[1],
		}

		// Check which tests exist
		pkgDir := filepath.Join(baseDir, dirName)
		if _, err := os.Stat(filepath.Join(pkgDir, "install")); err == nil {
			// Install test exists
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}
