package registry

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/acheong08/hackeurope-spr/pkg/models"
)

// LogCallback is an optional function for forwarding log messages (e.g. to WebSocket).
type LogCallback func(message, level string)

// Uploader handles uploading packages to Gitea registry
type Uploader struct {
	BaseURL     string
	Owner       string
	Token       string
	Concurrency int
	HTTPClient  *http.Client
	logCb       LogCallback
}

// NewUploader creates a new registry uploader
func NewUploader(baseURL, owner, token string) *Uploader {
	return &Uploader{
		BaseURL:     strings.TrimSuffix(baseURL, "/"),
		Owner:       owner,
		Token:       token,
		Concurrency: 10,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// SetLogCallback sets an optional callback for forwarding log messages.
func (u *Uploader) SetLogCallback(cb LogCallback) {
	u.logCb = cb
}

// logMsg prints to console and optionally forwards via the log callback.
func (u *Uploader) logMsg(message, level string) {
	log.Printf("%s", message)
	if u.logCb != nil {
		u.logCb(message, level)
	}
}

// PackageExists checks if a specific package version already exists in the registry
// Uses the npm registry protocol: GET /api/packages/{owner}/npm/{packageName}
// Returns true only if the specific version exists
func (u *Uploader) PackageExists(ctx context.Context, name, version string) (bool, error) {
	// Normalize package name for URL
	pkgPath := normalizePackageName(name)
	url := fmt.Sprintf("%s/api/packages/%s/npm/%s", u.BaseURL, u.Owner, pkgPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+u.Token)

	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check package existence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Parse response to check if specific version exists
		var pkgMetadata struct {
			Versions map[string]interface{} `json:"versions"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&pkgMetadata); err != nil {
			return false, fmt.Errorf("failed to decode package metadata: %w", err)
		}
		_, versionExists := pkgMetadata.Versions[version]
		return versionExists, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// DownloadTarball downloads a package tarball from npm
func (u *Uploader) DownloadTarball(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download tarball: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tarball: %w", err)
	}

	return data, nil
}

// FetchPackageMetadata fetches normalized package metadata from npm registry API
// This returns properly structured metadata (bin as object, repository as object, etc.)
func (u *Uploader) FetchPackageMetadata(ctx context.Context, name, version string) (map[string]interface{}, error) {
	// Handle scoped packages in URL
	urlName := name
	if strings.HasPrefix(name, "@") {
		urlName = strings.Replace(name, "/", "%2F", 1)
	}

	url := fmt.Sprintf("https://registry.npmjs.org/%s/%s", urlName, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// npm registry doesn't require auth for public packages
	req.Header.Set("Accept", "application/json")

	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch metadata: status %d", resp.StatusCode)
	}

	var metadata map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return metadata, nil
}

// UploadPackageWithMetadata uploads a package to the Gitea registry using npm protocol
// Uses pre-fetched metadata from npm API (already normalized) instead of extracting from tarball
func (u *Uploader) UploadPackageWithMetadata(ctx context.Context, name, version string, tarball []byte, apiMetadata map[string]interface{}) error {
	// Normalize package name for URL path
	pkgPath := normalizePackageName(name)

	url := fmt.Sprintf("%s/api/packages/%s/npm/%s", u.BaseURL, u.Owner, pkgPath)

	// Build the npm metadata JSON using API metadata (already normalized)
	metadata, err := u.buildMetadataFromAPI(name, version, tarball, apiMetadata)
	if err != nil {
		return fmt.Errorf("failed to build metadata: %w", err)
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(metadataJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+u.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload package: %w", err)
	}
	defer resp.Body.Close()

	// Success codes
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	// Package already exists - not an error
	if resp.StatusCode == http.StatusConflict {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to upload package: status %d, body: %s", resp.StatusCode, string(body))
}

// normalizeBinField ensures the bin field is in object format
// npm allows bin to be a string (e.g., "bin": "./cli.js") which needs to be
// converted to an object like {"package-name": "./cli.js"} for Gitea registry
func normalizeBinField(apiMetadata map[string]interface{}, pkgName string) map[string]interface{} {
	if bin, ok := apiMetadata["bin"]; ok {
		switch v := bin.(type) {
		case string:
			// Convert string to object using unscoped package name as key
			unscopedName := pkgName
			if strings.HasPrefix(pkgName, "@") {
				parts := strings.SplitN(pkgName, "/", 2)
				if len(parts) == 2 {
					unscopedName = parts[1]
				}
			}
			return map[string]interface{}{
				unscopedName: v,
			}
		case map[string]interface{}:
			// Already in correct format
			return v
		}
	}
	return nil
}

// normalizeRepositoryField ensures the repository field is in object format
// npm allows repository to be a string (e.g., "github:user/repo" or "user/repo")
// which needs to be converted to {"type": "git", "url": "..."} for Gitea registry
func normalizeRepositoryField(apiMetadata map[string]interface{}) map[string]interface{} {
	if repo, ok := apiMetadata["repository"]; ok {
		switch v := repo.(type) {
		case string:
			// Convert string to object
			url := v
			repoType := "git"

			// Handle shorthand formats
			if strings.HasPrefix(v, "github:") {
				// github:user/repo -> git+https://github.com/user/repo.git
				path := strings.TrimPrefix(v, "github:")
				url = "git+https://github.com/" + path + ".git"
			} else if strings.HasPrefix(v, "gitlab:") {
				// gitlab:user/repo -> git+https://gitlab.com/user/repo.git
				path := strings.TrimPrefix(v, "gitlab:")
				url = "git+https://gitlab.com/" + path + ".git"
			} else if strings.HasPrefix(v, "bitbucket:") {
				// bitbucket:user/repo -> git+https://bitbucket.org/user/repo.git
				path := strings.TrimPrefix(v, "bitbucket:")
				url = "git+https://bitbucket.org/" + path + ".git"
			} else if strings.HasPrefix(v, "gist:") {
				// gist:gistId
				gistId := strings.TrimPrefix(v, "gist:")
				url = "https://gist.github.com/" + gistId
				repoType = "gist"
			} else if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") && !strings.HasPrefix(v, "git+") && strings.Contains(v, "/") {
				// user/repo shorthand -> https://github.com/user/repo
				// But only if it looks like a shorthand (contains / but isn't a full URL)
				url = "git+https://github.com/" + v + ".git"
			}

			return map[string]interface{}{
				"type": repoType,
				"url":  url,
			}
		case map[string]interface{}:
			// Already in object format - ensure it has url field
			if _, hasURL := v["url"]; hasURL {
				return v
			}
			// Invalid object without url field, skip it
			return nil
		}
	}
	return nil
}

// buildMetadataFromAPI constructs npm package metadata JSON using pre-fetched API metadata
// The npm registry API already returns normalized fields (bin as object, repository as object, etc.)
func (u *Uploader) buildMetadataFromAPI(name, version string, tarball []byte, apiMetadata map[string]interface{}) (map[string]interface{}, error) {
	// Calculate hashes
	hash512 := sha512.Sum512(tarball)
	hash1 := sha1.Sum(tarball)
	integrity := fmt.Sprintf("sha512-%s", base64.StdEncoding.EncodeToString(hash512[:]))
	shasum := fmt.Sprintf("%x", hash1[:])

	// Create tarball filename
	tarballName := name
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			tarballName = parts[1]
		}
	}
	tarballFileName := fmt.Sprintf("%s-%s.tgz", tarballName, version)

	// Construct tarball URL
	tarballURL := fmt.Sprintf("%s/api/packages/%s/npm/%s/-/%s",
		u.BaseURL, u.Owner, normalizePackageName(name), tarballFileName)

	// Build manifest with required fields
	manifest := map[string]interface{}{
		"_id":     fmt.Sprintf("%s@%s", name, version),
		"name":    name,
		"version": version,
		"dist": map[string]interface{}{
			"integrity": integrity,
			"shasum":    shasum,
			"tarball":   tarballURL,
		},
	}

	// Copy normalized fields from API metadata (npm registry already normalizes these)
	if apiMetadata != nil {
		// Essential fields for npx and module resolution
		for _, field := range []string{"scripts", "main", "module", "type"} {
			if val, ok := apiMetadata[field]; ok {
				manifest[field] = val
			}
		}

		// Handle bin field specially - needs normalization from string to object
		if binVal := normalizeBinField(apiMetadata, name); binVal != nil {
			manifest["bin"] = binVal
		}

		// Handle repository field specially - needs normalization from string to object
		if repoVal := normalizeRepositoryField(apiMetadata); repoVal != nil {
			manifest["repository"] = repoVal
		}

		// Optional metadata fields (all already normalized by npm API)
		for _, field := range []string{"description", "author", "license", "keywords", "homepage", "bugs", "engines", "os", "cpu", "dependencies", "peerDependencies", "devDependencies", "bundledDependencies", "optionalDependencies"} {
			if val, ok := apiMetadata[field]; ok {
				manifest[field] = val
			}
		}
	}

	// Build root metadata
	root := map[string]interface{}{
		"_id":       name,
		"name":      name,
		"dist-tags": map[string]string{"latest": version},
		"versions": map[string]interface{}{
			version: manifest,
		},
		"_attachments": map[string]interface{}{
			tarballFileName: map[string]interface{}{
				"content_type": "application/octet-stream",
				"data":         base64.StdEncoding.EncodeToString(tarball),
				"length":       len(tarball),
			},
		},
	}

	return root, nil
}

// UploadGraph uploads all packages in the dependency graph
func (u *Uploader) UploadGraph(ctx context.Context, graph *models.DependencyGraph) error {
	// Filter out root package and collect all nodes
	var nodes []*models.PackageNode
	for _, node := range graph.Nodes {
		if graph.RootPackage != nil && node.ID == graph.RootPackage.ID {
			continue // Skip root package
		}
		nodes = append(nodes, node)
	}

	// Check for non-npm dependencies
	nonNpmDeps := u.extractNonNpmDeps(nodes)
	if len(nonNpmDeps) > 0 {
		return fmt.Errorf("unsupported non-npm dependencies found: %v. These dependency types are not yet supported", nonNpmDeps)
	}

	u.logMsg(fmt.Sprintf("Uploading %d packages to Gitea registry...", len(nodes)), "info")

	// Upload npm packages with worker pool
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, u.Concurrency)
	errChan := make(chan error, 1) // Buffered to hold first error
	var processedCount int
	var mu sync.Mutex
	var stopChan = make(chan struct{})

	for _, node := range nodes {
		wg.Add(1)
		go func(n *models.PackageNode) {
			defer wg.Done()

			select {
			case <-stopChan:
				return // Skip if error already occurred
			default:
			}

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := u.uploadNode(ctx, n); err != nil {
				select {
				case errChan <- fmt.Errorf("failed to upload %s: %w", n.ID, err):
					close(stopChan) // Signal other goroutines to stop
				default:
					// Error already reported, ignore
				}
				return
			}

			mu.Lock()
			processedCount++
			u.logMsg(fmt.Sprintf("[%d/%d] Uploaded: %s@%s", processedCount, len(nodes), n.Name, n.Version), "info")
			mu.Unlock()
		}(node)
	}

	wg.Wait()
	close(errChan)

	// Check if any error occurred
	if err := <-errChan; err != nil {
		return err
	}

	u.logMsg(fmt.Sprintf("Successfully uploaded %d packages", len(nodes)), "success")
	return nil
}

// uploadNode uploads a single package node
func (u *Uploader) uploadNode(ctx context.Context, node *models.PackageNode) error {
	// Check if already exists
	exists, err := u.PackageExists(ctx, node.Name, node.Version)
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}
	if exists {
		return nil // Skip existing packages
	}

	// Fetch normalized metadata from npm registry API
	// This gives us properly structured fields (bin as object, repository as object, etc.)
	metadata, err := u.FetchPackageMetadata(ctx, node.Name, node.Version)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata for %s@%s: %w", node.Name, node.Version, err)
	}

	// Get tarball URL - construct from npm registry if not provided
	tarballURL := node.ResolvedURL
	if tarballURL == "" {
		tarballURL = constructNpmTarballURL(node.Name, node.Version)
	}

	// Download tarball
	tarball, err := u.DownloadTarball(ctx, tarballURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	// Upload to registry with API metadata (already normalized)
	if err := u.UploadPackageWithMetadata(ctx, node.Name, node.Version, tarball, metadata); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	return nil
}

// extractNonNpmDeps extracts non-npm dependency URLs from nodes
func (u *Uploader) extractNonNpmDeps(nodes []*models.PackageNode) []string {
	var urls []string
	seen := make(map[string]bool)

	for _, node := range nodes {
		for _, depURL := range node.Dependencies {
			if isNonNpmDep(depURL) && !seen[depURL] {
				urls = append(urls, depURL)
				seen[depURL] = true
			}
		}
	}

	return urls
}

// isNonNpmDep checks if a dependency URL is not from npm registry
func isNonNpmDep(url string) bool {
	return strings.HasPrefix(url, "git+") ||
		strings.HasPrefix(url, "github:") ||
		strings.HasPrefix(url, "gitlab:") ||
		strings.HasPrefix(url, "bitbucket:") ||
		(strings.HasPrefix(url, "http://") && !strings.Contains(url, "registry.npmjs.org")) ||
		(strings.HasPrefix(url, "https://") && !strings.Contains(url, "registry.npmjs.org"))
}

// normalizePackageName normalizes a package name for URL
func normalizePackageName(name string) string {
	// Replace @scope/name with @scope%2fname
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			return parts[0] + "%2f" + parts[1]
		}
	}
	return name
}

// constructNpmTarballURL constructs the npm registry tarball URL for a package
// Format: https://registry.npmjs.org/@scope/name/-/name-{version}.tgz
//
//	https://registry.npmjs.org/name/-/name-{version}.tgz
func constructNpmTarballURL(name, version string) string {
	// Extract the unscoped name for the tarball filename
	tarballName := name
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			tarballName = parts[1]
		}
	}

	// The path uses the full name, tarball uses unscoped name
	return fmt.Sprintf("https://registry.npmjs.org/%s/-/%s-%s.tgz", name, tarballName, version)
}
