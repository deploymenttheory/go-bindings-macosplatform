package docc

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// baseURL is the DocC render-JSON endpoint. Page URLs are
// <baseURL>/<framework>/<symbol>.json with all path segments lowercased.
const baseURL = "https://developer.apple.com/tutorials/data/documentation"

// docPageURL is the human-readable page, stored in the sidecar for reference.
const docPageURL = "https://developer.apple.com/documentation"

// ErrNotFound is returned for a 404 — a symbol with no published page.
var ErrNotFound = errors.New("doc page not found")

// Client fetches DocC render JSON with a polite rate limit, retries, and an
// on-disk response cache so re-runs (and tests) avoid the network.
type Client struct {
	HTTP      *http.Client
	CacheDir  string        // when non-empty, responses are cached here
	MinDelay  time.Duration // minimum gap between live requests
	UserAgent string
	MaxRetry  int

	lastRequest time.Time
}

// NewClient returns a Client with sensible defaults.
func NewClient(cacheDir string, minDelay time.Duration) *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		CacheDir:  cacheDir,
		MinDelay:  minDelay,
		UserAgent: "go-bindings-macosplatform-appledocs/1.0 (+https://github.com/deploymenttheory/go-bindings-macosplatform)",
		MaxRetry:  3,
	}
}

// RenderURL builds the render-JSON URL for a symbol path (already lowercased
// segments joined by "/", e.g. "foundation/nsstring").
func RenderURL(symbolPath string) string {
	return baseURL + "/" + symbolPath + ".json"
}

// PageURL builds the human-readable documentation URL for a symbol path.
func PageURL(symbolPath string) string {
	return docPageURL + "/" + symbolPath
}

// FetchObjC fetches the render JSON for symbolPath and returns the
// Objective-C-projected RenderNode. Returns ErrNotFound on a 404.
func (c *Client) FetchObjC(symbolPath string) (*RenderNode, error) {
	data, err := c.fetchRaw(symbolPath)
	if err != nil {
		return nil, err
	}
	return ParseObjC(data)
}

func (c *Client) fetchRaw(symbolPath string) ([]byte, error) {
	if c.CacheDir != "" {
		if data, hit := c.readCache(symbolPath); hit {
			if string(data) == "404" {
				return nil, ErrNotFound
			}
			return data, nil
		}
	}

	url := RenderURL(symbolPath)
	var lastErr error
	for attempt := 0; attempt <= c.MaxRetry; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
		c.throttle()
		data, status, err := c.do(url)
		switch {
		case err != nil:
			lastErr = err
		case status == http.StatusNotFound:
			c.writeCache(symbolPath, []byte("404")) // negative cache
			return nil, ErrNotFound
		case status == http.StatusOK:
			if c.CacheDir != "" {
				c.writeCache(symbolPath, data)
			}
			return data, nil
		case status == http.StatusTooManyRequests || status >= 500:
			lastErr = fmt.Errorf("%s: HTTP %d", url, status)
		default:
			return nil, fmt.Errorf("%s: HTTP %d", url, status)
		}
	}
	return nil, fmt.Errorf("%s: %w", url, lastErr)
}

func (c *Client) do(url string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

func (c *Client) throttle() {
	if c.MinDelay <= 0 {
		return
	}
	if elapsed := time.Since(c.lastRequest); elapsed < c.MinDelay {
		time.Sleep(c.MinDelay - elapsed)
	}
	c.lastRequest = time.Now()
}

func (c *Client) cachePath(symbolPath string) string {
	sum := sha256.Sum256([]byte(symbolPath))
	name := strings.ReplaceAll(symbolPath, "/", "_") + "-" + hex.EncodeToString(sum[:6]) + ".json"
	return filepath.Join(c.CacheDir, name)
}

// readCache returns the cached body and whether a cache entry exists. A "404"
// body is a negative-cache hit (hit=true) that the caller maps to ErrNotFound.
func (c *Client) readCache(symbolPath string) ([]byte, bool) {
	data, err := os.ReadFile(c.cachePath(symbolPath))
	if err != nil {
		return nil, false
	}
	return data, true
}

func (c *Client) writeCache(symbolPath string, data []byte) {
	if err := os.MkdirAll(c.CacheDir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(c.cachePath(symbolPath), data, 0o644)
}
