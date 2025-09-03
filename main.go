// Package main provides a command-line interface (CLI) for interacting with language models.
// It allows users to select a host and a model, engage in chat, and view model responses.
package main

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
)

// Host represents a language model host with a name and URL.
type Host struct {
	Name string `json:"name"` // Name of the host, e.g., "Local Ollama"
	URL  string `json:"url"`  // URL of the host, e.g., "http://localhost:11434"
}

// Config holds the application's configuration, including a list of language model hosts
// and a debug flag.
type Config struct {
	Hosts []Host `json:"hosts"` // List of available language model hosts.
	Debug bool   `json:"debug"` // Debug flag; if true, additional debug information is displayed.
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

// LLMResponseMeta contains metadata about a language model's response,
// including performance metrics and model details.
type LLMResponseMeta struct {
	Model              string    `json:"model"`                // Name of the language model used.
	CreatedAt          time.Time `json:"created_at"`           // Timestamp when the response was created.
	Done               bool      `json:"done"`                 // Indicates if the response stream is complete.
	TotalDuration      int64     `json:"total_duration"`       // Total duration of the request in nanoseconds.
	LoadDuration       int64     `json:"load_duration"`        // Duration spent loading the model in nanoseconds.
	PromptEvalCount    int       `json:"prompt_eval_count"`    // Number of tokens in the prompt evaluated.
	PromptEvalDuration int64     `json:"prompt_eval_duration"` // Duration spent evaluating the prompt in nanoseconds.
	EvalCount          int       `json:"eval_count"`           // Number of tokens in the response evaluated.
	EvalDuration       int64     `json:"eval_duration"`        // Duration spent evaluating the response in nanoseconds.
}

// chatMessage represents a single message in a chat conversation,
// including the role of the sender (e.g., "user", "assistant") and the content of the message.
type chatMessage struct {
	Role    string `json:"role"`    // Role of the message sender (e.g., "user", "assistant").
	Content string `json:"content"` // Content of the message.
}

// streamChunk represents a single chunk of a streaming language model response.
// It includes partial message content and updated metadata.
type streamChunk struct {
	Model   string `json:"model"` // Name of the language model.
	Message struct {
		Role    string `json:"role"`    // Role of the message sender within this chunk.
		Content string `json:"content"` // Partial content of the message.
	} `json:"message"`
	Done               bool  `json:"done"`                 // Indicates if this is the final chunk of the stream.
	TotalDuration      int64 `json:"total_duration"`       // Total duration up to this chunk in nanoseconds.
	LoadDuration       int64 `json:"load_duration"`        // Duration spent loading the model up to this chunk in nanoseconds.
	PromptEvalCount    int   `json:"prompt_eval_count"`    // Number of tokens in the prompt evaluated up to this chunk.
	PromptEvalDuration int64 `json:"prompt_eval_duration"` // Duration spent evaluating the prompt up to this chunk in nanoseconds.
	EvalCount          int   `json:"eval_count"`           // Number of tokens in the response evaluated up to this chunk.
	EvalDuration       int64 `json:"eval_duration"`        // Duration spent evaluating the response up to this chunk in nanoseconds.
}

// ollamaTagsResponse represents the structure of the response from the Ollama /api/tags endpoint,
// which lists available models.
type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"` // Name of the model.
	} `json:"models"` // List of models.
}

// ollamaPsResponse represents the structure of the response from the Ollama /api/ps endpoint,
// which lists currently loaded models.
type ollamaPsResponse struct {
	Models []struct {
		Name string `json:"name"` // Name of the loaded model.
	} `json:"models"` // List of currently loaded models.
}

// viewState represents the current state of the application's view.
type viewState int

const (
	viewHostSelector  viewState = iota // viewHostSelector is the state where the user selects a host.
	viewModelSelector                  // viewModelSelector is the state where the user selects a model.
	viewLoadingChat                    // viewLoadingChat is the state while a model is being loaded for chat.
	viewChat                           // viewChat is the state where the user is interacting with the chat.
)

// model is the main application model for the Bubble Tea UI.
// It holds all the necessary state for the chat application.
type model struct {
	config    *Config      // Application configuration.
	client    *http.Client // HTTP client for API requests.
	state     viewState    // Current view state of the application.
	isLoading bool         // Indicates if an asynchronous operation is in progress.
	err       error        // Stores any error encountered during operations.

	hostList      list.Model      // Bubble Tea list model for host selection.
	modelList     list.Model      // Bubble Tea list model for model selection.
	textArea      textarea.Model  // Bubble Tea textarea model for user input.
	viewport      viewport.Model  // Bubble Tea viewport model for displaying chat history.
	spinner       spinner.Model   // Bubble Tea spinner model for indicating loading.
	chatHistory   []chatMessage   // Stores the history of chat messages.
	responseBuf   strings.Builder // Buffer to accumulate streaming responses.
	responseMeta  LLMResponseMeta // Metadata of the last language model response.
	selectedHost  Host            // The currently selected host.
	selectedModel string          // The currently selected model.
	loadedModels  []string        // List of models currently loaded on the selected host.

	width, height    int          // Current width and height of the terminal.
	program          *tea.Program // Reference to the Bubble Tea program.
	requestStartTime time.Time    // Timestamp when the last request started.
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
	title  string // The main title of the item.
	desc   string // A short description of the item.
	loaded bool   // Indicates if the model is currently loaded (only applicable for model items).
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
		// First, get the list of currently loaded models from /api/ps
		loadedModels, err := getLoadedModels(host, client)
		if err != nil {
			return modelsLoadErr(err)
		}

		// Then, get the list of all available models from /api/tags
		allModels, err := getAllModels(host, client)
		if err != nil {
			return modelsLoadErr(err)
		}

		// Create a set of loaded model names for quick lookups
		loadedModelSet := make(map[string]struct{})
		for _, m := range loadedModels {
			loadedModelSet[m] = struct{}{}
		}

		// Create the final list of model items, separating loaded models
		var loadedItems []list.Item
		var otherItems []list.Item
		for _, m := range allModels {
			_, isLoaded := loadedModelSet[m.Name]
			listItem := item{title: m.Name, desc: "Select this model", loaded: isLoaded}
			if isLoaded {
				loadedItems = append(loadedItems, listItem)
			} else {
				otherItems = append(otherItems, listItem)
			}
		}

		// Prepend loaded models to the list so they appear first
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

// getAllModels fetches the list of all available models from the /api/tags endpoint.
func getAllModels(host Host, client *http.Client) ([]struct{ Name string }, error) {
	req, err := http.NewRequest("GET", host.URL+"/api/tags", nil)
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

	var tagsResp ollamaTagsResponse
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil, err
	}

	models := make([]struct{ Name string }, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		models[i] = struct{ Name string }{Name: m.Name}
	}

	return models, nil
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
		// If there are any loaded models, the first one in the list is auto-selected.
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

// main is the entry point of the CLI application.
// It loads the configuration, initializes the Bubble Tea program, and starts the event loop.
func main() {
	// Open a log file for debugging
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatalf("could not open log file: %v", err)
	}
	defer f.Close()

	cfg, err := loadConfig("cli/config.json")
	if err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	m := initialModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.program = p

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
