# Benchmarks

Reproducible minimal-app comparison between **AnyUI**, **Gio**, **Fyne**, and **Wails** on Windows.

Each app creates a `480x320` window with:
- a title label
- a single-line text input
- a button with a click counter

Each sub-benchmark has its own `go.mod` so external UI dependencies never pollute the root `anyui` module.

## Running

```powershell
cd benchmarks
.\run.ps1
```

The script builds each binary with `-trimpath -ldflags="-s -w -H windowsgui"`, measures binary size, launches the app, samples the full process tree (including `msedgewebview2.exe` child processes for Wails) for Working Set RAM, Private Memory, and idle CPU, and writes `RESULTS.md`.

