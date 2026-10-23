package source

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheEntry is stored as JSON alongside the cached body.
type CacheEntry struct {
	URL       string    `json:"url"`
	FetchedAt time.Time `json:"fetched_at"`
}

// Cache is a disk-based HTTP response cache.
// File name = sha256 of the URL. Body in .html, metadata in .json.
type Cache struct {
	dir string
}

// NewCache creates a cache in the given directory, creating it if needed.
func NewCache(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &Cache{dir: dir}, nil
}

// Get returns the cached body if it exists and is still valid.
// eventDate is the event's start date; finalDays controls when results are
// considered final (older than finalDays → cached forever).
// For pages that should always be re-fetched (search, person results),
// pass alwaysFresh=true.
func (c *Cache) Get(url string, eventDate time.Time, finalDays int, alwaysFresh bool) ([]byte, bool) {
	return c.GetByKey(c.key(url), eventDate, finalDays, alwaysFresh)
}

// GetByKey is like Get but accepts a pre-computed cache key (for POST requests).
func (c *Cache) GetByKey(key string, eventDate time.Time, finalDays int, alwaysFresh bool) ([]byte, bool) {
	if alwaysFresh {
		return nil, false
	}

	metaPath := filepath.Join(c.dir, key+".json")
	bodyPath := filepath.Join(c.dir, key+".html")

	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, false
	}

	var entry CacheEntry
	if err := json.Unmarshal(metaBytes, &entry); err != nil {
		return nil, false
	}

	// If event is older than finalDays, results are final → cache forever
	cutoff := time.Now().AddDate(0, 0, -finalDays)
	if !eventDate.IsZero() && eventDate.Before(cutoff) {
		body, err := os.ReadFile(bodyPath)
		if err != nil {
			return nil, false
		}
		return body, true
	}

	// Event is recent → don't use cache (re-fetch)
	return nil, false
}

// Put stores a response body in the cache.
func (c *Cache) Put(url string, body []byte) error {
	return c.PutByKey(c.key(url), url, body)
}

// PutByKey stores a response body under a pre-computed cache key.
func (c *Cache) PutByKey(key, label string, body []byte) error {
	entry := CacheEntry{
		URL:       label,
		FetchedAt: time.Now(),
	}
	metaBytes, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	bodyPath := filepath.Join(c.dir, key+".html")
	metaPath := filepath.Join(c.dir, key+".json")

	if err := os.WriteFile(bodyPath, body, 0o644); err != nil {
		return fmt.Errorf("write cache body: %w", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0o644); err != nil {
		return fmt.Errorf("write cache meta: %w", err)
	}
	return nil
}

// Key computes the cache key for a URL.
func (c *Cache) Key(url string) string {
	return c.key(url)
}

func (c *Cache) key(url string) string {
	h := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", h)
}
