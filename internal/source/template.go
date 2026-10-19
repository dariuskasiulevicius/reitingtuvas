package source

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Template is a URL path template with named placeholders like {event}.
// It can expand placeholders into a concrete path and extract values
// from href strings that match the same pattern.
type Template struct {
	raw   string
	regex *regexp.Regexp
	names []string // placeholder names in order
}

// NewTemplate compiles a path template. Placeholders are {name} where
// name is [a-zA-Z_]+. The template is split into literal and placeholder
// parts; literals are escaped for regex, placeholders become named groups.
//
// Example:
//
//	t := NewTemplate("/e/{event}/class/{class}?day={day}")
//	t.Expand(map[string]string{"event":"123","class":"M21","day":"0"})
//	// → "/e/123/class/M21?day=0"
func NewTemplate(pattern string) (*Template, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty template")
	}

	var (
		reStr strings.Builder
		names []string
		i     int
	)

	reStr.WriteString("^")
	for i < len(pattern) {
		open := strings.IndexByte(pattern[i:], '{')
		if open == -1 {
			reStr.WriteString(regexp.QuoteMeta(pattern[i:]))
			break
		}
		open += i

		// literal before the placeholder
		reStr.WriteString(regexp.QuoteMeta(pattern[i:open]))

		close := strings.IndexByte(pattern[open:], '}')
		if close == -1 {
			return nil, fmt.Errorf("unclosed placeholder at position %d", open)
		}
		close += open

		name := pattern[open+1 : close]
		if name == "" || !isValidName(name) {
			return nil, fmt.Errorf("invalid placeholder name %q", name)
		}
		names = append(names, name)

		// Named group: match one or more non-empty chars that aren't
		// a path separator or query-string delimiter, except for {query}
		// which may contain spaces (URL-encoded later).
		reStr.WriteString(fmt.Sprintf("(?P<%s>[^/&#]+)", name))

		i = close + 1
	}
	reStr.WriteString("$")

	re, err := regexp.Compile(reStr.String())
	if err != nil {
		return nil, fmt.Errorf("compile regex: %w", err)
	}

	return &Template{raw: pattern, regex: re, names: names}, nil
}

// Expand fills all placeholders with the given values and returns the path.
// Missing keys cause an error. Values are URL-path-escaped.
func (t *Template) Expand(vals map[string]string) (string, error) {
	result := t.raw
	for _, name := range t.names {
		v, ok := vals[name]
		if !ok {
			return "", fmt.Errorf("missing template value %q", name)
		}
		result = strings.Replace(result, "{"+name+"}", url.PathEscape(v), 1)
	}
	return result, nil
}

// Extract matches an href against the template and returns the named values.
// Returns nil if the href does not match.
func (t *Template) Extract(href string) map[string]string {
	m := t.regex.FindStringSubmatch(href)
	if m == nil {
		return nil
	}
	vals := make(map[string]string, len(t.names))
	for i, name := range t.regex.SubexpNames() {
		if name != "" && i < len(m) {
			v, err := url.PathUnescape(m[i])
			if err != nil {
				v = m[i]
			}
			vals[name] = v
		}
	}
	return vals
}

// Names returns the placeholder names in template order.
func (t *Template) Names() []string {
	out := make([]string, len(t.names))
	copy(out, t.names)
	return out
}

// String returns the original template string.
func (t *Template) String() string {
	return t.raw
}

func isValidName(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
			return false
		}
	}
	return true
}

// Templates holds the parsed URL templates read from Settings.
type Templates struct {
	Search  *Template // search by name
	Person  *Template // person results by sid
	Class   *Template // class results: event, class, day
	Event   *Template // event page
}

// NewTemplates parses all template strings. Returns an error if any is invalid.
func NewTemplates(search, person, class, event string) (*Templates, error) {
	s, err := NewTemplate(search)
	if err != nil {
		return nil, fmt.Errorf("search template: %w", err)
	}
	p, err := NewTemplate(person)
	if err != nil {
		return nil, fmt.Errorf("person template: %w", err)
	}
	c, err := NewTemplate(class)
	if err != nil {
		return nil, fmt.Errorf("class template: %w", err)
	}
	e, err := NewTemplate(event)
	if err != nil {
		return nil, fmt.Errorf("event template: %w", err)
	}
	return &Templates{Search: s, Person: p, Class: c, Event: e}, nil
}
