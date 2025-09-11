# gollamacli CLI

gollamacli CLI is a powerful, terminal-based application designed for seamless interaction with large language models through the Ollama API. It offers a rich set of features to streamline your workflow, whether you're managing a single local model or a distributed network of language model hosts.

Key Features:

*   **Multiple Host Management:** Connect to and switch between multiple Ollama and LM Studio hosts defined in a simple `config.json` file.
*   **Interactive Chat:** Engage in conversations with your chosen language model through a user-friendly terminal interface.
*   **Model Synchronization:** Keep your models consistent across all your hosts with a single command. The `sync` feature will automatically pull missing models and delete any models that are not defined in your configuration file.
*   **Efficient Model Management:** Easily list, pull, and delete models on your hosts.
*   **Debug Mode:** Gain insights into performance with detailed metrics for model loading, prompt evaluation, and response generation.

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

## Build Instructions

To build the project using GoReleaser, run the following command:

```bash
goreleaser release --snapshot --clean --skip archive
```

This will create a snapshot release, which is a test release that doesn't create a Git tag or release on GitHub. The `--clean` flag will remove the `dist` directory before building, and the `--skip-archive` flag will prevent the creation of an archive file (e.g., `.tar.gz` or `.zip`).

## Usage

After building the project with GoReleaser, you will find the executables in the `dist` directory. The path to the executable will vary depending on your operating system and architecture. For example, on Linux with an AMD64 architecture, the executable will be at `dist/gollamacli_linux_amd64/gollamacli`.

### Chat

1.  **Run the application:**

    ```bash
    ./dist/gollamacli_linux_amd64/gollamacli chat
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
    ./dist/gollamacli_linux_amd64/gollamacli list models
    ```

*   **Pull Models:** Pull all models from the `config.json` file to each node.
    ```bash
    ./dist/gollamacli_linux_amd64/gollamacli pull models
    ```

*   **Delete Models:** Delete all models not in the `config.json` file from each node.
    ```bash
    ./dist/gollamacli_linux_amd64/gollamacli delete models
    ```

*   **Sync Models:** Sync all models from the `config.json` file to each node. This will delete any models not in the `config.json` file and pull any missing models.
    ```bash
    ./dist/gollamacli_linux_amd64/gollamacli sync models
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