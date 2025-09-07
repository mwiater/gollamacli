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
	// configFile is the name of the configuration file.
	configFile = "config.json"
)

// Config represents the application's configuration.
type Config struct {
	// Nodes is a list of Ollama nodes.
	Nodes  []string `json:"nodes"`
	// Models is a list of models to manage.
	Models []string `json:"models"`
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

// PullModels reads models from config.json and pulls them to each node.
func PullModels() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", configFile, err)
		return
	}

	var wg sync.WaitGroup
	for _, node := range config.Nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			fmt.Printf("Starting model pulls for %s...\n", node)
			for _, model := range config.Models {
				fmt.Printf("  -> Pulling model: %s on %s\n", model, node)
				pullModel(node, model)
			}
		}(node)
	}
	wg.Wait()
	fmt.Println("All model pull commands have finished.")
}

// pullModel pulls a model to a single node.
func pullModel(node, model string) {
	url := fmt.Sprintf("https://%s/api/pull", node)
	payload := map[string]string{"name": model}
	body, _ := json.Marshal(payload)
	_, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Error pulling model %s on %s: %v\n", model, node, err)
	}
}

// DeleteModels reads config.json and deletes any models not on the list from each node.
func DeleteModels() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", configFile, err)
		return
	}

	var wg sync.WaitGroup
	for _, node := range config.Nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			deleteModelsOnNode(node, config.Models)
		}(node)
	}
	wg.Wait()
	fmt.Println("All model cleanup commands have finished.")
}

// deleteModelsOnNode deletes any models not on the list from a single node.
func deleteModelsOnNode(node string, modelsToKeep []string) {
	fmt.Printf("Starting model cleanup for %s...\n", node)
	url := fmt.Sprintf("https://%s/api/tags", node)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error getting models from %s: %v\n", node, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body from %s: %v\n", node, err)
		return
	}

	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		fmt.Printf("Error parsing models from %s: %v\n", node, err)
		return
	}

	modelsToKeepSet := make(map[string]struct{})
	for _, m := range modelsToKeep {
		modelsToKeepSet[m] = struct{}{}
	}

	for _, installedModel := range tagsResp.Models {
		if _, keep := modelsToKeepSet[installedModel.Name]; !keep {
			fmt.Printf("  -> Deleting model: %s on %s\n", installedModel.Name, node)
			deleteModel(node, installedModel.Name)
		} else {
			fmt.Printf("  -> Keeping model: %s on %s\n", installedModel.Name, node)
		}
	}
}

// deleteModel deletes a model from a single node.
func deleteModel(node, model string) {
	url := fmt.Sprintf("https://%s/api/delete", node)
	payload := map[string]string{"model": model}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("DELETE", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error deleting model %s on %s: %v\n", model, node, err)
	}
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

	nodeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	nodeModels := make(map[string][]string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, node := range config.Nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			models, err := listModelsOnNode(node)
			mu.Lock()
			if err != nil {
				nodeModels[node] = []string{fmt.Sprintf("Error: %v", err)}
			} else {
				nodeModels[node] = models
			}
			mu.Unlock()
		}(node)
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
			fmt.Println("  >>>, " + cleanedModelString)
		}
		fmt.Println()
	}
}

// listModelsOnNode gets the models on a single node.
func listModelsOnNode(node string) ([]string, error) {
	modelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	loadedModelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))

	runningModels, err := getRunningModels(node)
	if err != nil {
		return nil, fmt.Errorf("could not get running models: %v", err)
	}

	url := fmt.Sprintf("https://%s/api/tags", node)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not list models: Ollama is not accessible on %s", node)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from %s: %v", node, err)
	}

	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil, fmt.Errorf("error parsing models from %s: %v", node, err)
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

// getRunningModels gets the running models on a single node.
func getRunningModels(node string) (map[string]struct{}, error) {
	runningModels := make(map[string]struct{})
	url := fmt.Sprintf("https://%s/api/ps", node)
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