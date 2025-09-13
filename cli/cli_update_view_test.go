package cli

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSingleModel_StateTransitions_And_View(t *testing.T) {
	cfg := &Config{Hosts: []Host{{Name: "HostA", URL: "http://x", Models: []string{"m1", "m2"}}}}
	m := initialModel(cfg)

	// Set a window size so View() renders
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// From host selector: press enter to trigger loading models
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*model)
	if !m.isLoading || m.state != viewHostSelector {
		t.Fatalf("expected loading host selector; got loading=%v state=%v", m.isLoading, m.state)
	}

	// Deliver modelsReadyMsg
	items := []list.Item{item{title: "m1", desc: "Select this model", loaded: true}, item{title: "m2", desc: "Select this model"}}
	m2, _ = m.Update(modelsReadyMsg{models: items, loadedModels: []string{"m1"}})
	m = m2.(*model)
	if m.state != viewModelSelector || len(m.modelList.Items()) != 2 {
		t.Fatalf("expected model selector with 2 items; state=%v count=%d", m.state, len(m.modelList.Items()))
	}

	// Select model and load
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*model)
	if !m.isLoading || m.state != viewLoadingChat {
		t.Fatalf("expected loading chat; got loading=%v state=%v", m.isLoading, m.state)
	}

	// Chat ready
	m2, _ = m.Update(chatReadyMsg{})
	m = m2.(*model)
	if m.state != viewChat {
		t.Fatalf("expected chat view; got %v", m.state)
	}

	// Send a user message
	m.textArea.SetValue("hello")
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*model)
	if len(m.chatHistory) == 0 || m.chatHistory[len(m.chatHistory)-1].Role != "user" {
		t.Fatalf("expected last message to be user; history=%v", m.chatHistory)
	}
	if !m.isLoading {
		t.Fatalf("expected loading after sending message")
	}

	// Stream a chunk then end
	m2, _ = m.Update(streamChunkMsg("world"))
	m = m2.(*model)
	if !strings.Contains(m.responseBuf.String(), "world") {
		t.Fatalf("expected response buffer to contain chunk")
	}
	m2, _ = m.Update(streamEndMsg{meta: LLMResponseMeta{Done: true}})
	m = m2.(*model)
	if m.isLoading {
		t.Fatalf("expected not loading after stream end")
	}
	if len(m.chatHistory) < 2 || m.chatHistory[len(m.chatHistory)-1].Role != "assistant" {
		t.Fatalf("expected assistant message after end; history=%v", m.chatHistory)
	}

	// Basic view rendering checks
	out := m.View()
	if !strings.Contains(out, "Assistant:") || !strings.Contains(out, "You:") {
		t.Fatalf("expected roles in view output; got: %s", out)
	}
}

func TestMultimodel_Assignment_And_Chat_Flow(t *testing.T) {
	cfg := &Config{Hosts: []Host{
		{Name: "H1", URL: "http://x", Models: []string{"m1", "m2"}},
		{Name: "H2", URL: "http://y", Models: []string{"m3"}},
		{Name: "H3", URL: "http://z", Models: []string{"m4"}},
		{Name: "H4", URL: "http://w", Models: []string{"m5"}},
	}}
	mm := initialMultimodelModel(cfg)
	// Provide a program placeholder to avoid nil deref in any goroutine sends (we don't start them here)
	mm.program = &tea.Program{}

	// Size for rendering
	_, _ = mm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Enter model selection for first host
	m2, _ := mm.updateAssignment(tea.KeyMsg{Type: tea.KeyEnter})
	mm = m2.(*multimodelModel)
	if !mm.inModelSelection || len(mm.modelList.Items()) == 0 {
		t.Fatalf("expected to be in model selection with items")
	}

	// Select current model
	m2, _ = mm.updateAssignment(tea.KeyMsg{Type: tea.KeyEnter})
	mm = m2.(*multimodelModel)
	if !mm.assignments[mm.selectedHostIndex].isAssigned {
		t.Fatalf("expected assignment to be marked assigned")
	}

	// Start chat (c)
	m2, _ = mm.updateAssignment(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mm = m2.(*multimodelModel)
	if mm.state != multimodelViewLoadingChat {
		t.Fatalf("expected loading chat state; got %v", mm.state)
	}

	// Ready
	m2, _ = mm.Update(multimodelChatReadyMsg{})
	mm = m2.(*multimodelModel)
	if mm.state != multimodelViewChat {
		t.Fatalf("expected chat view; got %v", mm.state)
	}

	// Send a prompt
	mm.textArea.SetValue("question")
	m2, _ = mm.updateChat(tea.KeyMsg{Type: tea.KeyEnter})
	mm = m2.(*multimodelModel)
	// At least the first assigned column should be streaming
	if !mm.columnResponses[0].isStreaming {
		t.Fatalf("expected first column streaming")
	}

	// Stream chunk to column 0, then end
	m2, _ = mm.Update(multimodelStreamChunkMsg{hostIndex: 0, message: chatMessage{Role: "assistant", Content: "hi"}})
	mm = m2.(*multimodelModel)
	if len(mm.columnResponses[0].chatHistory) == 0 {
		t.Fatalf("expected assistant message in column history")
	}
	m2, _ = mm.Update(multimodelStreamEndMsg{hostIndex: 0, meta: LLMResponseMeta{Done: true}})
	mm = m2.(*multimodelModel)
	if mm.isLoading {
		t.Fatalf("expected not loading after all streams ended")
	}

	// Render
	out := mm.multimodelChatView()
	if !strings.Contains(out, "Multimodel Chat") {
		t.Fatalf("expected chat header in view; got: %s", out)
	}
}
