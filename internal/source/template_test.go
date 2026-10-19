package source

import (
	"testing"
)

// Fake templates matching the testdata fixtures — no real site paths.
const (
	FakeSearchPath = "/p/find?q={query}"
	FakePersonPath = "/p/find?person={sid}"
	FakeClassPath  = "/e/{event}/class/{class}?day={day}"
	FakeEventPath  = "/e/{event}"
)

func TestNewTemplate_Valid(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		names   []string
	}{
		{"search", FakeSearchPath, []string{"query"}},
		{"person", FakePersonPath, []string{"sid"}},
		{"class", FakeClassPath, []string{"event", "class", "day"}},
		{"event", FakeEventPath, []string{"event"}},
		{"no placeholders", "/static/page", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := NewTemplate(tt.pattern)
			if err != nil {
				t.Fatalf("NewTemplate(%q) error: %v", tt.pattern, err)
			}
			got := tmpl.Names()
			if len(got) != len(tt.names) {
				t.Fatalf("names = %v, want %v", got, tt.names)
			}
			for i := range got {
				if got[i] != tt.names[i] {
					t.Errorf("name[%d] = %q, want %q", i, got[i], tt.names[i])
				}
			}
		})
	}
}

func TestNewTemplate_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{"empty", ""},
		{"unclosed", "/e/{event/class"},
		{"empty name", "/e/{}/class"},
		{"bad name chars", "/e/{event-id}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTemplate(tt.pattern)
			if err == nil {
				t.Errorf("NewTemplate(%q) should fail", tt.pattern)
			}
		})
	}
}

func TestTemplate_Expand(t *testing.T) {
	tmpl, _ := NewTemplate(FakeClassPath)
	got, err := tmpl.Expand(map[string]string{
		"event": "2026058",
		"class": "M21",
		"day":   "0",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "/e/2026058/class/M21?day=0"
	if got != want {
		t.Errorf("Expand = %q, want %q", got, want)
	}
}

func TestTemplate_Expand_MissingKey(t *testing.T) {
	tmpl, _ := NewTemplate(FakeClassPath)
	_, err := tmpl.Expand(map[string]string{"event": "1"})
	if err == nil {
		t.Error("expected error for missing keys")
	}
}

func TestTemplate_Expand_EncodesValues(t *testing.T) {
	tmpl, _ := NewTemplate(FakeSearchPath)
	got, err := tmpl.Expand(map[string]string{"query": "Vardenis Pavardenis"})
	if err != nil {
		t.Fatal(err)
	}
	want := "/p/find?q=Vardenis%20Pavardenis"
	if got != want {
		t.Errorf("Expand = %q, want %q", got, want)
	}
}

func TestTemplate_Extract(t *testing.T) {
	tmpl, _ := NewTemplate(FakeClassPath)
	vals := tmpl.Extract("/e/2026058/class/M21?day=0")
	if vals == nil {
		t.Fatal("Extract returned nil")
	}
	checks := map[string]string{"event": "2026058", "class": "M21", "day": "0"}
	for k, want := range checks {
		if vals[k] != want {
			t.Errorf("Extract[%q] = %q, want %q", k, vals[k], want)
		}
	}
}

func TestTemplate_Extract_NoMatch(t *testing.T) {
	tmpl, _ := NewTemplate(FakeClassPath)
	vals := tmpl.Extract("/completely/different/path")
	if vals != nil {
		t.Errorf("Extract should return nil for non-matching path, got %v", vals)
	}
}

func TestTemplate_Extract_PersonFromHref(t *testing.T) {
	// Parsers use this to extract sids from href attributes in search results.
	tmpl, _ := NewTemplate("/p/profile/{sid}")
	vals := tmpl.Extract("/p/profile/772")
	if vals == nil {
		t.Fatal("Extract returned nil")
	}
	if vals["sid"] != "772" {
		t.Errorf("sid = %q, want %q", vals["sid"], "772")
	}
}

func TestNewTemplates(t *testing.T) {
	ts, err := NewTemplates(FakeSearchPath, FakePersonPath, FakeClassPath, FakeEventPath)
	if err != nil {
		t.Fatal(err)
	}
	if ts.Search == nil || ts.Person == nil || ts.Class == nil || ts.Event == nil {
		t.Error("all templates should be non-nil")
	}
}

func TestNewTemplates_PropagatesError(t *testing.T) {
	_, err := NewTemplates("", FakePersonPath, FakeClassPath, FakeEventPath)
	if err == nil {
		t.Error("expected error for empty search template")
	}
}
