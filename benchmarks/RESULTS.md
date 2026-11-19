# UI Framework Benchmarks (Windows)

**Test Machine:** Майкрософт Windows 11 Pro (64-bit), AMD Ryzen 5 5600G with Radeon Graphics (12 logical CPUs), 31.4 GB RAM, `go version go1.26.2 windows/amd64`

Minimal window (480x320) with a label, text input, and click counter button.
Measured after 5s warmup over a 5s idle window (including child processes such as WebView2).

| Framework | CGO | Binary Size | Startup Time | Processes | Working Set (RAM) | Private Memory | Idle CPU |
| --- | :---: | ---: | ---: | ---: | ---: | ---: | ---: |
| AnyUI | 0 | 2.74 MB | 16.80 ms | 1 | 38.59 MB | 26.13 MB | 0% |
| Gio | 0 | 6.52 MB | 158.37 ms | 1 | 55.47 MB | 38.61 MB | 0% |
| Fyne | 1 | 23.29 MB | 248.72 ms | 1 | 169.39 MB | 157.43 MB | 0% |
| Wails | 0 | 10.66 MB | 526.04 ms | 7 | 409.12 MB | 198.73 MB | 0% |

- **WebView2 Runtime (required by Wails):** 878,0 MB (v157.0.4341.60) on disk.
- **Startup Time measurement:** time from main() start to first rendered frame (AnyUI, Gio), foreground lifecycle (Fyne), or OnDomReady (Wails).
