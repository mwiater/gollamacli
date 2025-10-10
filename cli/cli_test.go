// cli/cli_test.go
package cli

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLoadConfig(t *testing.T) {
	// Test case 1: Valid config
	validConfig := `{
		"hosts": [
			{
				"name": "Test Host",
				"url": "http://localhost:11434",
				"models": ["model1", "model2"]
			}
		]
	}`
	tmpfile, err := os.CreateTemp("", "config.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(validConfig)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(tmpfile.Name())
	if err != nil {
		t.Errorf("loadConfig() with valid config failed: %v", err)
	}
	if len(cfg.Hosts) != 1 {
		t.Errorf("Expected 1 host, got %d", len(cfg.Hosts))
	}

	// Test case 2: Invalid JSON
	invalidJSON := `{ "hosts": [`
	tmpfile2, err := os.CreateTemp("", "config.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile2.Name())
	if _, err := tmpfile2.Write([]byte(invalidJSON)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile2.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = loadConfig(tmpfile2.Name())
	if err == nil {
		t.Error("loadConfig() with invalid JSON should have failed, but it didn't")
	}

	// Test case 3: No hosts
	noHosts := `{ "hosts": [] }`
	tmpfile3, err := os.CreateTemp("", "config.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile3.Name())
	if _, err := tmpfile3.Write([]byte(noHosts)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile3.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = loadConfig(tmpfile3.Name())
	if err == nil {
		t.Error("loadConfig() with no hosts should have failed, but it didn't")
	}

	// Test case 4: File not found
	_, err = loadConfig("nonexistent.json")
	if err == nil {
		t.Error("loadConfig() with nonexistent file should have failed, but it didn't")
	}
}

func TestUpdate(t *testing.T) {
	cfg := &Config{
		Hosts: []Host{
			{
				Name:   "Test Host",
				URL:    "http://localhost:11434",
				Models: []string{"model1", "model2"},
			},
		},
	}
	m := initialModel(cfg)

	// Test case 1: Initial state
	if m.state != viewHostSelector {
		t.Errorf("Expected initial state to be viewHostSelector, got %v", m.state)
	}

	// Test case 2: Quit message
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("Expected a quit command, but got nil")
	}

	// Test case 3: Ctrl+c
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Expected a quit command, but got nil")
	}

	// Test case 4: Window size message
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 100})
	m = newModel.(*model)
	if m.width != 100 || m.height != 100 {
		t.Errorf("Expected width and height to be 100, got %d and %d", m.width, m.height)
	}

	// Test case 5: Host selection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ps" {
			w.Write([]byte(`{"models":[{"name":"model1"}]}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg.Hosts[0].URL = server.URL
	m = initialModel(cfg)

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	if m.state != viewHostSelector {
		t.Errorf("Expected state to be viewHostSelector, got %v", m.state)
	}
}

func TestView(t *testing.T) {
	cfg := &Config{
		Hosts: []Host{
			{
				Name:   "Test Host",
				URL:    "http://localhost:11434",
				Models: []string{"model1", "model2"},
			},
		},
	}
	m := initialModel(cfg)

	// Test case 1: Initializing view
	m.width = 0
	view := m.View()
	if view != "Initializing..." {
		t.Errorf("Expected view to be 'Initializing...', got '%s'", view)
	}

	// Test case 2: Error view
	m.width = 100
	m.err = modelsLoadErr(errors.New("test error"))
	view = m.View()
	if !strings.Contains(view, "Error") {
		t.Errorf("Expected view to contain 'Error', got '%s'", view)
	}
	m.err = nil

	// Test case 3: Host selector view
	view = m.View()
	if !strings.Contains(view, "Select a Host") {
		t.Errorf("Expected view to contain 'Select a Host', got '%s'", view)
	}
}
