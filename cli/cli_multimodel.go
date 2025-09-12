package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// multimodelViewState represents the current state of the multimodel application's view.
type multimodelViewState int

const (
	// multimodelViewAssignment is the state where users assign models to hosts
	multimodelViewAssignment multimodelViewState = iota
	// multimodelViewLoadingChat is the state while models are being loaded for chat
	multimodelViewLoadingChat
	// multimodelViewChat is the multimodel chat interface
	multimodelViewChat
)

// hostModelAssignment represents a host with its assigned model
type hostModelAssignment struct {
	host          Host
	selectedModel string
	models        []string
	isAssigned    bool
}

// multimodelColumnResponse holds the streaming response data for a single column
type multimodelColumnResponse struct {
	hostIndex        int
	content          strings.Builder
	isStreaming      bool
	error            error
	meta             LLMResponseMeta
	chatHistory      []chatMessage // Add chat history for this column
	requestStartTime time.Time
}

// multimodelModel is the main application model for multimodel mode
type multimodelModel struct {
	// Application configuration
	config *Config
	// HTTP client for API requests
	client *http.Client
	// Current view state
	state multimodelViewState
	// Indicates if an operation is in progress
	isLoading bool
	// Stores any error encountered
	err error

	// Host model assignments
	assignments []hostModelAssignment
	// Currently selected host for assignment
	selectedHostIndex int
	// Model selection list for current host
	modelList list.Model
	// Indicates if we're in model selection mode
	inModelSelection bool

	// Chat interface components
	textArea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	// Chat data
	chatHistory      []chatMessage
	columnResponses  [4]multimodelColumnResponse
	requestStartTime time.Time

	// UI dimensions
	width, height int
	// Reference to the Bubble Tea program
	program *tea.Program
}

// assignmentItem represents a host in the assignment list
type assignmentItem struct {
	host          Host
	assignedModel string
	isAssigned    bool
}

func (i assignmentItem) Title() string {
	title := i.host.Name
	if i.isAssigned {
		title += fmt.Sprintf(" → %s", i.assignedModel)
	}
	return title
}

func (i assignmentItem) Description() string {
	if i.isAssigned {
		return "Model assigned"
	}
	return "Select a model for this host"
}

func (i assignmentItem) FilterValue() string {
	return i.host.Name
}

// multimodelAssignmentsReadyMsg is sent when model assignments are loaded
type multimodelAssignmentsReadyMsg struct{}

// multimodelChatReadyMsg is sent when chat interface is ready
type multimodelChatReadyMsg struct{}

// multimodelChatReadyErr is sent when chat loading fails
type multimodelChatReadyErr error

// multimodelStreamChunkMsg is sent for streaming response chunks
type multimodelStreamChunkMsg struct {
	hostIndex int
	message   chatMessage // Change content to message
}

// multimodelStreamEndMsg is sent when a stream completes
type multimodelStreamEndMsg struct {
	hostIndex int
	meta      LLMResponseMeta
}

// multimodelStreamErr is sent when a stream encounters an error
type multimodelStreamErr struct {
	hostIndex int
	err       error
}

// initialMultimodelModel creates a new multimodel with default values
func initialMultimodelModel(cfg *Config) *multimodelModel {
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

	vp := viewport.New(100, 5)

	// Initialize assignments
	assignments := make([]hostModelAssignment, len(cfg.Hosts))
	for i, host := range cfg.Hosts {
		assignments[i] = hostModelAssignment{
			host:       host,
			models:     host.Models,
			isAssigned: false,
		}
	}

	// Initialize column responses
	var columnResponses [4]multimodelColumnResponse
	for i := range columnResponses {
		columnResponses[i] = multimodelColumnResponse{
			hostIndex: i,
		}
	}

	modelList := list.New(nil, list.NewDefaultDelegate(), 0, 0)

	return &multimodelModel{
		config:            cfg,
		state:             multimodelViewAssignment,
		assignments:       assignments,
		selectedHostIndex: 0,
		modelList:         modelList,
		spinner:           s,
		textArea:          ta,
		viewport:          vp,
		columnResponses:   columnResponses,
	}
}

// loadMultimodelChatCmd prepares the chat interface for multimodel
func loadMultimodelChatCmd(assignments []hostModelAssignment) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, you might want to pre-load models here
		// For now, we'll just signal that chat is ready
		time.Sleep(time.Millisecond * 500) // Simulate loading
		return multimodelChatReadyMsg{}
	}
}

// multimodelStreamChatCmd initiates streaming chat for all assigned host/model pairs
func multimodelStreamChatCmd(p *tea.Program, m *multimodelModel) tea.Cmd { // Pass the entire model
	return func() tea.Msg {
		// Start streaming for each assigned host/model pair
		for i, assignment := range m.assignments { // Use m.assignments
			if assignment.isAssigned {
				go func(hostIndex int, host Host, model string, history []chatMessage) { // Pass history to goroutine
					if err := streamToColumn(p, hostIndex, host, model, history, m.client); err != nil { // Use m.client
						p.Send(multimodelStreamErr{hostIndex: hostIndex, err: err})
					}
				}(i, assignment.host, assignment.selectedModel, m.columnResponses[i].chatHistory) // Pass individual column history
			}
		}
		return nil
	}
}

// streamToColumn handles streaming for a single column
func streamToColumn(p *tea.Program, hostIndex int, host Host, modelName string, history []chatMessage, client *http.Client) error {
	payload := map[string]any{
		"model":    modelName,
		"messages": history,
		"stream":   true,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(context.Background(), "POST", host.URL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var finalChunk streamChunk
	for {
		var chunk streamChunk
		if err := decoder.Decode(&chunk); err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		p.Send(multimodelStreamChunkMsg{
			hostIndex: hostIndex,
			message: chatMessage{
				Role:    chunk.Message.Role,
				Content: chunk.Message.Content,
			},
		})
		if chunk.Done {
			finalChunk = chunk
			break
		}
	}

	p.Send(multimodelStreamEndMsg{
		hostIndex: hostIndex,
		meta: LLMResponseMeta{
			Model:              finalChunk.Model,
			CreatedAt:          time.Now(),
			Done:               finalChunk.Done,
			TotalDuration:      finalChunk.TotalDuration,
			LoadDuration:       finalChunk.LoadDuration,
			PromptEvalCount:    finalChunk.PromptEvalCount,
			PromptEvalDuration: finalChunk.PromptEvalDuration,
			EvalCount:          finalChunk.EvalCount,
			EvalDuration:       finalChunk.EvalDuration,
		},
	})
	return nil
}

// Init initializes the multimodel Bubble Tea model
func (m *multimodelModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles all message updates for multimodel mode
func (m *multimodelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.state == multimodelViewChat {
				m.state = multimodelViewAssignment
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.modelList.SetSize(msg.Width-2, msg.Height-8)
		m.textArea.SetWidth(msg.Width - 3)
		headerHeight := 6
		footerHeight := 5
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - footerHeight

	case multimodelChatReadyMsg:
		m.isLoading = false
		m.state = multimodelViewChat
		m.textArea.Focus()
		return m, nil

	case multimodelChatReadyErr:
		m.isLoading = false
		m.err = msg
		return m, nil

	case multimodelStreamChunkMsg:
		if msg.hostIndex < len(m.columnResponses) {
			// Check if there's an existing assistant message to append to
			history := &m.columnResponses[msg.hostIndex].chatHistory
			if len(*history) > 0 && (*history)[len(*history)-1].Role == "assistant" {
				(*history)[len(*history)-1].Content += msg.message.Content
			} else {
				// Start a new assistant message
				*history = append(*history, msg.message)
			}
			m.columnResponses[msg.hostIndex].isStreaming = true
		}
		return m, nil

	case multimodelStreamEndMsg:
		if msg.hostIndex < len(m.columnResponses) {
			m.columnResponses[msg.hostIndex].meta = msg.meta
			m.columnResponses[msg.hostIndex].isStreaming = false
		}
		// Check if all streams are done
		allDone := true
		for i, assignment := range m.assignments {
			if assignment.isAssigned && i < len(m.columnResponses) && m.columnResponses[i].isStreaming {
				allDone = false
				break
			}
		}
		if allDone {
			m.isLoading = false
			m.textArea.Focus()

			// Add responses to chat history
			for i, assignment := range m.assignments {
				if assignment.isAssigned && i < len(m.columnResponses) && m.columnResponses[i].content.Len() > 0 {
					// For multimodel, we might want to store responses differently
					// For now, we'll just store the first response in history
					if len(m.chatHistory) > 0 && m.chatHistory[len(m.chatHistory)-1].Role == "user" {
						// Add combined response
						var combinedResponse strings.Builder
						for j, a := range m.assignments {
							if a.isAssigned && j < len(m.columnResponses) && m.columnResponses[j].content.Len() > 0 {
								combinedResponse.WriteString(fmt.Sprintf("[%s - %s]: %s\n\n",
									a.host.Name, a.selectedModel, m.columnResponses[j].content.String()))
							}
						}
						if combinedResponse.Len() > 0 {
							m.chatHistory = append(m.chatHistory, chatMessage{
								Role:    "assistant",
								Content: combinedResponse.String(),
							})
						}
						break
					}
				}
			}
		}
		return m, nil

	case multimodelStreamErr:
		if msg.hostIndex < len(m.columnResponses) {
			m.columnResponses[msg.hostIndex].error = msg.err
			m.columnResponses[msg.hostIndex].isStreaming = false
		}
		return m, nil

	case tickMsg:
		if m.isLoading {
			return m, tickCmd()
		}
		return m, nil
	}

	switch m.state {
	case multimodelViewAssignment:
		return m.updateAssignment(msg)
	case multimodelViewChat:
		return m.updateChat(msg)
	}

	if m.isLoading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updateAssignment handles updates in assignment mode
func (m *multimodelModel) updateAssignment(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.inModelSelection {
		var cmd tea.Cmd
		m.modelList, cmd = m.modelList.Update(msg)
		cmds = append(cmds, cmd)

		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter":
				if selectedItem, ok := m.modelList.SelectedItem().(item); ok {
					m.assignments[m.selectedHostIndex].selectedModel = selectedItem.Title()
					m.assignments[m.selectedHostIndex].isAssigned = true
					m.inModelSelection = false
				}
			case "esc":
				m.inModelSelection = false
			}
		}
	} else {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up", "k":
				if m.selectedHostIndex > 0 {
					m.selectedHostIndex--
				}
			case "down", "j":
				if m.selectedHostIndex < len(m.assignments)-1 {
					m.selectedHostIndex++
				}
			case "enter":
				// Enter model selection for current host
				items := make([]list.Item, len(m.assignments[m.selectedHostIndex].models))
				for i, model := range m.assignments[m.selectedHostIndex].models {
					items[i] = item{title: model, desc: "Select this model"}
				}
				m.modelList.SetItems(items)
				m.modelList.Title = fmt.Sprintf("Select Model for %s", m.assignments[m.selectedHostIndex].host.Name)
				m.inModelSelection = true
			case "c":
				// Start chat if at least one assignment is made
				hasAssignment := false
				for _, assignment := range m.assignments {
					if assignment.isAssigned {
						hasAssignment = true
						break
					}
				}
				if hasAssignment {
					m.state = multimodelViewLoadingChat
					m.isLoading = true
					m.requestStartTime = time.Now()
					// Clear previous responses
					for i := range m.columnResponses {
						m.columnResponses[i].content.Reset()
						m.columnResponses[i].error = nil
						m.columnResponses[i].isStreaming = false
					}
					cmds = append(cmds, m.spinner.Tick, loadMultimodelChatCmd(m.assignments), tickCmd())
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// updateChat handles updates in chat mode
func (m *multimodelModel) updateChat(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		userInput := strings.TrimSpace(m.textArea.Value())
		if userInput != "" {
			// Add user message to all assigned column histories
			userMsg := chatMessage{Role: "user", Content: userInput}
			for i := range m.columnResponses {
				if m.assignments[i].isAssigned {
					m.columnResponses[i].chatHistory = append(m.columnResponses[i].chatHistory, userMsg)
					m.columnResponses[i].requestStartTime = time.Now()
					m.columnResponses[i].isStreaming = true
				} else {
					m.columnResponses[i].isStreaming = false
				}
				m.columnResponses[i].content.Reset() // Clear content buffer for new streaming response
				m.columnResponses[i].error = nil
			}

			m.requestStartTime = time.Now()
			m.textArea.Blur()
			m.isLoading = true
			cmds = append(cmds, m.spinner.Tick, multimodelStreamChatCmd(m.program, m), tickCmd()) // Pass the entire model
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the multimodel UI based on current state
func (m *multimodelModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(1)
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	switch m.state {
	case multimodelViewAssignment:
		return m.assignmentView()
	case multimodelViewLoadingChat:
		timer := fmt.Sprintf("%.1f", time.Since(m.requestStartTime).Seconds())
		return fmt.Sprintf("\n  %s Loading multimodel chat... %ss\n", m.spinner.View(), timer)
	case multimodelViewChat:
		return m.multimodelChatView()
	default:
		return "Unknown state"
	}
}

// assignmentView renders the host/model assignment interface
func (m *multimodelModel) assignmentView() string {
	var builder strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	builder.WriteString(titleStyle.Render("Multimodel Mode - Assign Models to Hosts") + "\n\n")

	if m.inModelSelection {
		return lipgloss.NewStyle().Margin(1, 2).Render(m.modelList.View())
	}

	// Show host assignments
	for i, assignment := range m.assignments {
		var line strings.Builder

		// Selection indicator
		if i == m.selectedHostIndex {
			line.WriteString("> ")
		} else {
			line.WriteString("  ")
		}

		// Host name
		hostStyle := lipgloss.NewStyle().Bold(true)
		line.WriteString(hostStyle.Render(assignment.host.Name))
		line.WriteString(" → ")

		// Assigned model or placeholder
		if assignment.isAssigned {
			modelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
			line.WriteString(modelStyle.Render(assignment.selectedModel))
		} else {
			placeholderStyle := lipgloss.NewStyle().Faint(true)
			line.WriteString(placeholderStyle.Render("(no model assigned)"))
		}

		builder.WriteString(line.String() + "\n")
	}

	builder.WriteString("\n")

	// Instructions
	helpStyle := lipgloss.NewStyle().Faint(true)
	builder.WriteString(helpStyle.Render("↑/↓: Navigate  Enter: Select Model  C: Start Chat  Q: Quit\n"))

	// Chat button if assignments exist
	hasAssignment := false
	for _, assignment := range m.assignments {
		if assignment.isAssigned {
			hasAssignment = true
			break
		}
	}
	if hasAssignment {
		builder.WriteString("\n")
		chatStyle := lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipgloss.Color("0")).Padding(0, 1)
		builder.WriteString(chatStyle.Render("Press 'C' to start multimodel chat"))
	}

	return lipgloss.NewStyle().Margin(1, 2).Render(builder.String())
}

// multimodelChatView renders the 4-column chat interface
func (m *multimodelModel) multimodelChatView() string {
	var builder strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Padding(0, 1)
	header := headerStyle.Render("Multimodel Chat")
	help := lipgloss.NewStyle().Faint(true).Render(" (tab to reassign, q to quit)")
	builder.WriteString(header + help + "\n\n")

	// Calculate column width
	colWidth := (m.width - 8) / 4 // Account for borders and spacing

	// Column headers
	var headerCells []string
	for i := 0; i < 4; i++ {
		var colHeader string
		if i < len(m.assignments) && m.assignments[i].isAssigned {
			colHeader = fmt.Sprintf("%s\n%s", m.assignments[i].host.Name, m.assignments[i].selectedModel)
		} else {
			colHeader = "Empty\n"
		}

		colHeaderStyle := lipgloss.NewStyle().
			Width(colWidth).
			Height(2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Align(lipgloss.Center).
			Bold(true)

		headerCells = append(headerCells, colHeaderStyle.Render(colHeader))
	}
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, headerCells...)
	builder.WriteString(headerRow + "\n")

	// Response columns
	// Calculate dynamic height for chat history
	chatHeight := m.height - lipgloss.Height(headerRow) - lipgloss.Height(m.textArea.View()) - 5 // Adjust 5 for padding/margins

	var chatRows []string
	for i := 0; i < 4; i++ {
		var colChatHistory strings.Builder
		if i < len(m.assignments) && m.assignments[i].isAssigned {
			if m.columnResponses[i].error != nil {
				errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
				colChatHistory.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.columnResponses[i].error)))
			} else {
				userStyle := lipgloss.NewStyle().Bold(true)
				assistantStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))

				for _, msg := range m.columnResponses[i].chatHistory {
					var role, content string
					if msg.Role == "assistant" {
						role = assistantStyle.Render("Assistant: ")
						content = msg.Content
					} else {
						role = userStyle.Render("You: ")
						content = msg.Content
					}
					wrappedContent := lipgloss.NewStyle().Width(colWidth - lipgloss.Width(role) - 2).Render(content)
					colChatHistory.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, role, wrappedContent) + "\n")
				}
			}
		}

		colStyle := lipgloss.NewStyle().
			Width(colWidth).
			Height(chatHeight).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

		chatRows = append(chatRows, colStyle.Render(colChatHistory.String()))
	}
	builder.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, chatRows...) + "\n")

	// Input area or loading indicator
	var loadingIndicators []string
	for i := range m.columnResponses {
		if m.columnResponses[i].isStreaming {
			timer := fmt.Sprintf("%.1f", time.Since(m.columnResponses[i].requestStartTime).Seconds())
			loadingIndicators = append(loadingIndicators, fmt.Sprintf("%s Querying %s... %ss", m.spinner.View(), m.assignments[i].selectedModel, timer))
		}
	}

	if m.isLoading {
		builder.WriteString("\n" + strings.Join(loadingIndicators, "\n"))
	} else {
		builder.WriteString("\n" + m.textArea.View())
	}

	return builder.String()
}

// StartMultimodelGUI starts the multimodel GUI
func StartMultimodelGUI(cfg *Config) error {
	m := initialMultimodelModel(cfg)
	m.client = &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false,
		},
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.program = p

	_, err := p.Run()
	return err
}
