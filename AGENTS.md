**AGENTS.md**

- Purpose: Define a comprehensive, actionable test plan for full coverage.
- Scope: Unit and integration-style tests using Go’s testing, `httptest`, and light Bubble Tea model testing. No network calls; everything stubbed/mocked.
- Conventions: Table-driven tests where practical; `*_test.go` per package; `t.Parallel()` when safe; isolate stdout/stderr where commands print.

**Models Package Tests (models/)**

- LoadConfig: Success, missing file, malformed JSON.
- CreateHosts: Builds correct host types; unknown type is logged and ignored.
- PullModels: Ollama posts to `/api/pull` per model; LM Studio skipped; concurrency without races; logs errors and continues.
- `OllamaHost.PullModel`: Sends JSON `{name:model}` with header.
- DeleteModels: Keeps configured, deletes extras by calling `DeleteModel`; handles formatted list entries; LM Studio skipped.
- `OllamaHost.DeleteModel`: DELETE `/api/delete` with JSON body and header.
- UnloadModels: Queries `/api/ps`, unloads via `/api/chat` with `keep_alive:0`; LM Studio skipped; errors logged but continue.
- `OllamaHost.UnloadModel`: POST `/api/chat` with `{model, keep_alive:0}`.
- SyncModels: Calls DeleteModels then PullModels (verify call order via spies/log capture).
- ListModels (Ollama): Calls `/api/tags` + `/api/ps`; marks loaded; handles ps error; non-200 returns error.
- ListModels (LM Studio): Calls `/api/v0/models`; non-200 and malformed JSON return errors.
- getRunningModels (Ollama): Parses `/api/ps`; network errors bubble.

**CLI TUI (Single-Model) Tests (cli/cli.go)**

- loadConfig: Valid config OK; no hosts error; malformed JSON error.
- initialModel: Spinner, textarea, lists, viewport, HTTP client, and default state initialized.
- WindowSizeMsg: Sizes for lists, textarea width, viewport height set correctly.
- fetchAndSelectModelsCmd: Prioritizes loaded models at top; returns `modelsReadyMsg` with correct items/order.
- getLoadedModels: Success parses `/api/ps`; non-200 and network error paths return errors.
- loadModelCmd: POST `/api/generate` with `{model, ".", stream:false}`; non-200 returns error with body; success returns `chatReadyMsg`.
- streamChatCmd: Streams multiple JSON objects; emits `streamChunkMsg` then `streamEndMsg` with meta; network error emits `streamErr`.
- tickCmd: Produces ticks at expected cadence (assert receipt; tolerate timing).
- Update loop (state transitions):
  - Host selection enter: loading spinner + `fetchAndSelectModelsCmd`.
  - Model selection enter: loading spinner + `loadModelCmd`.
  - Chat enter with input: appends user message, resets textarea, starts `streamChatCmd`.
  - `streamChunkMsg`: appends to response buffer; viewport bottomed.
  - `streamEndMsg`: moves buffer to assistant message; records meta; clears loading.
  - `streamErr`/`modelsLoadErr`/`chatReadyErr`: sets `err`, stops loading.
  - `tickMsg`: schedules next tick when loading, noop otherwise.
  - Keys: `q/ctrl+c` quits; `tab` returns to selection from chat.
- View rendering: selector loading shows “Fetching models…”, chat loading shows “Loading <model>…”, chat view header includes host/model; streaming shows “Assistant is thinking…”; with Debug, `formatMeta` shown.
- formatMeta: Formats durations (seconds) and token counts in expected string.

**CLI TUI (Multimodel) Tests (cli/cli_multimodel.go)**

- initialMultimodelModel: Creates assignments per host, empty column responses, UI components.
- Assignment interactions (updateAssignment): up/down navigation; enter opens model selection; selecting model sets `selectedModel` and `isAssigned`; `esc` cancels; `c` starts chat only if at least one assignment, sets loading, resets buffers, stamps start times.
- loadMultimodelChatCmd: Returns `multimodelChatReadyMsg` after delay (control with timeout or injectable clock).
- multimodelStreamChatCmd: Starts one goroutine per assigned column; unassigned columns don’t start; per-column errors send `multimodelStreamErr` with index.
- streamToColumn: Posts `/api/chat` with `{model, messages, stream:true}`; emits chunk and end messages on JSON stream.
- Update (chat mode):
  - Enter with input: appends user message to each assigned column; sets `isStreaming=true`; stamps per-column start; clears buffers/errors; global `isLoading=true`.
  - `multimodelStreamChunkMsg`: appends to last assistant message or starts new; marks column streaming.
  - `multimodelStreamEndMsg`: writes meta and clears column streaming; when all assigned columns done, clears global loading, focuses and resets textarea.
  - Keys: `tab` back to assignment; `q/ctrl+c` quits.
- multimodelChatView: Renders 4 columns; headers show host/model or “Empty”; dynamic chat area height; while streaming, shows per-column loading indicators with elapsed.

**Cobra CLI Wiring Tests (cmd/gollamacli/)**

- Root structure: `gollamacli` contains `chat`, `list`, `pull`, `delete`, `sync`, `unload`.
- Group commands: `list` has `commands` and `models`; `pull|delete|sync|unload` each include `models`.
- Descriptions: `Short`/`Long` non-empty for each command.
- list commands: Running prints hierarchical paths (e.g., “gollamacli list models”) with two-column spacing.
- Command invocations: `list models`, `pull models`, `delete models`, `sync models`, `unload models` run without panic and produce output; stub underlying calls where needed.

**Config & Fixtures**

- `config.json.example` parses into both `cli.Config` and `models.Config`; includes at least one host, valid types; toggling `multimodel` works.

**Error Conditions & Edge Cases**

- Non-200 with body: `loadModelCmd` error includes status and body.
- Streaming disconnect mid-stream: emits `streamErr` and cleans up.
- JSON decode errors: `ps`, `tags`, `v0/models`, chat stream yield clear errors.
- Empty lists: Host with zero models behaves; selection view handles empty list gracefully.
- Sorting and stability: `ListModels` prints nodes sorted regardless of goroutine timing.
- ANSI/styling: `ListModels` dash removal and trimming don’t corrupt names.

**Test Utilities To Implement**

- HTTP test servers: stubs for `/api/tags`, `/api/ps`, `/api/generate`, `/api/chat` (stream), `/api/v0/models`.
- Output capture helpers: capture/assert stdout/stderr for Cobra commands.
- Time helpers: clock abstraction or tolerant assertions for elapsed text.
- Bubble Tea harness: helpers to instantiate models, send `tea.Msg`, and assert state and `View()` substrings without running the full program.

**Coverage Goals**

- Lines: ≥90% for `models`, `cli`, and `cmd/gollamacli`.
- Branches: Exercise error paths for network/JSON/state transitions.
- Concurrency: Guard fan-out in `PullModels` and multimodel streaming with at least one race-checked test (use `-race`).

**Execution Plan**

- Phase 1: models package tests (HTTP + logic).
- Phase 2: cli single-model tests (config, commands, Update/View) with fake servers.
- Phase 3: cli multimodel tests (assignment, streaming, View) with fake servers.
- Phase 4: Cobra wiring and command tree output.
- Phase 5: Edge cases and error-path drills.
- Phase 6: Race detector pass and coverage review.

**Notes**

- Avoid launching full TUIs; test Bubble Tea models by posting messages and inspecting state/view.
- Keep servers deterministic; stream JSON as consecutive objects to match `json.Decoder`.
- When spies aren’t possible, assert observable effects: server hit counts, printed output, model state changes.

