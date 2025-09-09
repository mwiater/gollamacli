package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

const (
	configFile = "config.json"
)

// Host represents a single host in the config
type Host struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

// Config represents the application's configuration
type Config struct {
	Hosts  []Host   `json:"hosts"`
	Models []string `json:"models"`
	Debug  bool     `json:"debug"`
}

// LLMHost defines the interface for a host
type LLMHost interface {
	PullModel(model string)
	DeleteModel(model string)
	ListModels() ([]string, error)
	GetName() string
	GetType() string
}

// OllamaHost is an implementation of LLMHost for Ollama
type OllamaHost struct {
	Name string
	URL  string
}

// LMStudioHost is an implementation of LLMHost for LM Studio
type LMStudioHost struct {
	Name string
	URL  string
}

func (h *OllamaHost) GetName() string {
	return h.Name
}

func (h *OllamaHost) GetType() string {
	return "ollama"
}

func (h *LMStudioHost) GetName() string {
	return h.Name
}

func (h *LMStudioHost) GetType() string {
	return "lmstudio"
}

// loadConfig reads and parses the configuration file.
func loadConfig() (Config, error) {
	var config Config
	file, err := os.Open(configFile)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	return config, err
}

// createHosts creates a slice of LLMHost based on the config
func createHosts(config Config) []LLMHost {
	var hosts []LLMHost
	for _, hostConfig := range config.Hosts {
		switch hostConfig.Type {
		case "ollama":
			hosts = append(hosts, &OllamaHost{Name: hostConfig.Name, URL: hostConfig.URL})
		case "lmstudio":
			hosts = append(hosts, &LMStudioHost{Name: hostConfig.Name, URL: hostConfig.URL})
		default:
			fmt.Printf("Unknown host type: %s\n", hostConfig.Type)
		}
	}
	return hosts
}

// PullModels reads models from config.json and pulls them to each node.
func PullModels() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", configFile, err)
		return
	}

	hosts := createHosts(config)
	var wg sync.WaitGroup
	for _, host := range hosts {
		wg.Add(1)
		go func(h LLMHost) {
			defer wg.Done()
			if h.GetType() != "ollama" {
				fmt.Printf("Pulling models is not supported for %s (%s)\n", h.GetName(), h.GetType())
				return
			}
			fmt.Printf("Starting model pulls for %s...\n", h.GetName())
			for _, model := range config.Models {
				fmt.Printf("  -> Pulling model: %s on %s\n", model, h.GetName())
				h.PullModel(model)
			}
		}(host)
	}
	wg.Wait()
	fmt.Println("All model pull commands have finished.")
}

// pullModel pulls a model to a single node.
func (h *OllamaHost) PullModel(model string) {
	url := fmt.Sprintf("%s/api/pull", h.URL)
	payload := map[string]string{"name": model}
	body, _ := json.Marshal(payload)
	_, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Error pulling model %s on %s: %v\n", model, h.Name, err)
	}
}

func (h *LMStudioHost) PullModel(model string) {
	fmt.Printf("Pulling models is not supported for LM Studio host: %s\n", h.Name)
}

// DeleteModels reads config.json and deletes any models not on the list from each node.
func DeleteModels() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", configFile, err)
		return
	}

	hosts := createHosts(config)
	var wg sync.WaitGroup
	for _, host := range hosts {
		wg.Add(1)
		go func(h LLMHost) {
			defer wg.Done()
			if h.GetType() != "ollama" {
				fmt.Printf("Deleting models is not supported for %s (%s)\n", h.GetName(), h.GetType())
				return
			}
			deleteModelsOnNode(h, config.Models)
		}(host)
	}
	wg.Wait()
	fmt.Println("All model cleanup commands have finished.")
}

// deleteModelsOnNode deletes any models not on the list from a single node.
func deleteModelsOnNode(host LLMHost, modelsToKeep []string) {
	fmt.Printf("Starting model cleanup for %s...\n", host.GetName())
	models, err := host.ListModels()
	if err != nil {
		fmt.Printf("Error getting models from %s: %v\n", host.GetName(), err)
		return
	}

	modelsToKeepSet := make(map[string]struct{})
	for _, m := range modelsToKeep {
		modelsToKeepSet[m] = struct{}{}
	}

	for _, installedModel := range models {
		// Extract model name from the formatted string
		parts := strings.Split(installedModel, " ")
		modelName := parts[1]
		if _, keep := modelsToKeepSet[modelName]; !keep {
			fmt.Printf("  -> Deleting model: %s on %s\n", modelName, host.GetName())
			host.DeleteModel(modelName)
		} else {
			fmt.Printf("  -> Keeping model: %s on %s\n", modelName, host.GetName())
		}
	}
}

// deleteModel deletes a model from a single node.
func (h *OllamaHost) DeleteModel(model string) {
	url := fmt.Sprintf("%s/api/delete", h.URL)
	payload := map[string]string{"model": model}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("DELETE", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error deleting model %s on %s: %v\n", model, h.Name, err)
	}
}

func (h *LMStudioHost) DeleteModel(model string) {
	fmt.Printf("Deleting models is not supported for LM Studio host: %s\n", h.Name)
}

// SyncModels runs DeleteModels and then PullModels.
func SyncModels() {
	DeleteModels()
	PullModels()
}

// ListModels lists all models on each node, indicating which are currently loaded.
func ListModels() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", configFile, err)
		return
	}

	hosts := createHosts(config)
	nodeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	nodeModels := make(map[string][]string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, host := range hosts {
		wg.Add(1)
		go func(h LLMHost) {
			defer wg.Done()
			models, err := h.ListModels()
			mu.Lock()
			if err != nil {
				nodeModels[h.GetName()] = []string{fmt.Sprintf("Error: %v", err)}
			} else {
				nodeModels[h.GetName()] = models
			}
			mu.Unlock()
		}(host)
	}
	wg.Wait()

	var sortedNodes []string
	for node := range nodeModels {
		sortedNodes = append(sortedNodes, node)
	}
	sort.Strings(sortedNodes)

	for _, node := range sortedNodes {
		fmt.Println(nodeStyle.Render(fmt.Sprintf("%s:", node)))
		for _, model := range nodeModels[node] {
			cleanedModelString := strings.TrimSpace(strings.ReplaceAll(model, "-", ""))
			fmt.Println("  >>> " + cleanedModelString)
		}
		fmt.Println()
	}
}

// listModelsOnNode gets the models on a single node.
func (h *OllamaHost) ListModels() ([]string, error) {
	modelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	loadedModelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))

	runningModels, err := h.getRunningModels()
	if err != nil {
		return nil, fmt.Errorf("could not get running models: %v", err)
	}

	url := fmt.Sprintf("%s/api/tags", h.URL)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not list models: Ollama is not accessible on %s", h.Name)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from %s: %v", h.Name, err)
	}

	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil, fmt.Errorf("error parsing models from %s: %v", h.Name, err)
	}

	var models []string
	for _, model := range tagsResp.Models {
		if _, ok := runningModels[model.Name]; ok {
			models = append(models, loadedModelStyle.Render(fmt.Sprintf("- %s (CURRENTLY LOADED)", model.Name)))
		} else {
			models = append(models, modelStyle.Render(fmt.Sprintf("- %s", model.Name)))
		}
	}
	return models, nil
}

func (h *LMStudioHost) ListModels() ([]string, error) {
	modelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	url := fmt.Sprintf("%s/api/v0/models", h.URL)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not list models: LM Studio is not accessible on %s", h.Name)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from %s: %v", h.Name, err)
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("error parsing models from %s: %v", h.Name, err)
	}

	var models []string
	for _, model := range modelsResp.Data {
		models = append(models, modelStyle.Render(fmt.Sprintf("- %s", model.ID)))
	}
	return models, nil
}

// getRunningModels gets the running models on a single node.
func (h *OllamaHost) getRunningModels() (map[string]struct{}, error) {
	runningModels := make(map[string]struct{})
	url := fmt.Sprintf("%s/api/ps", h.URL)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var psResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &psResp); err != nil {
		return nil, err
	}

	for _, model := range psResp.Models {
		runningModels[model.Name] = struct{}{}
	}

	return runningModels, nil
}