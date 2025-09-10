# gollamacli CLI

gollamacli CLI is a terminal-based application for interacting with large language models through the Ollama API. It provides a user-friendly interface for selecting a host, choosing a model, and engaging in a conversation.

## Configuration

Before running the application, you need to create a `config.json` file in the `cli` directory. This file defines the available Ollama hosts and other settings.

### `config.json` format:

```json
{
  "hosts": [
    {
      "name": "Ollama01",
      "url": "http://localhost:11434",
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
      "name": "LMStudio01",
      "url": "http://localhost:1234",
      "type": "lmstudio",
      "models": [
        "stablelm-zephyr:3b",
        "granite3.3:2b",
        "dolphin-phi:2.7b",
        "qwen3:1.7b"
      ]
    }
  ],
  "debug": true
}
```

### Configuration Options:

*   `hosts`: A list of Ollama hosts to connect to. Each host object should have:
    *   `name`: A user-friendly name for the host (e.g., "Local Ollama", "Work Server").
    *   `url`: The URL of the Ollama API endpoint (e.g., "http://localhost:11434").
    *   `type`: The type of host, either "ollama" or "lmstudio".
    *   `models`: A list of models to be managed by the tool for this host.
*   `debug`: A boolean value (`true` or `false`) that toggles debug mode. When enabled, performance metrics are displayed after each response.

## Usage

### Chat

1.  **Run the application:**

    ```bash
    ./bin/gollamacli chat
    ```

2.  **Select a Host:**
    The application will first display a list of the hosts defined in your `config.json` file. Use the arrow keys to navigate and press `Enter` to select a host.

3.  **Select a Model:**
    After selecting a host, the application will fetch and display a list of available models from that host. Models that are already loaded into memory on the host will be indicated with "Currently loaded". Use the arrow keys to navigate and press `Enter` to select a model.

4.  **Chat:**
    Once a model is selected, you will be taken to the chat interface.
    *   Type your message in the input area at the bottom of the screen and press `Enter` to send.
    *   The conversation history is displayed above the input area.
    *   The assistant's responses will be streamed to the screen as they are generated.

### Model Management

The following commands are available for managing models:

*   **List Models:** List all models on each node.
    ```bash
    ./bin/gollamacli list models
    ```

*   **Pull Models:** Pull all models from the `config.json` file to each node.
    ```bash
    ./bin/gollamacli pull models
    ```

*   **Delete Models:** Delete all models not in the `config.json` file from each node.
    ```bash
    ./bin/gollamacli delete models
    ```

*   **Sync Models:** Sync all models from the `config.json` file to each node. This will delete any models not in the `config.json` file and pull any missing models.
    ```bash
    ./bin/gollamacli sync models
    ```

### Keyboard Shortcuts (Chat):

*   `q` or `Ctrl+c`: Quit the application.
*   `Tab`: Return to the host selection screen from the chat view.

## Debug Mode

When `debug` is set to `true` in `config.json`, the following performance metrics will be displayed after each response from the model:

*   **Model Load Duration:** The time it took to load the model into memory.
*   **Prompt Eval:**
    *   **Duration:** The time it took to process the prompt.
    *   **Tokens:** The number of tokens in the prompt.
*   **Response Eval:**
    *   **Duration:** The time it took to generate the response.
    *   **Tokens:** The number of tokens in the response.
*   **Total Duration:** The total time from sending the request to receiving the full response.

A `debug.log` file is also created to log detailed information about the application's execution.

## Build Instructions

### Linux

```bash
GOOS=linux GOARCH=amd64 go build -o bin/gollamacli cmd/main.go
```

### Windows

```bash
GOOS=windows GOARCH=amd64 go build -o bin/gollamacli.exe cmd/main.go
```