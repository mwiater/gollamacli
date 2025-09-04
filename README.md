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
      "url": "https://01.your-ollama-server.com"
    },
    {
      "name": "Ollama02",
      "url": "https://02.your-ollama-server.com"
    }
  ],
  "debug": true
}
```

### Configuration Options:

*   `hosts`: A list of Ollama hosts to connect to. Each host object should have:
    *   `name`: A user-friendly name for the host (e.g., "Local Ollama", "Work Server").
    *   `url`: The URL of the Ollama API endpoint (e.g., "http://localhost:11434").
*   `debug`: A boolean value (`true` or `false`) that toggles debug mode. When enabled, performance metrics are displayed after each response.

## Usage

1.  **Run the application:**

    ```bash
    go run chat.go
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

### Keyboard Shortcuts:

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

## Example Usage

```bash
# Run the application
go run chat.go

# Select a host from the list
# Select a model from the list

# Start chatting
Ask Anything: What is the capital of France?

Assistant: The capital of France is Paris.

# Press 'Tab' to go back to the host selection
# Press 'q' to quit
```
