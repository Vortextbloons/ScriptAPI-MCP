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
	diskCache  *diskCache
	lastSource string
}

// LastFetchSource reports whether the last registry request used npm or cache.
func (c *Client) LastFetchSource() string { return c.lastSource }

// NewClient creates a new npm Client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		cache:     NewCache(),
		diskCache: newDiskCache(),
	}
}

// FetchVersionMatrix fetches version data for a module from npm
func (c *Client) FetchVersionMatrix(module string) (*VersionMatrix, error) {
	cacheKey := "versions:" + module
	if data, ok := c.cache.Get(cacheKey); ok {
		var vm VersionMatrix
		if err := json.Unmarshal(data, &vm); err == nil {
			c.lastSource = "memory cache"
			return &vm, nil
		}
	}
	if data, savedAt, ok := c.diskCache.Get(cacheKey); ok && time.Since(savedAt) <= versionTTL {
		var vm VersionMatrix
		if json.Unmarshal(data, &vm) == nil {
			c.cache.Set(cacheKey, data, versionTTL)
			c.lastSource = "disk cache saved " + savedAt.UTC().Format(time.RFC3339)
			return &vm, nil
		}
	}

	url := fmt.Sprintf("%s/%s", registryURL, module)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		if data, savedAt, ok := c.diskCache.Get(cacheKey); ok {
			var vm VersionMatrix
			if json.Unmarshal(data, &vm) == nil {
				c.lastSource = "offline fallback from disk cache saved " + savedAt.UTC().Format(time.RFC3339)
				return &vm, nil
			}
		}
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
	c.diskCache.Set(cacheKey, data)
	c.lastSource = "npm registry"
	return vm, nil
}

// FetchTypes fetches the .d.ts file for a specific module version
func (c *Client) FetchTypes(module, version string) ([]byte, error) {
	cacheKey := fmt.Sprintf("dts:%s@%s", module, version)
	if data, ok := c.cache.Get(cacheKey); ok {
		c.lastSource = "memory cache"
		return data, nil
	}
	if data, savedAt, ok := c.diskCache.Get(cacheKey); ok && time.Since(savedAt) <= dtsTTL {
		c.cache.Set(cacheKey, data, dtsTTL)
		c.lastSource = "disk cache saved " + savedAt.UTC().Format(time.RFC3339)
		return data, nil
	}

	// Get tarball URL from registry
	url := fmt.Sprintf("%s/%s/%s", registryURL, module, version)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		if data, savedAt, ok := c.diskCache.Get(cacheKey); ok {
			c.lastSource = "offline fallback from disk cache saved " + savedAt.UTC().Format(time.RFC3339)
			return data, nil
		}
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
		if data, savedAt, ok := c.diskCache.Get(cacheKey); ok {
			c.lastSource = "offline fallback from disk cache saved " + savedAt.UTC().Format(time.RFC3339)
			return data, nil
		}
		return nil, fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	// Extract index.d.ts from tarball
	dts, err := extractDTS(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to extract .d.ts: %w", err)
	}

	c.cache.Set(cacheKey, dts, dtsTTL)
	c.diskCache.Set(cacheKey, dts)
	c.lastSource = "npm registry"
	return dts, nil
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

// VersionMatrix holds parsed npm version data for a module
type VersionMatrix struct {
	Module   string            `json:"module"`
	Versions []string          `json:"versions"`
	Tags     map[string]string `json:"tags"`
}
