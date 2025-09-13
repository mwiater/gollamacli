# gollamacli

gollamacli is a powerful, terminal-based application designed for seamless interaction with large language models through the Ollama API. It offers a rich set of features to streamline your workflow, whether you're managing a single local model or a distributed network of language model hosts.

## Key Features

- **Multiple Host Management:** Connect to and switch between multiple hosts defined in a simple `config.json` file (currently supports Ollama).
- **Interactive Chat:** Engage in conversations with your chosen language model through a user-friendly terminal interface.
- **Multimodel Chat:** Chat with multiple models from different hosts simultaneously in a single interface.
- **Model Synchronization:** Keep your models consistent across all your hosts with a single command. The `sync` feature will automatically pull missing models and delete any models that are not defined in your configuration file.
- **Efficient Model Management:** Easily list, pull, delete, and unload models on your hosts.
- **Debug Mode:** Gain insights into performance with detailed metrics for model loading, prompt evaluation, and response generation.

## Installation

To install gollamacli, you can use `go install`:

```bash
go install github.com/mwiater/gollamacli/cmd/gollamacli@latest
```

## Configuration

Before running the application, you need to create a `config.json` file in the same directory as the `gollamacli` executable. This file defines the available hosts and other settings.

### `config.json` format

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
      "url": "http://192.168.0.11:11434",
      "type": "ollama",
      "models": [
        "stablelm-zephyr:3b",
        "granite3.3:2b",
        "gemma3n:e2b"
      ]
    }
  ],
  "debug": true,
  "multimodel": false
}
```

### Configuration Options

- `hosts`: A list of hosts to connect to. Each host object includes (currently only `ollama` is supported):
  - `name`: A user-friendly name for the host (e.g., "Local Ollama", "Work Server").
  - `url`: The URL of the API endpoint (e.g., `http://localhost:11434`).
  - `type`: The type of host (e.g., `ollama`).
  - `models`: A list of models to be managed by the tool for this host.
- `debug`: A boolean value (`true` or `false`) that toggles debug mode. When enabled, performance metrics are displayed after each response.
- `multimodel`: A boolean value (`true` or `false`) that toggles multimodel chat mode.

## Usage

### Chat

To start a chat session, run the `chat` command:

```bash
gollamacli chat
```

If `multimodel` is set to `false` in your `config.json`, you will be prompted to select a host and then a model to chat with.

If `multimodel` is set to `true`, you will enter the multimodel chat interface, where you can assign models to different columns and chat with them simultaneously.

### Model Management

The following commands are available for managing models:

- **List Models**
  ```bash
  gollamacli list models
  ```
- **Pull Models**
  ```bash
  gollamacli pull models
  ```
- **Delete Models**
  ```bash
  gollamacli delete models
  ```
- **Sync Models**
  ```bash
  gollamacli sync models
  ```
- **Unload Models**
  ```bash
  gollamacli unload models
  ```

### Keyboard Shortcuts (Chat)

- `q` or `Ctrl+c`: Quit the application.
- `Tab`: Return to the host/model selection screen from the chat view.

## Build Instructions

To build the project from source, you can use GoReleaser. Run the following command to create a snapshot release:

```bash
goreleaser release --snapshot --clean --skip archive
```

This will create a snapshot release, which is a test release that doesn't create a Git tag or release on GitHub. The `--clean` flag will remove the `dist` directory before building, and the `--skip-archive` flag will prevent the creation of an archive file (e.g., `.tar.gz` or `.zip`). The executables will be located in the `dist` directory.

## Debug Mode

When `debug` is set to `true` in `config.json`, the following performance metrics will be displayed after each response from the model:

- **Model Load Duration:** The time it took to load the model into memory.
- **Prompt Eval:**
  - **Duration:** The time it took to process the prompt.
  - **Tokens:** The number of tokens in the prompt.
- **Response Eval:**
  - **Duration:** The time it took to generate the response.
  - **Tokens:** The number of tokens in the response.
- **Total Duration:** The total time from sending the request to receiving the full response.

A `debug.log` file is also created to log detailed information about the application's execution.

## Godoc

Install:

```bash
go install -v golang.org/x/tools/cmd/godoc@latest
```

Run:

```bash
godoc -http=:6060
```

## Tests

Run the test suite:

```bash
go test ./...
```

## Coverage

Generate a coverage report:

```bash
mkdir -p .coverage
go test ./... -coverprofile=.coverage/coverage.out
go tool cover -func=.coverage/coverage.out
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change. Ensure that tests run successfully before submitting your contribution.

## License

This project is licensed under the [MIT License](LICENSE).
