package source

import (
	"testing"
	"time"
)

func TestCache_PutAndGet_FinalEvent(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatal(err)
	}

	url := "https://example.org/e/123/class/M21?day=1"
	body := []byte("<html>test</html>")

	if err := c.Put(url, body); err != nil {
		t.Fatal(err)
	}

	// Event 30 days ago, finalDays=14 → should return cached
	eventDate := time.Now().AddDate(0, 0, -30)
	got, ok := c.Get(url, eventDate, 14, false)
	if !ok {
		t.Fatal("expected cache hit for final event")
	}
	if string(got) != string(body) {
		t.Errorf("body = %q", got)
	}
}

func TestCache_RecentEvent_NotCached(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatal(err)
	}

	url := "https://example.org/e/456/class/W35?day=1"
	body := []byte("<html>recent</html>")
	_ = c.Put(url, body)

	// Event 5 days ago, finalDays=14 → should NOT return cached
	eventDate := time.Now().AddDate(0, 0, -5)
	_, ok := c.Get(url, eventDate, 14, false)
	if ok {
		t.Error("expected cache miss for recent event")
	}
}

func TestCache_AlwaysFresh_Bypasses(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatal(err)
	}

	url := "https://example.org/p/find?person=772"
	_ = c.Put(url, []byte("<html>person</html>"))

	// Even with old event date, alwaysFresh=true → miss
	eventDate := time.Now().AddDate(0, 0, -100)
	_, ok := c.Get(url, eventDate, 14, true)
	if ok {
		t.Error("expected cache miss with alwaysFresh=true")
	}
}

func TestCache_Miss_NoEntry(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := c.Get("https://example.org/nonexistent", time.Now(), 14, false)
	if ok {
		t.Error("expected cache miss for nonexistent entry")
	}
}
