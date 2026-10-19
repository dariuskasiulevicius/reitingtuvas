package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// version is set by the linker at build time.
var version = "dev"

func main() {
	exePath, err := execDir()
	if err != nil {
		exitWithError(fmt.Sprintf("Nepavyko nustatyti programos vietos: %v", err))
	}
	_ = exePath // will be used for workbook path in M6

	fmt.Printf("reitingtuvas %s\n", version)
	fmt.Println("Spauskite Enter, kad uždarytumėte.")
	waitForEnter()
}

// execDir returns the directory where the executable lives.
// On macOS double-click the working directory is $HOME, so we
// resolve relative to the executable, not the cwd.
func execDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func exitWithError(msg string) {
	fmt.Fprintf(os.Stderr, "Klaida: %s\n", msg)
	fmt.Println("Spauskite Enter, kad uždarytumėte.")
	waitForEnter()
	os.Exit(1)
}

func waitForEnter() {
	buf := make([]byte, 1)
	os.Stdin.Read(buf) //nolint:errcheck
}
