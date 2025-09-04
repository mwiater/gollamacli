package models

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

const (
	nodesFile  = "NODES.TXT"
	modelsFile = "MODELS.TXT"
)

// readLines reads a file and returns its lines as a slice of strings.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// PullModels reads models from MODELS.TXT and pulls them to each node in NODES.TXT.
func PullModels() {
	nodes, err := readLines(nodesFile)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", nodesFile, err)
		return
	}

	models, err := readLines(modelsFile)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", modelsFile, err)
		return
	}

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			fmt.Printf("Starting model pulls for %s...\n", node)
			for _, model := range models {
				fmt.Printf("  -> Pulling model: %s on %s\n", model, node)
				pullModel(node, model)
			}
		}(node)
	}
	wg.Wait()
	fmt.Println("All model pull commands have finished.")
}

func pullModel(node, model string) {
	url := fmt.Sprintf("https://%s/api/pull", node)
	payload := map[string]string{"name": model}
	body, _ := json.Marshal(payload)
	_, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Error pulling model %s on %s: %v\n", model, node, err)
	}
}

// DeleteModels reads MODELS.TXT and deletes any models not on the list from each node in NODES.TXT.
func DeleteModels() {
	nodes, err := readLines(nodesFile)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", nodesFile, err)
		return
	}

	modelsToKeep, err := readLines(modelsFile)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", modelsFile, err)
		return
	}

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			deleteModelsOnNode(node, modelsToKeep)
		}(node)
	}
	wg.Wait()
	fmt.Println("All model cleanup commands have finished.")
}

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
