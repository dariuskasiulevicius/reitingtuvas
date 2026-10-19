# Go single binary, run by double-click

The tool is run by non-technical club people on Windows and macOS, so it ships as a Go single binary (built by GitHub Actions for Windows x64, macOS Intel and Apple Silicon) that runs on double-click, reading its config and participant list from files next to the executable and writing the report into the same folder. Python + uv was the natural fit for scraping and iterating on the rating formula, but it requires a terminal for every run, which is where non-technical users get stuck.

## Consequences

- Binaries are unsigned for now: users must click through Windows SmartScreen ("More info → Run anyway") and macOS Gatekeeper once (macOS 15+: System Settings → Privacy & Security → "Open Anyway"; the old right-click → Open bypass no longer works). Code signing would remove this but costs a paid Apple developer account and a Windows certificate.
- Double-clicking a plain executable on macOS runs it with the home folder as working directory, so all files are resolved relative to the executable's own location, never the working directory.
