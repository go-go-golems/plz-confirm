---
Title: How to Use plz-confirm
Slug: how-to-use
Short: Complete guide to using plz-confirm with setup, widget commands, and practical examples
Topics:
- guide
- usage
- tutorial
- examples
Commands:
- confirm
- select
- form
- table
- upload
- serve
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

plz-confirm enables AI agents and automated tools to request feedback from users through a web-based interface. When an agent needs user input (confirmation, selection, form data, etc.), it calls a `plz-confirm` command, which displays an interactive dialog in the user's browser. The user responds via the browser, and the agent receives the result to continue its workflow.

## How It Works

plz-confirm bridges automated agents and human users through a request-response pattern:

1. **Agent creates a request**: The agent (running in a CLI script or automated workflow) calls `plz-confirm confirm` (or another widget command) with dialog parameters.
2. **Backend broadcasts**: The server stores the request and broadcasts it to connected web clients via WebSocket.
3. **User gets notified**: The user's browser displays an interactive dialog (confirmation, form, file upload, etc.) based on the request type.
4. **User responds**: The user interacts with the dialog in their browser and submits their response.
5. **Agent receives result**: The CLI command receives structured results and the agent continues execution.

This architecture allows agents to leverage rich web UIs for complex interactions while keeping the command-line interface simple and scriptable. The agent uses the CLI, while the user interacts through their browser.

## Quick Setup

### Prerequisites

- The plz-confirm server must be running (typically started by your system administrator or deployment)
- A web browser for receiving notifications

### Step 1: Ensure the Server is Running

The plz-confirm server handles API requests and WebSocket connections. It should be running and accessible (typically at `http://localhost:3000` or a configured URL).

### Step 2: Open the Web UI

Open the plz-confirm web interface in your browser (the URL will be provided by your administrator). You should see the plz-confirm interface with a "SYSTEM_IDLE" status, indicating it's waiting for requests from agents.

### Step 3: Agents Will Request Your Input

When an agent needs your feedback, it will call a `plz-confirm` command. You'll see a notification appear in your browser with an interactive dialog. For example, an agent might request confirmation:

```bash
# This command is run by the agent (you don't run this yourself)
plz-confirm confirm \
  --title "Deploy to Production" \
  --message "This will deploy the latest code to production. Continue?" \
  --approve-text "Deploy" \
  --reject-text "Cancel"
```

When this happens, a confirmation dialog will appear in your browser. Click "Deploy" or "Cancel" to provide your response. The agent will receive your choice and continue its workflow.

**Note**: As a user, you interact only through the browser. The CLI commands are used by agents and automated tools, not by end users directly.

## Widget Commands

plz-confirm supports five widget types, each designed for a specific interaction pattern. All widget commands share common flags for server connection and timeouts, plus widget-specific parameters.

### Common Flags

All widget commands support these flags:

- `--base-url`: Base URL for the backend server (default: `http://localhost:3000` for dev proxy)
- `--timeout`: Request expiration in seconds (default: 300)
- `--wait-timeout`: How long to wait for a response in seconds (default: 60)
- `--output`: Output format: `table`, `json`, `yaml`, `csv` (default: `yaml`)

### Confirm Command

The `confirm` command displays a yes/no confirmation dialog with customizable button text.

**Use cases:**
- Deployment confirmations
- Destructive operation warnings
- User consent for data processing

**Example:**

```bash
plz-confirm confirm \
  --title "Delete Database" \
  --message "This will permanently delete the production database. This action cannot be undone." \
  --approve-text "Delete" \
  --reject-text "Cancel"
```

**Output columns:**
- `request_id`: Unique identifier for the request
- `approved`: Boolean indicating user's choice
- `timestamp`: ISO 8601 timestamp of the response

**Using in scripts:**

```bash
APPROVED=$(plz-confirm confirm \
  --title "Continue?" \
  --output json | jq -r '.approved')

if [ "$APPROVED" != "true" ]; then
  echo "User cancelled, exiting."
  exit 1
fi
```

### Select Command

The `select` command displays a list of options for single or multi-selection.

**Use cases:**
- Region selection for deployments
- Environment selection (dev/staging/prod)
- Multi-select from a list of items

**Example:**

```bash
plz-confirm select \
  --title "Select Region" \
  --option us-east-1 \
  --option us-west-2 \
  --option eu-central-1 \
  --option ap-northeast-1 \
  --searchable \
  --multi
```

**Output columns:**
- `request_id`: Unique identifier for the request
- `selected_json`: JSON array of selected options (or single string if `--multi` not used)

**Using in scripts:**

```bash
REGION=$(plz-confirm select \
  --title "Select Deployment Region" \
  --option us-east-1 \
  --option us-west-2 \
  --output json | jq -r '.selected_json')

echo "Deploying to region: $REGION"
```

### Form Command

The `form` command displays a dynamic form based on a JSON Schema definition.

**Use cases:**
- User registration forms
- Configuration wizards
- Data entry with validation

**Example:**

Create a schema file `user-schema.json`:

```json
{
  "properties": {
    "username": {
      "type": "string",
      "minLength": 3,
      "title": "Username"
    },
    "email": {
      "type": "string",
      "format": "email",
      "title": "Email Address"
    },
    "accessLevel": {
      "type": "number",
      "minimum": 1,
      "maximum": 5,
      "title": "Access Level"
    }
  },
  "required": ["username", "email"]
}
```

Then use it:

```bash
plz-confirm form \
  --title "Create User Account" \
  --schema @user-schema.json
```

**Output columns:**
- `request_id`: Unique identifier for the request
- `data_json`: JSON object containing form field values

**Using in scripts:**

```bash
USER_DATA=$(plz-confirm form \
  --title "Create New User" \
  --schema @user-schema.json \
  --output json | jq -r '.data_json')

USERNAME=$(echo "$USER_DATA" | jq -r '.username')
EMAIL=$(echo "$USER_DATA" | jq -r '.email')
echo "Creating user: $USERNAME ($EMAIL)"
```

### Table Command

The `table` command displays tabular data with row selection capabilities.

**Use cases:**
- Server selection from a list
- Database record selection
- Multi-row operations

**Example:**

Create a data file `servers.json`:

```json
[
  {"id": 1, "name": "server-1", "status": "running", "region": "us-east-1"},
  {"id": 2, "name": "server-2", "status": "stopped", "region": "us-west-2"},
  {"id": 3, "name": "server-3", "status": "running", "region": "eu-central-1"}
]
```

Then use it:

```bash
plz-confirm table \
  --title "Select Server" \
  --data @servers.json \
  --columns name,status,region \
  --searchable \
  --multi-select
```

**Output columns:**
- `request_id`: Unique identifier for the request
- `selected_json`: JSON object (or array if `--multi-select`) containing selected row(s)

**Using in scripts:**

```bash
SELECTED=$(plz-confirm table \
  --title "Select Server to Restart" \
  --data @servers.json \
  --columns name,status,region \
  --output json | jq -r '.selected_json')

SERVER_ID=$(echo "$SELECTED" | jq -r '.id')
echo "Restarting server ID: $SERVER_ID"
```

### Upload Command

The `upload` command displays a file upload dialog with type and size restrictions.

**Use cases:**
- Log file uploads
- Configuration file imports
- Bulk data uploads

**Example:**

```bash
plz-confirm upload \
  --title "Upload Log Files" \
  --accept .log \
  --accept .txt \
  --accept text/plain \
  --multiple \
  --max-size 5242880
```

**Output columns:**
- `request_id`: Unique identifier for the request
- `file_name`: Name of the uploaded file
- `file_size`: Size in bytes
- `file_path`: Server-side path to the file
- `mime_type`: MIME type of the file

Each uploaded file appears as a separate row in the output.

**Using in scripts:**

```bash
UPLOAD_RESULT=$(plz-confirm upload \
  --title "Upload Log Files" \
  --accept .log \
  --multiple \
  --output json)

echo "$UPLOAD_RESULT" | jq -c '.[]' | while read -r file; do
  FILE_NAME=$(echo "$file" | jq -r '.file_name')
  echo "Processing: $FILE_NAME"
done
```

## Practical Examples

These examples show how agents use plz-confirm commands in their workflows. As a user, you'll see the dialogs in your browser when agents run these commands.

### Deployment Confirmation Workflow

An agent adds safety checks to deployment scripts:

```bash
#!/bin/bash
set -e

# Request user confirmation before deploying
APPROVED=$(plz-confirm confirm \
  --title "Deploy to Production" \
  --message "Deploy version $(git describe --tags) to production?" \
  --approve-text "Deploy" \
  --reject-text "Cancel" \
  --output json | jq -r '.approved')

if [ "$APPROVED" != "true" ]; then
  echo "Deployment cancelled by user"
  exit 1
fi

# Continue with deployment
echo "Deploying..."
# ... deployment logic ...
```

### Multi-Step Configuration Workflow

An agent combines multiple widgets for complex workflows:

```bash
#!/bin/bash
set -e

# Step 1: Confirm deployment
APPROVED=$(plz-confirm confirm \
  --title "Deploy Application" \
  --message "This will deploy the latest version. Continue?" \
  --output json | jq -r '.approved')

[ "$APPROVED" = "true" ] || exit 1

# Step 2: Select environment
ENV=$(plz-confirm select \
  --title "Select Environment" \
  --option production \
  --option staging \
  --option development \
  --output json | jq -r '.selected_json')

# Step 3: Select region
REGION=$(plz-confirm select \
  --title "Select Region" \
  --option us-east-1 \
  --option us-west-2 \
  --output json | jq -r '.selected_json')

# Step 4: Execute deployment
echo "Deploying to $ENV in $REGION..."
# ... deployment logic ...
```

### Dynamic Server Selection

An agent selects servers from an API response and requests user confirmation:

```bash
#!/bin/bash

# Fetch server list from API
curl -s https://api.example.com/servers | jq '[.[] | {id, name, status, region}]' > servers.json

# Let user select a server
SELECTED=$(plz-confirm table \
  --title "Select Server to Restart" \
  --data @servers.json \
  --columns name,status,region \
  --searchable \
  --output json | jq -r '.selected_json')

SERVER_ID=$(echo "$SELECTED" | jq -r '.id')
SERVER_NAME=$(echo "$SELECTED" | jq -r '.name')

echo "Restarting server: $SERVER_NAME (ID: $SERVER_ID)"
# ... restart logic ...
```

### User Registration Form

An agent collects structured user data with validation:

```bash
#!/bin/bash

# Create a JSON Schema for user registration
cat > user-schema.json <<EOF
{
  "properties": {
    "username": {"type": "string", "minLength": 3},
    "email": {"type": "string", "format": "email"},
    "role": {
      "type": "string",
      "enum": ["admin", "user", "viewer"]
    }
  },
  "required": ["username", "email", "role"]
}
EOF

# Collect user data
USER_DATA=$(plz-confirm form \
  --title "Create New User" \
  --schema @user-schema.json \
  --output json | jq -r '.data_json')

# Use the data
USERNAME=$(echo "$USER_DATA" | jq -r '.username')
EMAIL=$(echo "$USER_DATA" | jq -r '.email')
ROLE=$(echo "$USER_DATA" | jq -r '.role')

echo "Creating user: $USERNAME ($EMAIL) with role: $ROLE"
# ... user creation logic ...
```

### File Upload Processing

An agent requests file uploads and processes them:

```bash
#!/bin/bash

# Request log file upload
UPLOAD_RESULT=$(plz-confirm upload \
  --title "Upload Log Files for Analysis" \
  --accept .log \
  --accept .txt \
  --multiple \
  --max-size 10485760 \
  --output json)

# Process each uploaded file
echo "$UPLOAD_RESULT" | jq -c '.[]' | while read -r file; do
  FILE_PATH=$(echo "$file" | jq -r '.file_path')
  FILE_NAME=$(echo "$file" | jq -r '.file_name')
  
  echo "Analyzing: $FILE_NAME"
  # ... analysis logic ...
done
```

## Output Formats

plz-confirm uses Glazed's output layers, so agents can format results for different use cases:

**YAML format (default):**
```bash
plz-confirm confirm --title "Continue?"
```

**JSON format (for scripting):**
```bash
plz-confirm confirm --title "Continue?" --output json
```

**Table format:**
```bash
plz-confirm confirm --title "Continue?" --output table
```

**CSV format:**
```bash
plz-confirm confirm --title "Continue?" --output csv
```

The JSON output is particularly useful for agent scripts, as it can be easily parsed with `jq`:

```bash
# Agent script example
if plz-confirm confirm --title "Continue?" --output json | jq -e '.approved' > /dev/null; then
  echo "User approved, continuing..."
else
  echo "User cancelled, exiting."
  exit 1
fi
```

## Tips for Users

- **Keep the browser open**: The web UI must be open and connected to receive notifications from agents.
- **Respond promptly**: Agents wait for your response (default timeout is 60 seconds). If you don't respond in time, the request will timeout.
- **Check notifications**: When an agent requests your input, a dialog will appear in your browser. Review the request carefully before responding.

## Tips for Agent Developers

- **Use JSON output for scripting**: The `--output json` flag makes it easy to parse results with `jq` or other JSON tools.
- **Set appropriate timeouts**: Use `--wait-timeout` to control how long the CLI waits for user input. Default is 60 seconds.
- **Combine widgets for complex workflows**: Chain multiple widget commands together for multi-step processes.
- **Use file inputs**: The `@file.json` syntax (or `-` for stdin) makes it easy to pass complex data to `form` and `table` commands.
- **Handle timeouts gracefully**: If a user doesn't respond in time, the command will exit with an error. Handle this case in your agent scripts.

