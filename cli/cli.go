// cli/cli.go
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mwiater/gollamacli/models"
)

// Host describes a language model host and its configured models.
// It is used by the TUI to display selectable hosts and by the
// network layer to build API requests.
type Host struct {
	// Name is a user-friendly label for the host, for example "Local Ollama".
	Name string `json:"name"`
	// URL is the HTTP endpoint of the host, such as "http://localhost:11434".
	URL string `json:"url"`
	// Models lists the model identifiers that are available or desired on the host.
	Models []string `json:"models"`
}

// Config contains application settings that drive the CLI/TUI behavior.
// It includes the set of available hosts, debug mode, and multimodel mode.
type Config struct {
	// Hosts is the list of language model backends the application can target.
	Hosts []Host `json:"hosts"`
	// Debug enables display of timing metrics and logs additional details.
	Debug bool `json:"debug"`
	// Multimodel toggles the four-column chat interface for multiple models.
	Multimodel bool `json:"multimodel"`
}

// loadConfig reads and parses the configuration file from the given path.
// It returns a Config struct or an error if the file cannot be read or parsed,
// or if no hosts are defined in the configuration.
func loadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config JSON: %w", err)
	}
	if len(cfg.Hosts) == 0 {
		return nil, errors.New("config must contain at least one host")
	}
	return &cfg, nil
}

// LLMResponseMeta holds timing and tokenization metrics for a model response.
// The metadata typically arrives on the final chunk of a streaming response
// and is rendered when debug mode is enabled.
type LLMResponseMeta struct {
	// Model is the name of the model that produced the response.
	Model string `json:"model"`
	// CreatedAt is the time when the response metadata was assembled.
	CreatedAt time.Time `json:"created_at"`
	// Done indicates whether the stream has finished.
	Done bool `json:"done"`
	// TotalDuration is the total request time in nanoseconds.
	TotalDuration int64 `json:"total_duration"`
	// LoadDuration is the model load time in nanoseconds.
	LoadDuration int64 `json:"load_duration"`
	// PromptEvalCount is the number of prompt tokens evaluated.
	PromptEvalCount int `json:"prompt_eval_count"`
	// PromptEvalDuration is the prompt evaluation time in nanoseconds.
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	// EvalCount is the number of tokens generated during response.
	EvalCount int `json:"eval_count"`
	// EvalDuration is the response evaluation time in nanoseconds.
	EvalDuration int64 `json:"eval_duration"`
}

// chatMessage represents a single message in a chat conversation,
// including the role of the sender (e.g., "user", "assistant") and the content of the message.
type chatMessage struct {
	// Role of the message sender (e.g., "user", "assistant").
	Role string `json:"role"`
	// Content of the message.
	Content string `json:"content"`
}

// streamChunk represents a single chunk of a streaming language model response.
// It includes partial message content and updated metadata.
type streamChunk struct {
	// Name of the language model.
	Model   string `json:"model"`
	Message struct {
		// Role of the message sender within this chunk.
		Role string `json:"role"`
		// Partial content of the message.
		Content string `json:"content"`
	} `json:"message"`
	// Indicates if this is the final chunk of the stream.
	Done bool `json:"done"`
	// Total duration up to this chunk in nanoseconds.
	TotalDuration int64 `json:"total_duration"`
	// Duration spent loading the model up to this chunk in nanoseconds.
	LoadDuration int64 `json:"load_duration"`
	// Number of tokens in the prompt evaluated up to this chunk.
	PromptEvalCount int `json:"prompt_eval_count"`
	// Duration spent evaluating the prompt up to this chunk in nanoseconds.
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	// Number of tokens in the response evaluated up to this chunk.
	EvalCount int `json:"eval_count"`
	// Duration spent evaluating the response up to this chunk in nanoseconds.
	EvalDuration int64 `json:"eval_duration"`
}

// ollamaTagsResponse represents the structure of the response from the Ollama /api/tags endpoint,
// which lists available models.
type ollamaTagsResponse struct {
	// List of models.
	Models []struct {
		// Name of the model.
		Name string `json:"name"`
	} `json:"models"`
}

// ollamaPsResponse represents the structure of the response from the Ollama /api/ps endpoint,
// which lists currently loaded models.
type ollamaPsResponse struct {
	// List of currently loaded models.
	Models []struct {
		// Name of the loaded model.
		Name string `json:"name"`
	} `json:"models"`
}

// viewState represents the current state of the application's view.
type viewState int

const (
	// viewHostSelector is the state where the user selects a host.
	viewHostSelector viewState = iota
	// viewModelSelector is the state where the user selects a model.
	viewModelSelector
	// viewLoadingChat is the state while a model is being loaded for chat.
	viewLoadingChat
	// viewChat is the state where the user is interacting with the chat.
	viewChat
)

// model is the main application model for the Bubble Tea UI.
// It holds all the necessary state for the chat application.
type model struct {
	// Application configuration.
	config *Config
	// HTTP client for API requests.
	client *http.Client
	// Current view state of the application.
	state viewState
	// Indicates if an asynchronous operation is in progress.
	isLoading bool
	// Stores any error encountered during operations.
	err error

	// Bubble Tea list model for host selection.
	hostList list.Model
	// Bubble Tea list model for model selection.
	modelList list.Model
	// Bubble Tea textarea model for user input.
	textArea textarea.Model
	// Bubble Tea viewport model for displaying chat history.
	viewport viewport.Model
	// Bubble Tea spinner model for indicating loading.
	spinner spinner.Model
	// Stores the history of chat messages.
	chatHistory []chatMessage
	// Buffer to accumulate streaming responses.
	responseBuf strings.Builder
	// Metadata of the last language model response.
	responseMeta LLMResponseMeta
	// The currently selected host.
	selectedHost Host
	// The currently selected model.
	selectedModel string
	// List of models currently loaded on the selected host.
	loadedModels []string

	// Current width and height of the terminal.
	width, height int
	// Reference to the Bubble Tea program.
	program *tea.Program
	// Timestamp when the last request started.
	requestStartTime time.Time
}

// initialModel initializes a new model with default values and sets up
// the necessary Bubble Tea components like spinner, textarea, and lists.
func initialModel(cfg *Config) *model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()
	ta.Prompt = "Ask Anything: "
	ta.ShowLineNumbers = false
	ta.CharLimit = -1
	ta.SetHeight(1)
	ta.KeyMap.InsertNewline.SetEnabled(false)

	hostItems := make([]list.Item, len(cfg.Hosts))
	for i, h := range cfg.Hosts {
		hostItems[i] = item{title: h.Name, desc: h.URL}
	}
	hostDelegate := list.NewDefaultDelegate()
	hostList := list.New(hostItems, hostDelegate, 0, 0)
	hostList.Title = "Select a Host"

	vp := viewport.New(100, 5)

	return &model{
		config: cfg,
		client: &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2: false,
			},
		},
		state:     viewHostSelector,
		spinner:   s,
		textArea:  ta,
		hostList:  hostList,
		modelList: list.New(nil, list.NewDefaultDelegate(), 0, 0),
		viewport:  vp,
	}
}

// item represents a selectable item in a Bubble Tea list,
// used for both hosts and models.
type item struct {
	// The main title of the item.
	title string
	// A short description of the item.
	desc string
	// Indicates if the model is currently loaded (only applicable for model items).
	loaded bool
}

// Title returns the title of the list item.
func (i item) Title() string { return i.title }

// Description returns the description of the list item.
// If the item represents a loaded model, it returns "Currently loaded".
func (i item) Description() string {
	if i.loaded {
		return "Currently loaded"
	}
	return i.desc
}

// FilterValue returns the title of the item, used for filtering in the list.
func (i item) FilterValue() string { return i.title }

// modelsReadyMsg is sent when the model list is fetched and processed.
type modelsReadyMsg struct {
	models       []list.Item
	loadedModels []string
}

// modelsLoadErr is sent when an error occurs while fetching models.
type modelsLoadErr error

// chatReadyMsg is sent when the chat interface is ready for interaction.
type chatReadyMsg struct{}

// chatReadyErr is sent when an error occurs while preparing the chat interface.
type chatReadyErr error

// streamChunkMsg is sent when a new chunk of a streaming response is received.
type streamChunkMsg string

// streamEndMsg is sent when a streaming response has completed.
type streamEndMsg struct{ meta LLMResponseMeta }

// streamErr is sent when an error occurs during a streaming response.
type streamErr error

// tickMsg is a regular tick message used for animations or timed updates.
type tickMsg time.Time

// fetchAndSelectModelsCmd fetches loaded models, then all models,
// and prepares the model list for selection. It prioritizes loaded models
// by placing them at the top of the list.
func fetchAndSelectModelsCmd(host Host, client *http.Client) tea.Cmd {
	return func() tea.Msg {
		loadedModels, err := getLoadedModels(host, client)
		if err != nil {
			return modelsLoadErr(err)
		}

		allModels := host.Models

		loadedModelSet := make(map[string]struct{})
		for _, m := range loadedModels {
			loadedModelSet[m] = struct{}{}
		}

		var loadedItems []list.Item
		var otherItems []list.Item
		for _, m := range allModels {
			_, isLoaded := loadedModelSet[m]
			listItem := item{title: m, desc: "Select this model", loaded: isLoaded}
			if isLoaded {
				loadedItems = append(loadedItems, listItem)
			} else {
				otherItems = append(otherItems, listItem)
			}
		}

		finalModelItems := append(loadedItems, otherItems...)

		return modelsReadyMsg{
			models:       finalModelItems,
			loadedModels: loadedModels,
		}
	}
}

// getLoadedModels fetches the names of currently loaded models from the /api/ps endpoint.
func getLoadedModels(host Host, client *http.Client) ([]string, error) {
	req, err := http.NewRequest("GET", host.URL+"/api/ps", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-200 status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var psResp ollamaPsResponse
	if err := json.Unmarshal(body, &psResp); err != nil {
		return nil, err
	}

	loadedModels := make([]string, len(psResp.Models))
	for i, m := range psResp.Models {
		loadedModels[i] = m.Name
	}
	return loadedModels, nil
}

// loadModelCmd is a Bubble Tea command that attempts to load a specified model
// on the given host by sending a minimal generate request to /api/generate.
// This is typically used to ensure a model is ready for chat.
// It returns a tea.Msg indicating success (chatReadyMsg) or failure (chatReadyErr).
func loadModelCmd(host Host, modelName string, client *http.Client) tea.Cmd {
	return func() tea.Msg {
		payload := map[string]any{
			"model":  modelName,
			"prompt": ".",
			"stream": false,
		}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(context.Background(), "POST", host.URL+"/api/generate", bytes.NewReader(body))
		if err != nil {
			return chatReadyErr(err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return chatReadyErr(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return chatReadyErr(fmt.Errorf("API returned non-200 status: %s. Body: %s", resp.Status, string(bodyBytes)))
		}

		return chatReadyMsg{}
	}
}

// streamChatCmd is a Bubble Tea command that initiates a streaming chat conversation
// with the selected language model. It sends the chat history and streams back
// responses chunk by chunk.
// It sends streamChunkMsg for each new chunk and streamEndMsg when the stream completes.
// Errors during streaming result in a streamErr message.
func streamChatCmd(p *tea.Program, host Host, modelName string, history []chatMessage, client *http.Client) tea.Cmd {
	return func() tea.Msg {
		payload := map[string]any{
			"model":    modelName,
			"messages": history,
			"stream":   true,
		}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(context.Background(), "POST", host.URL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			return streamErr(err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return streamErr(err)
		}

		go func() {
			defer resp.Body.Close()
			decoder := json.NewDecoder(resp.Body)
			var finalChunk streamChunk
			for {
				var chunk streamChunk
				if err := decoder.Decode(&chunk); err != nil {
					if err != io.EOF {
						p.Send(streamErr(err))
					}
					break
				}
				p.Send(streamChunkMsg(chunk.Message.Content))
				if chunk.Done {
					finalChunk = chunk
					break
				}
			}
			p.Send(streamEndMsg{meta: LLMResponseMeta{
				Model:              finalChunk.Model,
				CreatedAt:          time.Now(),
				Done:               finalChunk.Done,
				TotalDuration:      finalChunk.TotalDuration,
				LoadDuration:       finalChunk.LoadDuration,
				PromptEvalCount:    finalChunk.PromptEvalCount,
				PromptEvalDuration: finalChunk.PromptEvalDuration,
				EvalCount:          finalChunk.EvalCount,
				EvalDuration:       finalChunk.EvalDuration,
			}})
		}()

		return nil
	}
}

// tickCmd returns a Bubble Tea command that sends a tickMsg at a regular interval.
// This is used for animations or periodic UI updates.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Init initializes the Bubble Tea model. It returns a command to start the spinner animation.
func (m *model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update is the central update function for the Bubble Tea model.
// It handles incoming messages and updates the application's state accordingly.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.state == viewChat {
				m.state = viewHostSelector
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.hostList.SetSize(msg.Width-2, msg.Height-4)
		m.modelList.SetSize(msg.Width-2, msg.Height-4)
		m.textArea.SetWidth(msg.Width - 3)
		headerHeight := 4
		footerHeight := 5
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - footerHeight

	case chatReadyMsg:
		m.isLoading = false
		m.state = viewChat
		m.textArea.Focus()
		m.viewport.GotoBottom()
		return m, nil

	case chatReadyErr:
		m.isLoading = false
		m.err = msg
		return m, nil

	case modelsReadyMsg:
		m.isLoading = false
		m.modelList.SetItems(msg.models)
		m.loadedModels = msg.loadedModels
		m.modelList.Title = fmt.Sprintf("Select a Model from %s", m.selectedHost.Name)
		m.state = viewModelSelector
		if len(m.loadedModels) > 0 {
			m.modelList.Select(0)
		}
		return m, nil

	case modelsLoadErr:
		m.isLoading = false
		m.err = msg
		return m, nil

	case streamChunkMsg:
		m.responseBuf.WriteString(string(msg))
		m.viewport.GotoBottom()
		return m, nil

	case streamEndMsg:
		m.responseMeta = msg.meta
		if m.responseBuf.Len() > 0 {
			m.chatHistory = append(m.chatHistory, chatMessage{
				Role:    "assistant",
				Content: m.responseBuf.String(),
			})
			m.responseBuf.Reset()
		}
		m.isLoading = false
		m.textArea.Focus()
		m.viewport.GotoBottom()
		return m, nil

	case streamErr:
		m.isLoading = false
		m.err = msg
		return m, nil
	case tickMsg:
		if m.isLoading {
			return m, tickCmd()
		}
		return m, nil
	}

	switch m.state {
	case viewHostSelector:
		m.hostList, cmd = m.hostList.Update(msg)
		cmds = append(cmds, cmd)
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
			if _, ok := m.hostList.SelectedItem().(item); ok {
				m.selectedHost = m.config.Hosts[m.hostList.Index()]
				m.isLoading = true
				m.requestStartTime = time.Now()
				m.err = nil
				cmds = append(cmds, m.spinner.Tick, fetchAndSelectModelsCmd(m.selectedHost, m.client), tickCmd())
			}
		}

	case viewModelSelector:
		m.modelList, cmd = m.modelList.Update(msg)
		cmds = append(cmds, cmd)
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
			if selectedItem, ok := m.modelList.SelectedItem().(item); ok {
				m.selectedModel = selectedItem.Title()
				m.state = viewLoadingChat
				m.isLoading = true
				m.requestStartTime = time.Now()
				m.err = nil
				cmds = append(cmds, m.spinner.Tick, loadModelCmd(m.selectedHost, m.selectedModel, m.client), tickCmd())
			}
		}

	case viewChat:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

		m.textArea, cmd = m.textArea.Update(msg)
		cmds = append(cmds, cmd)

		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
			userInput := strings.TrimSpace(m.textArea.Value())
			if userInput != "" {
				m.responseMeta = LLMResponseMeta{}
				m.requestStartTime = time.Now()
				m.chatHistory = append(m.chatHistory, chatMessage{Role: "user", Content: userInput})
				m.textArea.Reset()
				m.isLoading = true
				m.err = nil
				cmds = append(cmds, m.spinner.Tick, streamChatCmd(m.program, m.selectedHost, m.selectedModel, m.chatHistory, m.client))
			}
		}
	}

	if m.isLoading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the application's UI based on its current state.
func (m *model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(1)
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	switch m.state {
	case viewHostSelector, viewModelSelector:
		var listModel list.Model
		if m.state == viewHostSelector {
			listModel = m.hostList
		} else {
			listModel = m.modelList
		}
		if m.isLoading {
			timer := fmt.Sprintf("%.1f", time.Since(m.requestStartTime).Seconds())
			return fmt.Sprintf("\n  %s Fetching models... %ss\n", m.spinner.View(), timer)
		}
		return lipgloss.NewStyle().Margin(1, 2).Render(listModel.View())

	case viewLoadingChat:
		timer := fmt.Sprintf("%.1f", time.Since(m.requestStartTime).Seconds())
		return fmt.Sprintf("\n  %s Loading %s... %ss\n", m.spinner.View(), m.selectedModel, timer)

	case viewChat:
		return m.chatView()

	default:
		return "Unknown state"
	}
}

// chatView renders the chat interface, including the header, chat history,
// current response (if streaming), and the input text area.
func (m *model) chatView() string {
	var builder strings.Builder

	headerStyle := lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Padding(0, 1)
	hostInfo := fmt.Sprintf("Host: %s", m.selectedHost.Name)
	modelInfo := fmt.Sprintf("Model: %s", m.selectedModel)
	status := lipgloss.JoinHorizontal(lipgloss.Top,
		headerStyle.Render(hostInfo),
		headerStyle.MarginLeft(1).Render(modelInfo),
	)
	help := lipgloss.NewStyle().Faint(true).Render(" (tab to change, q to quit)")
	builder.WriteString(status + help + "\n\n")

	var historyBuilder strings.Builder
	userStyle := lipgloss.NewStyle().Bold(true)
	assistantStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))

	for _, msg := range m.chatHistory {
		var role, content string
		if msg.Role == "assistant" {
			role = assistantStyle.Render("Assistant: ")
			content = msg.Content
		} else {
			role = userStyle.Render("You: ")
			content = msg.Content
		}
		wrappedContent := lipgloss.NewStyle().Width(m.width - lipgloss.Width(role) - 2).Render(content)
		historyBuilder.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, role, wrappedContent) + "\n")
	}

	if m.responseBuf.Len() > 0 {
		role := assistantStyle.Render("Assistant: ")
		wrappedContent := lipgloss.NewStyle().Width(m.width - lipgloss.Width(role) - 2).Render(m.responseBuf.String())
		historyBuilder.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, role, wrappedContent))
	}

	m.viewport.SetContent(historyBuilder.String())
	builder.WriteString(m.viewport.View())

	if m.isLoading {
		timer := fmt.Sprintf("%.1f", time.Since(m.requestStartTime).Seconds())
		loadingText := fmt.Sprintf(" Assistant is thinking... %ss", timer)
		builder.WriteString("\n" + m.spinner.View() + loadingText)
	} else {
		builder.WriteString("\n" + m.textArea.View())
	}

	if m.config.Debug && m.responseMeta.Done {
		builder.WriteString("\n" + formatMeta(m.responseMeta))
	}

	return builder.String()
}

// formatMeta formats the LLMResponseMeta into a human-readable string,
// displaying various performance metrics of the language model response.
func formatMeta(meta LLMResponseMeta) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	loadDur := float64(meta.LoadDuration) / 1e9
	promptEvalDur := float64(meta.PromptEvalDuration) / 1e9
	evalDur := float64(meta.EvalDuration) / 1e9
	totalDur := float64(meta.TotalDuration) / 1e9

	return style.Render(fmt.Sprintf(
		"  >>> [Model Load Duration: %.1fs] [Prompt Eval: %.1fs | %d Tokens] [Response Eval: %.1fs | %d Tokens] [Total Duration: %.1fs]",
		loadDur,
		promptEvalDur,
		meta.PromptEvalCount,
		evalDur,
		meta.EvalCount,
		totalDur,
	))
}

// StartGUI initializes and runs the interactive TUI for single-model chat.
// It reads configuration from config.json, optionally switches to multimodel
// mode, and blocks until the UI exits. It logs diagnostic output to debug.log
// when enabled. StartGUI does not return a value.
func StartGUI() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatalf("could not open log file: %v", err)
	}
	defer f.Close()

	cfg, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	if cfg.Multimodel {
		models.UnloadModels()
		if err := StartMultimodelGUI(cfg); err != nil {
			log.Fatalf("Error running multimodel program: %v", err)
		}
		return
	}

	m := initialModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.program = p

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
