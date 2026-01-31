# Web Terminal Dependencies

The web terminal requires xterm.js and addons. The dependencies are **already installed** and embedded in the Go binary.

## Installed Packages

This directory contains:
- `@xterm/xterm` - Terminal emulator
- `@xterm/addon-fit` - Fit terminal to container
- `@xterm/addon-web-links` - Clickable links in terminal output

## How It Works

When building the Go binary, the entire `web/` directory is embedded using `go:embed web/*` in `handler_http.go`. This includes all the node_modules files, making the web terminal work offline without requiring a CDN.

## Updating Dependencies

To update the xterm packages:

```bash
cd web
npm install @xterm/xterm @xterm/addon-fit @xterm/addon-web-links
```

After updating, rebuild your Go binary to embed the new versions.

## Directory Structure

```
web/
├── node_modules/
│   └── @xterm/
│       ├── xterm/
│       │   ├── css/xterm.css
│       │   └── lib/xterm.js
│       ├── addon-fit/
│       │   └── lib/addon-fit.js
│       └── addon-web-links/
│           └── lib/addon-web-links.js
├── admin/
│   ├── index.html  (web terminal UI)
│   └── index.js    (terminal logic)
├── index.html      (landing page)
└── package.json
```

## Usage

The web terminal is available at `/admin` when using HTTPHandler:

```go
httpHandler := consolekit.NewHTTPHandler(executor, ":8080", "admin", "password")
httpHandler.Start()
// Access at: http://localhost:8080/admin
```

## File Sizes

The embedded xterm files add approximately 500KB to the binary size, but enable a fully functional offline web terminal.
