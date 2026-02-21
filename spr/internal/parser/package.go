package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acheong08/hackeurope-spr/pkg/models"
)

// PackageJSON represents the structure of package.json
type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// ParsePackageJSON reads and parses a package.json file
func ParsePackageJSON(path string) (*PackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	return &pkg, nil
}

// ToPackage converts PackageJSON to models.Package
func (p *PackageJSON) ToPackage() *models.Package {
	return &models.Package{
		ID:      p.Name + "@" + p.Version,
		Name:    p.Name,
		Version: p.Version,
	}
}

// GetAllDependencies returns production + dev dependencies
func (p *PackageJSON) GetAllDependencies() map[string]string {
	all := make(map[string]string)
	for k, v := range p.Dependencies {
		all[k] = v
	}
	for k, v := range p.DevDependencies {
		all[k] = v
	}
	return all
}

// ValidatePackageJSON checks if a package.json file exists and is valid
func ValidatePackageJSON(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("package.json not found at %s", path)
	}

	pkg, err := ParsePackageJSON(path)
	if err != nil {
		return err
	}

	if pkg.Name == "" {
		return fmt.Errorf("package.json missing 'name' field")
	}

	if pkg.Version == "" {
		return fmt.Errorf("package.json missing 'version' field")
	}

	return nil
}

// FindPackageJSON searches for package.json in the given directory
func FindPackageJSON(dir string) (string, error) {
	path := filepath.Join(dir, "package.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("package.json not found in %s", dir)
	}
	return path, nil
}
