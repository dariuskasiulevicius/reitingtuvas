package source

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProgressFunc is called before each fetch with current index, total, and description.
type ProgressFunc func(current, total int, desc string)

// FetchStats tracks request and cache counters.
type FetchStats struct {
	Requests   int // HTTP requests sent
	CacheHits  int // pages served from cache
	Errors     int // pages that failed after all retries
}

// Total returns the total number of fetch calls (requests + cache hits + errors).
func (s FetchStats) Total() int { return s.Requests + s.CacheHits + s.Errors }

// Client fetches pages sequentially with delay and back-off.
type Client struct {
	baseURL   string
	delay     time.Duration
	userAgent string
	http      *http.Client
	cache     *Cache
	finalDays int
	lastFetch time.Time
	backoffs  []time.Duration // retry back-off durations
	Stats     FetchStats
}

// NewClient creates a new sequential HTTP client.
func NewClient(baseURL string, delay time.Duration, cache *Cache, finalDays int, version string) *Client {
	return &Client{
		baseURL:   baseURL,
		delay:     delay,
		userAgent: fmt.Sprintf("reitingtuvas/%s", version),
		http:      &http.Client{Timeout: 30 * time.Second},
		cache:     cache,
		finalDays: finalDays,
		backoffs:  []time.Duration{0, 5 * time.Second, 30 * time.Second},
	}
}

// FetchResult holds the outcome of a fetch.
type FetchResult struct {
	Body    []byte
	FromCache bool
	Error   error
}

// Fetch downloads a page, respecting delay and using cache when applicable.
// path is the URL path (from template expansion).
// eventDate is used for cache decisions; zero means always fresh.
// alwaysFresh=true bypasses cache entirely (for search/person pages).
func (c *Client) Fetch(path string, eventDate time.Time, alwaysFresh bool) FetchResult {
	url := c.baseURL + path

	// Check cache first
	if c.cache != nil {
		if body, ok := c.cache.Get(url, eventDate, c.finalDays, alwaysFresh); ok {
			c.Stats.CacheHits++
			return FetchResult{Body: body, FromCache: true}
		}
	}

	// Enforce delay
	c.waitDelay()

	// Fetch with back-off: 0s → 5s → 30s → give up
	var lastErr error

	for attempt, backoff := range c.backoffs {
		if attempt > 0 {
			time.Sleep(backoff)
		}

		c.Stats.Requests++
		body, err := c.doRequest(url)
		if err == nil {
			// Store in cache
			if c.cache != nil {
				_ = c.cache.Put(url, body)
			}
			return FetchResult{Body: body}
		}
		lastErr = err
	}

	c.Stats.Errors++
	return FetchResult{Error: fmt.Errorf("after 3 attempts: %w", lastErr)}
}

// Post fetches via HTTP POST with form data, using cache and retry logic.
func (c *Client) Post(path string, formData url.Values, eventDate time.Time, alwaysFresh bool) FetchResult {
	fullURL := c.baseURL + path
	cacheKey := postCacheKey(fullURL, formData)

	if c.cache != nil {
		if body, ok := c.cache.GetByKey(cacheKey, eventDate, c.finalDays, alwaysFresh); ok {
			c.Stats.CacheHits++
			return FetchResult{Body: body, FromCache: true}
		}
	}

	c.waitDelay()

	var lastErr error
	for attempt, backoff := range c.backoffs {
		if attempt > 0 {
			time.Sleep(backoff)
		}
		c.Stats.Requests++
		body, err := c.doPost(fullURL, formData)
		if err == nil {
			if c.cache != nil {
				_ = c.cache.PutByKey(cacheKey, fullURL+"|POST", body)
			}
			return FetchResult{Body: body}
		}
		lastErr = err
	}

	c.Stats.Errors++
	return FetchResult{Error: fmt.Errorf("after 3 attempts: %w", lastErr)}
}

func (c *Client) doPost(rawURL string, formData url.Values) ([]byte, error) {
	req, err := http.NewRequest("POST", rawURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 || resp.StatusCode >= 500 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	c.lastFetch = time.Now()
	return body, nil
}

func postCacheKey(rawURL string, formData url.Values) string {
	h := sha256.Sum256([]byte(rawURL + "|POST|" + formData.Encode()))
	return fmt.Sprintf("%x", h)
}

func (c *Client) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 || resp.StatusCode >= 500 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	c.lastFetch = time.Now()
	return body, nil
}

func (c *Client) waitDelay() {
	if c.lastFetch.IsZero() {
		return
	}
	elapsed := time.Since(c.lastFetch)
	if elapsed < c.delay {
		time.Sleep(c.delay - elapsed)
	}
}
