package registry

import (
	"context"
	"testing"

	"github.com/acheong08/hackeurope-spr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePackageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"lodash", "lodash"},
		{"@sveltejs/kit", "@sveltejs%2fkit"},
		{"@types/node", "@types%2fnode"},
		{"express", "express"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePackageName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNonNpmDep(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz", false},
		{"git+https://github.com/user/repo.git", true},
		{"github:user/repo", true},
		{"https://example.com/lib.tgz", true},
		{"http://internal.com/package.tgz", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isNonNpmDep(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUploaderPackageExists(t *testing.T) {
	// This test requires a real Gitea instance
	// Skip if not configured
	token := "4ff7d1cf00600a6a2fde771f62bc6c3ebb87b80c"
	if token == "" {
		t.Skip("No registry token provided")
	}

	uploader := NewUploader("https://git.duti.dev", "acheong08", token)

	ctx := context.Background()

	// Test with a package that likely doesn't exist
	exists, err := uploader.PackageExists(ctx, "test-package-that-does-not-exist", "1.0.0")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUploaderDownloadTarball(t *testing.T) {
	uploader := NewUploader("https://git.duti.dev", "acheong08", "")

	ctx := context.Background()

	// Download a real npm package (lodash)
	url := "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz"
	data, err := uploader.DownloadTarball(ctx, url)

	require.NoError(t, err)
	assert.Greater(t, len(data), 1000, "Should have downloaded some data")
}

func TestExtractNonNpmDeps(t *testing.T) {
	uploader := NewUploader("", "", "")

	nodes := []*models.PackageNode{
		{
			Package: models.Package{Name: "test", Version: "1.0.0"},
			Dependencies: map[string]string{
				"lodash":      "^4.17.21",
				"private-lib": "git+https://github.com/org/repo.git",
			},
		},
		{
			Package: models.Package{Name: "another", Version: "2.0.0"},
			Dependencies: map[string]string{
				"custom":      "https://example.com/package.tgz",
				"private-lib": "git+https://github.com/org/repo.git", // Duplicate
			},
		},
	}

	urls := uploader.extractNonNpmDeps(nodes)

	assert.Len(t, urls, 2, "Should have 2 unique non-npm deps")
	assert.Contains(t, urls, "git+https://github.com/org/repo.git")
	assert.Contains(t, urls, "https://example.com/package.tgz")
}

func TestConstructNpmTarballURL(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "lodash",
			version:  "4.17.21",
			expected: "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz",
		},
		{
			name:     "@sveltejs/kit",
			version:  "2.52.2",
			expected: "https://registry.npmjs.org/@sveltejs/kit/-/kit-2.52.2.tgz",
		},
		{
			name:     "@types/node",
			version:  "20.0.0",
			expected: "https://registry.npmjs.org/@types/node/-/node-20.0.0.tgz",
		},
		{
			name:     "express",
			version:  "4.18.2",
			expected: "https://registry.npmjs.org/express/-/express-4.18.2.tgz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constructNpmTarballURL(tt.name, tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}
