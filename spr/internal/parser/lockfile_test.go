package parser

import (
	"testing"

	"github.com/acheong08/hackeurope-spr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLockfile(t *testing.T) {
	lm := NewLockfileManager()

	rootPackage := &models.Package{
		ID:      "demo-package@0.0.1",
		Name:    "demo-package",
		Version: "0.0.1",
	}

	graph, err := lm.ParseLockfile("../../../poc/small-test/package-lock.json", rootPackage)
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Check root package
	assert.Equal(t, "demo-package", graph.RootPackage.Name)
	assert.Equal(t, "0.0.1", graph.RootPackage.Version)

	// Check that we have nodes
	assert.Greater(t, len(graph.Nodes), 0, "Should have parsed some packages")

	// Check direct dependencies
	directDeps := graph.GetDirectDependencies()
	assert.Greater(t, len(directDeps), 0, "Should have direct dependencies")

	// Print some info for debugging
	t.Logf("Total packages: %d", len(graph.Nodes))
	t.Logf("Direct dependencies: %d", len(directDeps))

	for _, dep := range directDeps {
		t.Logf("Direct dep: %s@%s", dep.Name, dep.Version)
	}
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		// Simple packages
		{"node_modules/lodash", "lodash"},
		{"node_modules/express", "express"},

		// Scoped packages
		{"node_modules/@sveltejs/kit", "@sveltejs/kit"},
		{"node_modules/@types/node", "@types/node"},
		{"node_modules/@tailwindcss/vite", "@tailwindcss/vite"},

		// Nested dependencies (returns the package at that path, not parent)
		{"node_modules/foo/node_modules/bar", "bar"},
		{"node_modules/lodash/node_modules/@types/node", "@types/node"},

		// Scoped packages with nested deps
		{"node_modules/@scope/pkg/node_modules/dep", "dep"},

		// Scoped package with nested scoped dep
		{"node_modules/@scope/pkg/node_modules/@other/dep", "@other/dep"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractPackageName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
