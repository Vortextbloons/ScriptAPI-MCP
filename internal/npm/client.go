package npm

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	registryURL = "https://registry.npmjs.org"
	versionTTL  = 30 * time.Second
	dtsTTL      = 2 * time.Minute
)

// Client is an HTTP client for the npm registry with caching
type Client struct {
	httpClient *http.Client
	cache      *Cache
}

// NewClient creates a new npm Client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		cache: NewCache(),
	}
}

// FetchVersionMatrix fetches version data for a module from npm
func (c *Client) FetchVersionMatrix(module string) (*VersionMatrix, error) {
	cacheKey := "versions:" + module
	if data, ok := c.cache.Get(cacheKey); ok {
		var vm VersionMatrix
		if err := json.Unmarshal(data, &vm); err == nil {
			return &vm, nil
		}
	}

	url := fmt.Sprintf("%s/%s", registryURL, module)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version matrix: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned status %d", resp.StatusCode)
	}

	var payload struct {
		Versions map[string]struct{} `json:"versions"`
		DistTags map[string]string   `json:"dist-tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode npm response: %w", err)
	}

	vm := &VersionMatrix{
		Module:   module,
		Versions: make([]string, 0, len(payload.Versions)),
		Tags:     payload.DistTags,
	}
	for v := range payload.Versions {
		vm.Versions = append(vm.Versions, v)
	}
	sort.Strings(vm.Versions)

	data, _ := json.Marshal(vm)
	c.cache.Set(cacheKey, data, versionTTL)
	return vm, nil
}

// FetchVersionData fetches metadata for a specific module version from npm
func (c *Client) FetchVersionData(module, version string) (map[string]any, error) {
	cacheKey := "metadata:" + module + "@" + version
	if data, ok := c.cache.Get(cacheKey); ok {
		var result map[string]any
		if err := json.Unmarshal(data, &result); err == nil {
			return result, nil
		}
	}

	url := fmt.Sprintf("%s/%s/%s", registryURL, module, version)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode version data: %w", err)
	}

	data, _ := json.Marshal(result)
	c.cache.Set(cacheKey, data, dtsTTL)
	return result, nil
}

// FetchTypes fetches the .d.ts file for a specific module version
func (c *Client) FetchTypes(module, version string) ([]byte, error) {
	cacheKey := fmt.Sprintf("dts:%s@%s", module, version)
	if data, ok := c.cache.Get(cacheKey); ok {
		return data, nil
	}

	// Get tarball URL from registry
	url := fmt.Sprintf("%s/%s/%s", registryURL, module, version)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned status %d", resp.StatusCode)
	}

	var payload struct {
		Dist struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode package metadata: %w", err)
	}

	// Download tarball
	resp, err = c.httpClient.Get(payload.Dist.Tarball)
	if err != nil {
		return nil, fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	// Extract index.d.ts from tarball
	dts, err := extractDTS(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to extract .d.ts: %w", err)
	}

	c.cache.Set(cacheKey, dts, dtsTTL)
	return dts, nil
}

// ClearCache clears all cached data to force fresh fetches
func (c *Client) ClearCache() {
	c.cache.Clear()
}

// ClearVersionCache clears cached version data for a specific module
func (c *Client) ClearVersionCache(module string) {
	c.cache.Delete("versions:" + module)
}

func extractDTS(r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(hdr.Name, "index.d.ts") && !strings.Contains(hdr.Name, "node_modules") {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("index.d.ts not found in tarball")
}

// VersionMatrix is re-exported for convenience
type VersionMatrix struct {
	Module   string            `json:"module"`
	Versions []string          `json:"versions"`
	Tags     map[string]string `json:"tags"`
}
