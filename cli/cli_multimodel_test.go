// cli/cli_multimodel_test.go
package cli

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialMultimodelModel(t *testing.T) {
	cfg := &Config{
		Hosts: []Host{
			{
				Name:   "Test Host 1",
				URL:    "http://localhost:11434",
				Models: []string{"model1", "model2"},
			},
			{
				Name:   "Test Host 2",
				URL:    "http://localhost:11435",
				Models: []string{"model3", "model4"},
			},
		},
	}
	m := initialMultimodelModel(cfg)

	if m.state != multimodelViewAssignment {
		t.Errorf("Expected initial state to be multimodelViewAssignment, got %v", m.state)
	}

	if len(m.assignments) != 2 {
		t.Errorf("Expected 2 assignments, got %d", len(m.assignments))
	}
}

func TestMultimodelUpdate(t *testing.T) {
	cfg := &Config{
		Hosts: []Host{
			{
				Name:   "Test Host 1",
				URL:    "http://localhost:11434",
				Models: []string{"model1", "model2"},
			},
		},
	}
	m := initialMultimodelModel(cfg)

	// Test case 1: Ctrl+c
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Expected a quit command, but got nil")
	}

	// Test case 2: Navigate assignment view
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(*multimodelModel)
	if m.selectedHostIndex != 0 {
		t.Errorf("Expected selectedHostIndex to be 0, got %d", m.selectedHostIndex)
	}

	// Test case 3: Enter model selection
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*multimodelModel)
	if !m.inModelSelection {
		t.Error("Expected inModelSelection to be true, but it's false")
	}

	// Test case 4: Exit model selection with esc
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")})
	m = newModel.(*multimodelModel)
	if m.inModelSelection {
		t.Error("Expected inModelSelection to be false, but it's true")
	}

	// Test case 5: Assign model
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Enter model selection
	m = newModel.(*multimodelModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Select first model
	m = newModel.(*multimodelModel)
	if !m.assignments[0].isAssigned {
		t.Error("Expected model to be assigned, but it's not")
	}
	if m.assignments[0].selectedModel != "model1" {
		t.Errorf("Expected selected model to be 'model1', got '%s'", m.assignments[0].selectedModel)
	}

	// Test case 6: Start chat
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = newModel.(*multimodelModel)
	if m.state != multimodelViewLoadingChat {
		t.Errorf("Expected state to be multimodelViewLoadingChat, got %v", m.state)
	}
}

func TestMultimodelView(t *testing.T) {
	cfg := &Config{
		Hosts: []Host{
			{
				Name:   "Test Host 1",
				URL:    "http://localhost:11434",
				Models: []string{"model1", "model2"},
			},
		},
	}
	m := initialMultimodelModel(cfg)

	// Test case 1: Initializing view
	m.width = 0
	view := m.View()
	if view != "Initializing..." {
		t.Errorf("Expected view to be 'Initializing...', got '%s'", view)
	}

	// Test case 2: Error view
	m.width = 100
	m.err = multimodelChatReadyErr(errors.New("test error"))
	view = m.View()
	if !strings.Contains(view, "Error") {
		t.Errorf("Expected view to contain 'Error', got '%s'", view)
	}
	m.err = nil

	// Test case 3: Assignment view
	view = m.View()
	if !strings.Contains(view, "Multimodel Mode - Assign Models to Hosts") {
		t.Errorf("Expected view to contain 'Multimodel Mode - Assign Models to Hosts', got '%s'", view)
	}
}
