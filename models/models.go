// models/models.go
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
	"github.com/k0kubun/pp"
)

// ModelParameters holds the detailed parameters of a model.
type ModelParameters struct {
	Model      string       `json:"model,omitempty"`
	License    string       `json:"license,omitempty"`
	Modelfile  string       `json:"modelfile,omitempty"`
	Parameters string       `json:"parameters,omitempty"`
	Template   string       `json:"template,omitempty"`
	Details    ModelDetails `json:"details,omitempty"`
}

// ModelDetails holds the nested details of a model.
type ModelDetails struct {
	Family            string `json:"family,omitempty"`
	Format            string `json:"format,omitempty"`
	ParameterSize     string `json:"parameter_size,omitempty"`
	QuantizationLevel string `json:"quantization_level,omitempty"`
}

const (
	configFile = "config.json"
)

// Host represents a single host entry in the configuration.
// It includes a display name, base URL, host type, and a list of associated
// model identifiers. Currently only `ollama` hosts are supported.
type Host struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Type   string   `json:"type"`
	Models []string `json:"models"`
}

// Config is the application's configuration structure containing the set of hosts
// and global flags such as debug mode.
type Config struct {
	Hosts []Host `json:"hosts"`
	Debug bool   `json:"debug"`
}

// LLMHost defines the model lifecycle and metadata operations a host must support.
// Implementations should pull, delete, list, and unload models, and expose basic metadata.
type LLMHost interface {
	PullModel(model string)
	DeleteModel(model string)
	ListModels() ([]string, error)
	ListRawModels() ([]string, error)
	UnloadModel(model string)
	GetName() string
	GetType() string
	GetModels() []string
	GetModelParameters() ([]ModelParameters, error)
}

// OllamaHost implements LLMHost for Ollama servers.
type OllamaHost struct {
	Name   string
	URL    string
	Models []string
}

// GetName returns the display name of the Ollama host.
func (h *OllamaHost) GetName() string {
	return h.Name
}

// GetType returns the type identifier for Ollama hosts ("ollama").
func (h *OllamaHost) GetType() string {
	return "ollama"
}

// GetModels returns the configured models for the Ollama host.
func (h *OllamaHost) GetModels() []string {
	return h.Models
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
			hosts = append(hosts, &OllamaHost{Name: hostConfig.Name, URL: hostConfig.URL, Models: hostConfig.Models})
		default:
			fmt.Printf("Unknown host type: %s\n", hostConfig.Type)
		}
	}
	return hosts
}

// PullModels reads models from config.json and pulls them to each supported host.
// For Ollama hosts, it issues /api/pull requests for each configured model.
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
			for _, model := range h.GetModels() {
				fmt.Printf("  -> Pulling model: %s on %s\n", model, h.GetName())
				h.PullModel(model)
			}
		}(host)
	}
	wg.Wait()
	fmt.Println("All model pull commands have finished.")
}

// PullModel pulls the provided model to the Ollama host via the /api/pull endpoint.
func (h *OllamaHost) PullModel(model string) {
	url := fmt.Sprintf("%s/api/pull", h.URL)
	payload := map[string]string{"name": model}
	body, _ := json.Marshal(payload)
	_, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Error pulling model %s on %s: %v\n", model, h.Name, err)
	}
}

// DeleteModels reads config.json and deletes any models not on the list from each supported host.
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
			deleteModelsOnNode(h, h.GetModels())
		}(host)
	}
	wg.Wait()
	fmt.Println("All model cleanup commands have finished.")
}

// deleteModelsOnNode deletes models on a single host that are not present in modelsToKeep.
func deleteModelsOnNode(host LLMHost, modelsToKeep []string) {
	fmt.Printf("Starting model cleanup for %s...\n", host.GetName())
	models, err := host.ListRawModels()
	if err != nil {
		fmt.Printf("Error getting models from %s: %v\n", host.GetName(), err)
		return
	}

	modelsToKeepSet := make(map[string]struct{})
	for _, m := range modelsToKeep {
		modelsToKeepSet[m] = struct{}{}
	}

	for _, installedModelName := range models {
		modelName := installedModelName
		if _, keep := modelsToKeepSet[modelName]; !keep {
			fmt.Printf("  -> Deleting model: %s on %s\n", modelName, host.GetName())
			host.DeleteModel(modelName)
		} else {
			fmt.Printf("  -> Keeping model: %s on %s\n", modelName, host.GetName())
		}
	}
}

// DeleteModel deletes the specified model from an Ollama host via the /api/delete endpoint.
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

// UnloadModels unloads all currently loaded models on each supported host.
func UnloadModels() {
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
				fmt.Printf("Unloading models is not supported for %s (%s)\n", h.GetName(), h.GetType())
				return
			}
			fmt.Printf("Unloading models for %s...\n", h.GetName())
			runningModels, err := h.(*OllamaHost).getRunningModels()
			if err != nil {
				fmt.Printf("Error getting running models from %s: %v\n", h.GetName(), err)
				return
			}
			for model := range runningModels {
				fmt.Printf("  -> Unloading model: %s on %s\n", model, h.GetName())
				h.UnloadModel(model)
			}
		}(host)
	}
	wg.Wait()
	fmt.Println("All model unload commands have finished.")
}

// UnloadModel unloads a model from an Ollama host by sending a chat request with keep_alive set to 0.
func (h *OllamaHost) UnloadModel(model string) {
	url := fmt.Sprintf("%s/api/chat", h.URL)
	payload := map[string]any{"model": model, "keep_alive": 0}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error unloading model %s on %s: %v\n", model, h.Name, err)
	}
}

// function aliases allow tests to spy call order.
var (
	deleteModelsFunc = DeleteModels
	pullModelsFunc   = PullModels
)

// SyncModels deletes any models not in config and then pulls missing models.
func SyncModels() {
	deleteModelsFunc()
	pullModelsFunc()
}

// ListModels lists models on each configured host, indicating which are currently loaded for Ollama hosts.
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

// OllamaHost.ListRawModels returns the models available on an Ollama host without styling.
func (h *OllamaHost) ListRawModels() ([]string, error) {
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
		models = append(models, model.Name) // ONLY append the raw model name
	}
	return models, nil
}

// ListModels returns the models available on an Ollama host, labeling currently loaded models.
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

// getRunningModels returns the set of currently running models on an Ollama host by querying /api/ps.
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

// ListModelParameters lists the parameters of each model on each host.
func ListModelParameters() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", configFile, err)
		return
	}

	hosts := createHosts(config)
	nodeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	for _, host := range hosts {
		fmt.Println(nodeStyle.Render(fmt.Sprintf("%s:", host.GetName())))
		if host.GetType() != "ollama" {
			fmt.Printf("Listing model parameters is not supported for %s (%s)\n", host.GetName(), host.GetType())
			continue
		}

		params, err := host.GetModelParameters()
		if err != nil {
			fmt.Printf("Error getting model parameters from %s: %v\n", host.GetName(), err)
			continue
		}

		for _, p := range params {
			// Follow the 'list models' structure: model header, then details indented.
			fmt.Printf("  >>> %s\n", p.Model)

			// Extract only the settings of interest from the parameters text.
			settings := extractSettings(p.Parameters)

			fmt.Println("----------------------------------------------------------------")
			pp.Println(params)
			fmt.Println("********************")
			pp.Println(settings)
			fmt.Println("----------------------------------------------------------------")
			fmt.Printf("      temperature: %s\n", settings["temperature"])
			fmt.Printf("      top_p: %s\n", settings["top_p"])
			fmt.Printf("      top_k: %s\n", settings["top_k"])
			fmt.Printf("      repeat_penalty: %s\n", settings["repeat_penalty"])
			fmt.Printf("      min_p: %s\n", settings["min_p"])
		}
		fmt.Println()
	}
}

// GetModelParameters retrieves the parameters for each model on the host.
func (h *OllamaHost) GetModelParameters() ([]ModelParameters, error) {
	// Query tags directly to avoid relying on styled output from ListModels().
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

	var allParams []ModelParameters
	for _, m := range tagsResp.Models {
		params, err := h.getModelParametersFromAPI(m.Name)
		if err != nil {
			return nil, fmt.Errorf("error getting parameters for model %s from %s: %v", m.Name, h.Name, err)
		}
		allParams = append(allParams, params)
	}

	return allParams, nil
}

// getModelParametersFromAPI retrieves the parameters for a single model from the API.
func (h *OllamaHost) getModelParametersFromAPI(model string) (ModelParameters, error) {
	url := fmt.Sprintf("%s/api/show", h.URL)
	payload := map[string]string{"name": model}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return ModelParameters{}, fmt.Errorf("error getting model parameters for %s on %s: %v", model, h.Name, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ModelParameters{}, fmt.Errorf("error reading response body from %s: %v", h.Name, err)
	}

	var params ModelParameters
	if err := json.Unmarshal(respBody, &params); err != nil {
		return ModelParameters{}, fmt.Errorf("error parsing model parameters from %s: %v", h.Name, err)
	}

	params.Model = model

	return params, nil
}

// extractSettings parses the modelfile parameters text and extracts a
// small set of sampling settings of interest. It supports formats like:
//
//	"temperature 0.8"
//	"parameter temperature 0.8"
//	"repeat_penalty=1.1"
func extractSettings(paramsText string) map[string]string {
	wanted := map[string]string{
		"temperature":    "n/a",
		"top_p":          "n/a",
		"top_k":          "n/a",
		"repeat_penalty": "n/a",
		"min_p":          "n/a",
	}

	lines := strings.Split(paramsText, "\n")
	for _, line := range lines {
		s := strings.TrimSpace(strings.ToLower(line))
		if s == "" {
			continue
		}

		// key=value form
		if strings.Contains(s, "=") {
			kv := strings.SplitN(s, "=", 2)
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			if _, ok := wanted[key]; ok && wanted[key] == "n/a" && val != "" {
				wanted[key] = val
			}
			continue
		}

		// space-delimited forms
		fields := strings.Fields(s)
		if len(fields) >= 2 {
			key := fields[0]
			valIdx := 1
			if key == "parameter" && len(fields) >= 3 {
				key = fields[1]
				valIdx = 2
			}
			if _, ok := wanted[key]; ok && wanted[key] == "n/a" {
				wanted[key] = fields[valIdx]
			}
		}
	}

	return wanted
}
