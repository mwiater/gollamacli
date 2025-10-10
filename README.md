# gollamacli

`gollamacli` is a terminal-first companion for interacting with large language models that expose the Ollama API. It helps you browse available hosts, launch an immersive chat session, and keep model inventories aligned across machines, whether you are experimenting locally or coordinating a distributed cluster.

![Alt text](.screens/multichat_01.png?raw=true "Multichat")

## Feature Highlights
- **Multiple host management** – Define any number of Ollama hosts in `config.json`, switch between them instantly, and keep connection details in one place.
- **Interactive chat interface** – Drive a focused terminal UI for conversational work with the model you select.
- **Multimodel chat mode** – Assign up to four hosts/models at once and chat with them side by side in a coordinated layout.
- **Model synchronization** – Use a single command to pull models that are missing from a host and prune those not listed in configuration.
- **Comprehensive model tooling** – List, pull, delete, unload, and otherwise manage models without leaving the CLI.
- **Debug instrumentation** – Surface timing, token counts, and other diagnostics whenever you need deeper performance insight.

## Requirements
- Go toolchain installed (the project targets recent Go releases).
- Access to one or more running Ollama instances reachable from your terminal.

## Installation
Install the command with `go install`:

```bash
go install github.com/mwiater/gollamacli/cmd/gollamacli@latest
```

The resulting binary will be placed in your Go bin directory (e.g., `$GOPATH/bin`).

## Configuration
`gollamacli` reads its settings from a JSON document. Place `config.json` alongside the executable (or point to the file with `--config` at runtime). At minimum you must define one host.

### Example `config.json`
```json
{
  "hosts": [
    {
      "name": "Ollama01",
      "url": "http://192.168.0.10:11434",
      "type": "ollama",
      "models": [
        "stablelm-zephyr:3b",
        "granite3.3:2b",
        "gemma3n:e2b",
        "deepseek-r1:1.5b",
        "llama3.2:1b",
        "granite3.1-moe:1b",
        "dolphin-phi:2.7b",
        "qwen3:1.7b"
      ]
    },
    {
      "name": "Ollama02",
      "url": "http://192.168.0.10:11434",
      "type": "ollama",
      "models": [
        "stablelm-zephyr:3b",
        "granite3.3:2b",
        "gemma3n:e2b",
        "deepseek-r1:1.5b",
        "llama3.2:1b",
        "granite3.1-moe:1b",
        "dolphin-phi:2.7b",
        "qwen3:1.7b"
      ],
      "systemprompt": ""
    },
    {
      "name": "Ollama02",
      "url": "http://192.168.0.11:11434",
      "type": "ollama",
      "models": [
        "stablelm-zephyr:3b",
        "granite3.3:2b",
        "gemma3n:e2b"
      ],
      "systemprompt": ""
    }
  ],
  "debug": true,
  "multimodel": false
}
```

### Configuration Reference
- `hosts`: Array of host definitions (Ollama is the currently supported type).
  - `name`: A friendly label shown in the UI (e.g., `"Local Ollama"`).
  - `url`: Base URL of the Ollama API endpoint (`http://host:11434`).
  - `type`: Host backend identifier (`"ollama"`).
  - `models`: Desired model identifiers to monitor on the host.
  - `systemprompt`: Optional system prompt string. Leave empty to use the model default.
- `debug`: Boolean flag. When `true`, timing/token metrics are shown and `debug.log` captures detailed traces.
- `multimodel`: Boolean flag. When `true`, the CLI launches directly into the multimodel chat interface.

## Running the CLI

### Launch an Interactive Chat
Start a session with:

```bash
gollamacli chat
```

- If `multimodel` is `false`, the app opens in host selection mode. Pick a host, choose a loaded model (or request a load), and begin chatting in a scrollable viewport.
- If `multimodel` is `true`, the assignment view appears. Map hosts to columns, confirm your selections, and converse with multiple models concurrently.

### Model Management Commands
Manage the models across your hosts with dedicated subcommands:

- List models available on the configured hosts:
  ```bash
  gollamacli list models
  ```
- Pull models that are missing locally:
  ```bash
  gollamacli pull models
  ```
- Delete models that you no longer need:
  ```bash
  gollamacli delete models
  ```
- Synchronize hosts with the exact model inventory in `config.json`:
  ```bash
  gollamacli sync models
  ```
- Unload models from memory without deleting the artifacts:
  ```bash
  gollamacli unload models
  ```

### Keyboard Shortcuts (Chat Interface)
- `esc` or `Ctrl+c`: Quit the application.
- `Tab`: Return from the chat view to host/model selection.

## Debug Mode Details
With `debug` enabled in configuration, the chat interface displays:
- **Model load duration** – Time required to bring the model into memory.
- **Prompt eval duration/tokens** – Latency and token count for prompt processing.
- **Response eval duration/tokens** – Latency and tokens consumed during generation.
- **Total duration** – End-to-end latency from request to final token.

Each run also appends to `debug.log`, providing a persistent trace for troubleshooting.

## Building From Source
Release artifacts can be produced with GoReleaser. Create a snapshot build with:

```bash
goreleaser release --snapshot --clean --skip archive
```

Snapshots do not publish Git tags or remote releases. Output binaries are written to `dist`, and `--clean` removes any previous contents before the build.

## Additional Documentation
Generate API documentation locally using `godoc`:

```bash
go install -v golang.org/x/tools/cmd/godoc@latest
godoc -http=:6060
```

Open <http://localhost:6060> in a browser to explore the documentation.

## Testing
Run the entire suite:

```bash
go test ./...
```

## Coverage Reporting
Produce a coverage profile and summary:

```bash
go clean -testcache
go test ./... -coverprofile=.coverage/coverage.out
go tool cover -func=.coverage/coverage.out
```

## Contributing
Contributions are welcome. For major changes, open an issue to discuss the proposal before submitting a pull request. Always run the test suite (and applicable coverage checks) prior to sharing your work.

## License
This project is distributed under the [MIT License](LICENSE).
