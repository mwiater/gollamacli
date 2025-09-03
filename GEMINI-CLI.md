# Gemini CLI

## Install

Repo: https://github.com/google-gemini/gemini-cli
Global install: `npm install -g @google/gemini-cli@latest`

## Settings

`ls -laF ~/.gemini`

E.g.:

```
-rw-rw-r--  1 matt matt   36 Jul 28 17:36 installation_id
-rw-rw-r--  1 matt matt   42 Jul 28 17:37 settings.json
```


### Checkpointing

REF: https://github.com/google-gemini/gemini-cli/blob/main/docs/checkpointing.md

`nano ~/.gemini/settings.json`

```
{
  "checkpointing": {
    "enabled": true
  }
}
```

## Example

`cd cli`

interactive:

`gemini`

Non-interactive: 

`gemini -p "I want you to examing and understand this self-contained go file: chat.go"`

## Help
`gemini --help`

```
Usage: gemini [options] [command]

Gemini CLI - Launch an interactive CLI, use -p/--prompt for non-interactive mode

Commands:
  gemini      Launch Gemini CLI                                                                                                                                                                                                                                                                                                                                   [default]
  gemini mcp  Manage MCP servers

Options:
  -m, --model                     Model                                                                                                                                                                                                                                                                                                                            [string]
  -p, --prompt                    Prompt. Appended to input on stdin (if any).                                                                                                                                                                                                                                                                                     [string]
  -i, --prompt-interactive        Execute the provided prompt and continue in interactive mode                                                                                                                                                                                                                                                                     [string]
  -s, --sandbox                   Run in sandbox?                                                                                                                                                                                                                                                                                                                 [boolean]
      --sandbox-image             Sandbox image URI.                                                                                                                                                                                                                                                                                                               [string]
  -d, --debug                     Run in debug mode?                                                                                                                                                                                                                                                                                             [boolean] [default: false]
  -a, --all-files                 Include ALL files in context?                                                                                                                                                                                                                                                                                  [boolean] [default: false]
      --all_files                 Include ALL files in context?                                                                                                                                                                                      [deprecated: Use --all-files instead. We will be removing --all_files in the coming weeks.] [boolean] [default: false]
      --show-memory-usage         Show memory usage in status bar                                                                                                                                                                                                                                                                                [boolean] [default: false]
      --show_memory_usage         Show memory usage in status bar                                                                                                                                                                    [deprecated: Use --show-memory-usage instead. We will be removing --show_memory_usage in the coming weeks.] [boolean] [default: false]
  -y, --yolo                      Automatically accept all actions (aka YOLO mode, see https://www.youtube.com/watch?v=xvFZjo5PgG0 for more details)?                                                                                                                                                                                            [boolean] [default: false]
      --approval-mode             Set the approval mode: default (prompt for approval), auto_edit (auto-approve edit tools), yolo (auto-approve all tools)                                                                                                                                                               [string] [choices: "default", "auto_edit", "yolo"]
      --telemetry                 Enable telemetry? This flag specifically controls if telemetry is sent. Other --telemetry-* flags set specific values but do not enable telemetry on their own.                                                                                                                                                                 [boolean]
      --telemetry-target          Set the telemetry target (local or gcp). Overrides settings files.                                                                                                                                                                                                                                     [string] [choices: "local", "gcp"]
      --telemetry-otlp-endpoint   Set the OTLP endpoint for telemetry. Overrides environment variables and settings files.                                                                                                                                                                                                                                         [string]
      --telemetry-otlp-protocol   Set the OTLP protocol for telemetry (grpc or http). Overrides settings files.                                                                                                                                                                                                                          [string] [choices: "grpc", "http"]
      --telemetry-log-prompts     Enable or disable logging of user prompts for telemetry. Overrides settings files.                                                                                                                                                                                                                                              [boolean]
      --telemetry-outfile         Redirect all telemetry output to the specified file.                                                                                                                                                                                                                                                                             [string]
  -c, --checkpointing             Enables checkpointing of file edits                                                                                                                                                                                                                                                                            [boolean] [default: false]
      --experimental-acp          Starts the agent in ACP mode                                                                                                                                                                                                                                                                                                    [boolean]
      --allowed-mcp-server-names  Allowed MCP server names                                                                                                                                                                                                                                                                                                          [array]
  -e, --extensions                A list of extensions to use. If not provided, all extensions are used.                                                                                                                                                                                                                                                            [array]
  -l, --list-extensions           List all available extensions and exit.                                                                                                                                                                                                                                                                                         [boolean]
      --proxy                     Proxy for gemini client, like schema://user:password@host:port                                                                                                                                                                                                                                                                   [string]
      --include-directories       Additional directories to include in the workspace (comma-separated or multiple --include-directories)                                                                                                                                                                                                                            [array]
  -v, --version                   Show version number                                                                                                                                                                                                                                                                                                             [boolean]
  -h, --help                      Show help   
```