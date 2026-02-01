# Wails SQLiter Browser

A native Desktop application for browsing SQLite databases, built with [Wails](https://wails.io) and React.

## Features

- **Native Performance**: Uses Go for backend logic and SQLite interactions.
- **Native UI**: Uses OS-native file dialogs.
- **Security**: No local web server or open ports.
- **Simple**: Drop-in SQLite file viewing.

## Development

### Prerequisites
- Go 1.21+
- Node.js / npm
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Setup
```bash
# Initialize dependency
go mod tidy
go work use .

# Run in dev mode
wails dev
```

### Build
```bash
wails build
```

The binary will be in `build/bin/`.

## Architecture
See [WailsStart.md](./WailsStart.md) for the architectural philosophy and implementation details.
