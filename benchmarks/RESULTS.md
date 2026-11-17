# UI Framework Benchmarks (Windows)

**Test Machine:** Microsoft Windows 11 Pro (64-bit), AMD Ryzen 5 5600G with Radeon Graphics (12 logical CPUs), 31.4 GB RAM, `go version go1.26.2 windows/amd64`

Minimal window (480x320) with a label, text input, and click counter button.
Measured after 5s warmup over a 5s idle window (including child processes such as WebView2).

| Framework | CGO | Binary Size | Startup Time | Processes | Working Set (RAM) | Private Memory | Idle CPU |
| --- | :---: | ---: | ---: | ---: | ---: | ---: | ---: |
| AnyUI | 0 | 2.7 MB | 17.56 ms | 1 | 38.49 MB | 28.64 MB | 0% |
| Gio | 0 | 6.52 MB | 117.93 ms | 1 | 52.74 MB | 38.75 MB | 0% |
| Fyne | 1 | 23.29 MB | 229.49 ms | 1 | 128.1 MB | 159.79 MB | 0% |
| Wails | 0 | 10.66 MB | 364.00 ms | 7 | 339.62 MB | 188.65 MB | 0% |

- **WebView2 Runtime (required by Wails):** 878,2 MB (v157.0.4341.46) on disk.
- **Startup Time measurement:** time from main() start to first rendered frame (AnyUI, Gio), foreground lifecycle (Fyne), or OnDomReady (Wails).
