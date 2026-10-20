package source

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Fetch_Success(t *testing.T) {
	body := "<html><body>test page</body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/e/123" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 0, nil, 14, "test")
	res := c.Fetch("/e/123", time.Time{}, true)
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if string(res.Body) != body {
		t.Errorf("body = %q", res.Body)
	}
	if res.FromCache {
		t.Error("should not be from cache")
	}
}

func TestClient_Fetch_BackoffOn500(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(500)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 0, nil, 14, "test")
	c.backoffs = []time.Duration{0, 0, 0} // no sleep in tests
	res := c.Fetch("/test", time.Time{}, true)
	if res.Error != nil {
		t.Fatalf("expected success after retries, got %v", res.Error)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestClient_Fetch_AllFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 0, nil, 14, "test")
	c.backoffs = []time.Duration{0, 0, 0}
	res := c.Fetch("/fail", time.Time{}, true)
	if res.Error == nil {
		t.Error("expected error after all attempts fail")
	}
}

func TestClient_Fetch_UsesCache(t *testing.T) {
	fetchCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchCount++
		w.Write([]byte("from server"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	cache, _ := NewCache(dir)

	c := NewClient(srv.URL, 0, cache, 14, "test")

	// First fetch: hits server, caches
	eventDate := time.Now().AddDate(0, 0, -30) // final
	res := c.Fetch("/e/old/class/M21?day=1", eventDate, false)
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if res.FromCache {
		t.Error("first fetch should not be from cache")
	}
	if fetchCount != 1 {
		t.Errorf("fetchCount = %d", fetchCount)
	}

	// Second fetch: same URL, final event → cache hit
	res = c.Fetch("/e/old/class/M21?day=1", eventDate, false)
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if !res.FromCache {
		t.Error("second fetch should be from cache")
	}
	if fetchCount != 1 {
		t.Errorf("fetchCount = %d, should still be 1", fetchCount)
	}
}

func TestClient_Fetch_UserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 0, nil, 14, "1.0.0")
	c.Fetch("/ua", time.Time{}, true)
	if gotUA != "reitingtuvas/1.0.0" {
		t.Errorf("User-Agent = %q", gotUA)
	}
}
