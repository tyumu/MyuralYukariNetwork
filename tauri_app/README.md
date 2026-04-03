# Minimal Tauri App

This folder adds a minimal desktop shell for the architecture:

- Tauri: frontend + desktop runtime
- Go: main backend API
- Python: MemU sidecar API

## What is included

- `vite` frontend with two checks:
  - Tauri command invocation (`greet`)
  - HTTP call to Go backend `/health`
- Tauri Rust backend (`src-tauri`) with one command: `greet`

## Quick start

1. Install JS dependencies:

```bash
cd tauri_app
npm install
```

2. Start desktop app in dev mode:

```bash
npm run tauri dev
```

3. (Optional) Build frontend assets only:

```bash
npm run build
```

## Notes

- By default, the sample Go API URL is `http://127.0.0.1:8080`.
- Update it in the UI if your Go backend runs on another port.
- `bundle.active` is disabled for minimal setup; enable it when preparing installers.
