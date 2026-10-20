// Command leakcheck scans tracked git files for forbidden strings.
//
// Usage: go run ./internal/tools/leakcheck
//
// It reads .forbidden-strings (one pattern per line, blank lines and
// # comments ignored) from the repo root and fails if any tracked file
// contains any of them. This runs locally before pushing; it cannot
// run in CI because CI must not know the forbidden strings.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	patterns, err := loadPatterns(".forbidden-strings")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("leakcheck: .forbidden-strings not found, skipping")
			return
		}
		fatal(err)
	}
	if len(patterns) == 0 {
		fmt.Println("leakcheck: no patterns in .forbidden-strings, skipping")
		return
	}

	files, err := trackedFiles()
	if err != nil {
		fatal(err)
	}

	var found []string
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			continue // file may be deleted but still tracked
		}
		lower := bytes.ToLower(content)
		for _, p := range patterns {
			if bytes.Contains(lower, []byte(strings.ToLower(p))) {
				found = append(found, fmt.Sprintf("  %s contains %q", f, p))
			}
		}
	}

	if len(found) > 0 {
		fmt.Println("leakcheck: FORBIDDEN STRINGS FOUND")
		for _, f := range found {
			fmt.Println(f)
		}
		os.Exit(1)
	}
	fmt.Printf("leakcheck: %d files checked, no leaks\n", len(files))
}

func loadPatterns(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var patterns []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, sc.Err()
}

func trackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "leakcheck: %v\n", err)
	os.Exit(2)
}
