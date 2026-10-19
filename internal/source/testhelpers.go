package source

// Fake URL templates for use in tests and fixtures.
// These paths are made up and must not match any real site.
const (
	FakeBaseURL    = "https://results.example.org"
	FakeSearch     = "/p/find?q={query}"
	FakePerson     = "/p/find?person={sid}"
	FakeClass      = "/e/{event}/class/{class}?day={day}"
	FakeEvent      = "/e/{event}"
	FakeProfile    = "/p/profile/{sid}"
)

// FakeTemplates returns a Templates instance using the fake paths.
// Panics on error (safe in tests).
func FakeTemplates() *Templates {
	t, err := NewTemplates(FakeSearch, FakePerson, FakeClass, FakeEvent)
	if err != nil {
		panic(err)
	}
	return t
}
