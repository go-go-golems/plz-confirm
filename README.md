# plz-confirm

**plz-confirm** enables AI agents and automated tools to request feedback from users through a rich web-based interface. When an agent needs user input (confirmation, selection, form data, file uploads, etc.), it calls a `plz-confirm` command, which displays an interactive dialog in the user's browser. The user responds via the browser, and the agent receives structured results to continue its workflow.

## Overview

plz-confirm bridges automated agents and human users through a request-response pattern:

1. **Agent creates a request**: The agent calls `plz-confirm confirm` (or another widget command) with dialog parameters
2. **Backend broadcasts**: The server stores the request and broadcasts it to connected web clients via WebSocket
3. **User gets notified**: The user's browser displays an interactive dialog and receives browser notifications
4. **User responds**: The user interacts with the dialog and submits their response
5. **Agent receives result**: The CLI command receives structured results and the agent continues execution

This architecture allows agents to leverage rich web UIs for complex interactions while keeping the command-line interface simple and scriptable.

## Features

- **Five Widget Types**: Confirmation dialogs, selection menus, forms, file uploads, and data tables
- **Real-time Communication**: WebSocket-based bidirectional communication between CLI and web UI
- **Browser Notifications**: Native browser notifications alert users when new requests arrive
- **Multiple Output Formats**: JSON, YAML, CSV, and table output formats
- **Session Management**: Support for multiple concurrent sessions
- **Production Ready**: Embedded frontend assets for easy deployment

## Quick Start

### Prerequisites

- Go 1.24+ (for building the CLI and server)
- Node.js and pnpm (for building the frontend)
- A web browser

### Installation

```bash
# Clone the repository
git clone https://github.com/go-go-golems/plz-confirm.git
cd plz-confirm

# Install frontend dependencies (required for building)
cd agent-ui-system
pnpm install
cd ..

# Build the project (builds frontend and embeds it into Go binary)
make build

# The binary will be built in the current directory
# Or build a specific binary:
go build -o plz-confirm ./cmd/plz-confirm
```

**Note**: The `make build` command runs `go generate ./...` which builds the frontend with Vite and embeds it into the Go binary. This is required for production builds.

### Running the Server

Start the plz-confirm server:

```bash
# After building with 'make build', run the server (defaults to :3000)
./plz-confirm serve

# Or specify a custom port
./plz-confirm serve --addr :3000

# Or build and run in one step (without embedding frontend)
go run ./cmd/plz-confirm serve
```

**Note**: If you run `go run` directly without building the frontend first, the server will start but won't serve the web UI (it will only serve the API). For production, always use `make build` first.

The server will serve the embedded frontend on port 3000 by default. Open `http://localhost:3000` in your browser.

### Development Mode

For frontend development with hot-reload:

```bash
# Terminal 1: Start the Go backend server on :3001 (Vite proxies to this)
go run ./cmd/plz-confirm serve --addr :3001

# Terminal 2: Start the Vite dev server (proxies API/WS to backend on :3001)
cd agent-ui-system
pnpm dev --host --port 3000
```

Open `http://localhost:3000` for the development UI (with hot-reload). In dev mode, Vite runs on :3000 and proxies API/WebSocket requests to the backend on :3001.

## Usage Examples

### Confirmation Dialog

Request a yes/no confirmation from the user:

```bash
plz-confirm confirm \
  --title "Deploy to Production" \
  --message "This will deploy the latest code to production. Continue?" \
  --approve-text "Deploy" \
  --reject-text "Cancel"
```

### Selection Menu

Present options for the user to choose from:

```bash
plz-confirm select \
  --title "Select Environment" \
  --option production \
  --option staging \
  --option development \
  --searchable
```

### Form Input

Collect structured form data:

```bash
plz-confirm form \
  --title "Database Configuration" \
  --schema schema.json
```

Where `schema.json` contains a JSON Schema definition:

```json
{
  "properties": {
    "host": { "type": "string" },
    "port": { "type": "number", "minimum": 1, "maximum": 65535 },
    "username": { "type": "string" },
    "password": { "type": "string", "format": "password" }
  },
  "required": ["host", "port", "username", "password"]
}
```

### File Upload

Request file uploads from the user:

```bash
plz-confirm upload \
  --title "Upload Log Files" \
  --accept .log \
  --accept .txt \
  --multiple \
  --max-size 5242880
```

### Data Table

Display tabular data and allow selection:

```bash
plz-confirm table \
  --title "Select Server" \
  --data servers.json \
  --columns name,status,region,cpu \
  --searchable \
  --multi-select
```

## Widget Commands

All widget commands support these common flags:

- `--base-url`: Base URL for the backend server (default: `http://localhost:3000`)
- `--timeout`: Request expiration in seconds (default: 300)
- `--wait-timeout`: How long to wait for a response in seconds (default: 60)
- `--output`: Output format: `table`, `json`, `yaml`, `csv` (default: `yaml`)

### Available Commands

- `plz-confirm confirm` - Yes/no confirmation dialogs
- `plz-confirm select` - Single or multi-select menus
- `plz-confirm form` - JSON Schema-based forms
- `plz-confirm upload` - File upload dialogs
- `plz-confirm table` - Data table with selection
- `plz-confirm serve` - Start the backend server

## Architecture

plz-confirm consists of three main components:

1. **CLI Client** (`cmd/plz-confirm/`): Command-line interface for agents to create requests
2. **Backend Server** (`internal/server/`): Go server handling API requests and WebSocket connections
3. **Web Frontend** (`agent-ui-system/client/`): React-based UI for user interactions

### Communication Flow

```
┌─────────┐         ┌──────────┐         ┌─────────┐
│  Agent  │────────▶│  Server  │────────▶│ Browser │
│  (CLI)  │  HTTP   │  (Go)    │ WebSocket│  (UI)   │
└─────────┘         └──────────┘         └─────────┘
     │                    │                      │
     │                    │                      │
     └────────────────────┼──────────────────────┘
                          │
                    (Response)
```

## Building for Production

To build the frontend and embed it into the Go binary:

```bash
# Ensure frontend dependencies are installed
cd agent-ui-system
pnpm install
cd ..

# Build everything (frontend + Go binary with embedded assets)
make build

# Or build manually:
# 1. Build and embed frontend
go generate ./internal/server

# 2. Build the Go binary
go build -o plz-confirm ./cmd/plz-confirm
```

The `make build` command:
1. Runs `go generate ./...` which builds the frontend with Vite and copies it to `internal/server/embed/public/`
2. Runs `go build ./...` which compiles the Go code with embedded frontend assets

The embedded frontend will be served automatically when running `plz-confirm serve`.

## Configuration

### Server Options

```bash
plz-confirm serve --addr :3000
```

- `--addr`: Address to listen on (default: `:3000`)

### Client Options

All widget commands support:

- `--base-url`: Backend server URL
- `--timeout`: Request expiration time
- `--wait-timeout`: Response wait time
- `--output`: Output format (table/json/yaml/csv)

## Documentation

For detailed documentation, see:

- **User Guide**: `plz-confirm help how-to-use` or see `pkg/doc/how-to-use.md`
- **Command Reference**: `plz-confirm help <command>`

### For LLM Agents

The complete usage guide is available via:

```bash
plz-confirm help how-to-use
```

This command outputs a comprehensive guide covering all widget types, command-line options, examples, and usage patterns. **This is the documentation you should provide to LLM agents** to enable them to use plz-confirm effectively. The guide includes:

- Complete command reference for all widget types
- Detailed examples for each widget
- Common flags and options
- Output format specifications
- Best practices and use cases

You can also view the guide directly in the repository at `pkg/doc/how-to-use.md`.

## Development

### Project Structure

```
plz-confirm/
├── cmd/plz-confirm/          # CLI entry point
├── internal/
│   ├── cli/                  # Widget command implementations
│   ├── server/               # HTTP/WebSocket server
│   ├── store/                 # Request storage
│   └── types/                # Shared types
├── agent-ui-system/          # React frontend
│   └── client/               # Frontend source code
└── pkg/doc/                  # Documentation
```

### Running Tests

```bash
# Run Go tests
go test ./...

# Run frontend tests
cd agent-ui-system
pnpm test
```

### Development Scripts

The project includes tmux scripts for easy development setup:

```bash
# Start server and frontend dev server in tmux
bash ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/scripts/tmux-up.sh

# Attach to tmux session
tmux attach -t DESIGN-PLZ-CONFIRM-001
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

See [LICENSE](LICENSE) file for details.

## Related Projects

- [Glazed](https://github.com/go-go-golems/glazed) - CLI framework used by plz-confirm
