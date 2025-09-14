package models

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helper to create and chdir into a temporary working directory.
// It returns the directory path and a cleanup function to chdir back.
func withTempWorkdir(t *testing.T) (string, func()) {
	t.Helper()

	// 1. Get the original working directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	// 2. Create the temp directory
	dir := t.TempDir()

	// 3. Change into the new directory
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}

	// 4. Return the path and a function that restores the original directory
	cleanup := func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("chdir back: %v", err)
		}
	}

	return dir, cleanup
}

func writeConfig(t *testing.T, cfg Config) {
	t.Helper()
	b, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(".", "config.json"), b, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// captureOutput runs f while capturing stdout output.
func captureOutput(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()
	f()
	w.Close()
	os.Stdout = old
	return <-outC
}

func Test_loadConfig_SuccessAndMissing(t *testing.T) {
	_, cleanup := withTempWorkdir(t)
	defer cleanup()

	// Missing file
	if _, err := loadConfig(); err == nil {
		t.Fatalf("expected error for missing config.json")
	}

	// Success
	cfg := Config{Hosts: []Host{{Name: "h", URL: "http://example", Type: "ollama", Models: []string{"m1"}}}}
	writeConfig(t, cfg)
	got, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	if len(got.Hosts) != 1 || got.Hosts[0].Name != "h" {
		t.Fatalf("unexpected cfg: %+v", got)
	}
}

func Test_createHosts_BuildsTypesAndIgnoresUnknown(t *testing.T) {
	cfg := Config{Hosts: []Host{{Name: "o", URL: "u1", Type: "ollama"}, {Name: "x", URL: "u2", Type: "unknown"}}}
	hosts := createHosts(cfg)
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].GetType() != "ollama" {
		t.Fatalf("unexpected type: %s", hosts[0].GetType())
	}
}

func Test_OllamaHost_PullModel(t *testing.T) {
	var gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = b
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	h := &OllamaHost{Name: "h", URL: srv.URL, Models: nil}
	h.PullModel("mod1")
	if gotPath != "/api/pull" {
		t.Fatalf("expected /api/pull, got %s", gotPath)
	}
	if !bytes.Contains(gotBody, []byte(`"name":"mod1"`)) {
		t.Fatalf("unexpected body: %s", string(gotBody))
	}
}

func Test_deleteModelsOnNode_DeletesExtras(t *testing.T) {
	sh := &stubHost{}
	deleteModelsOnNode(sh, sh.GetModels())
	got := strings.Join(sh.deleted, ",")
	if !strings.Contains(got, "del1") || !strings.Contains(got, "del2") {
		t.Fatalf("expected deletions of del1,del2; got %s", got)
	}
}

// stubHost implements LLMHost for unit tests of deleteModelsOnNode.
type stubHost struct{ deleted []string }

func (s *stubHost) PullModel(string)     {}
func (s *stubHost) DeleteModel(m string) { s.deleted = append(s.deleted, m) }
func (s *stubHost) ListModels() ([]string, error) {
	return []string{"- keep", "- del1", "- del2 (CURRENTLY LOADED)"}, nil
}
func (s *stubHost) UnloadModel(string)  {}
func (s *stubHost) GetName() string     { return "stub" }
func (s *stubHost) GetType() string     { return "ollama" }
func (s *stubHost) GetModels() []string { return []string{"keep"} }

func Test_OllamaHost_DeleteModel(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = b
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	h := &OllamaHost{Name: "h", URL: srv.URL}
	h.DeleteModel("mx")
	if gotMethod != http.MethodDelete || gotPath != "/api/delete" {
		t.Fatalf("unexpected %s %s", gotMethod, gotPath)
	}
	if !bytes.Contains(gotBody, []byte(`"model":"mx"`)) {
		t.Fatalf("unexpected body: %s", string(gotBody))
	}
}

func Test_OllamaHost_UnloadModel(t *testing.T) {
	var gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = b
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	h := &OllamaHost{Name: "h", URL: srv.URL}
	h.UnloadModel("mm")
	if gotPath != "/api/chat" {
		t.Fatalf("expected /api/chat, got %s", gotPath)
	}
	if !bytes.Contains(gotBody, []byte(`"keep_alive":0`)) || !bytes.Contains(gotBody, []byte(`"model":"mm"`)) {
		t.Fatalf("unexpected body: %s", string(gotBody))
	}
}

func Test_getRunningModels_ParsesNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ps" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{{"name": "a"}, {"name": "b"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	h := &OllamaHost{Name: "h", URL: srv.URL}
	got, err := h.getRunningModels()
	if err != nil {
		t.Fatalf("getRunningModels error: %v", err)
	}
	if _, ok := got["a"]; !ok {
		t.Fatalf("expected 'a' present")
	}
	if _, ok := got["b"]; !ok {
		t.Fatalf("expected 'b' present")
	}
}

func Test_OllamaHost_ListModels(t *testing.T) {
	// Ollama server
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ps":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{{"name": "x"}}})
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{{"name": "x"}, {"name": "y"}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ollama.Close)

	oh := &OllamaHost{Name: "o", URL: ollama.URL}
	gotO, err := oh.ListModels()
	if err != nil {
		t.Fatalf("ollama ListModels err: %v", err)
	}
	joined := strings.Join(gotO, " ")
	if !strings.Contains(joined, "x") || !strings.Contains(joined, "y") {
		t.Fatalf("expected x and y in %v", gotO)
	}
}

func Test_PullModels_CallsOllama(t *testing.T) {
	_, cleanup := withTempWorkdir(t)
	defer cleanup()

	var ollamaHits int
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/pull" {
			ollamaHits++
		}
	}))
	t.Cleanup(ollama.Close)

	cfg := Config{Hosts: []Host{
		{Name: "o", URL: ollama.URL, Type: "ollama", Models: []string{"m1"}},
	}}
	writeConfig(t, cfg)

	_ = captureOutput(t, PullModels)

	if ollamaHits != 1 {
		t.Fatalf("expected 1 pull hit, got %d", ollamaHits)
	}
}

func Test_DeleteModels_CallsOllama(t *testing.T) {
	_, cleanup := withTempWorkdir(t)
	defer cleanup()

	var deleteHits int
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ps":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{{"name": "keep"}}})
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{{"name": "keep"}, {"name": "del"}}})
		case "/api/delete":
			deleteHits++
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ollama.Close)

	cfg := Config{Hosts: []Host{
		{Name: "o", URL: ollama.URL, Type: "ollama", Models: []string{"keep"}},
	}}
	writeConfig(t, cfg)

	_ = captureOutput(t, DeleteModels)

	if deleteHits != 1 {
		t.Fatalf("expected 1 delete hit, got %d", deleteHits)
	}
}

func Test_UnloadModels_CallsOllama(t *testing.T) {
	_, cleanup := withTempWorkdir(t)
	defer cleanup()

	var unloadHits int
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ps":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{{"name": "m1"}}})
		case "/api/chat":
			unloadHits++
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ollama.Close)

	cfg := Config{Hosts: []Host{
		{Name: "o", URL: ollama.URL, Type: "ollama"},
	}}
	writeConfig(t, cfg)

	_ = captureOutput(t, UnloadModels)

	if unloadHits != 1 {
		t.Fatalf("expected 1 unload hit, got %d", unloadHits)
	}
}

func Test_SyncModels_CallsInOrder(t *testing.T) {
	var calls []string
	oldDel, oldPull := deleteModelsFunc, pullModelsFunc
	deleteModelsFunc = func() { calls = append(calls, "del") }
	pullModelsFunc = func() { calls = append(calls, "pull") }
	t.Cleanup(func() { deleteModelsFunc = oldDel; pullModelsFunc = oldPull })

	SyncModels()

	if strings.Join(calls, ",") != "del,pull" {
		t.Fatalf("unexpected call order: %v", calls)
	}
}

func Test_ListModels_TopLevel_SortsAndAggregates(t *testing.T) {
	_, cleanup := withTempWorkdir(t)
	defer cleanup()

	t.Setenv("NO_COLOR", "1")

	// success Ollama host named 'b'
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ps":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{}})
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{{"name": "m"}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(good.Close)

	// failing Ollama host named 'a'
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(bad.Close)

	cfg := Config{Hosts: []Host{
		{Name: "b", URL: good.URL, Type: "ollama"},
		{Name: "a", URL: bad.URL, Type: "ollama"},
	}}
	writeConfig(t, cfg)

	out := captureOutput(t, ListModels)

	idxA := strings.Index(out, "a:")
	idxB := strings.Index(out, "b:")
	if idxA == -1 || idxB == -1 {
		t.Fatalf("missing host output: %q", out)
	}
	if idxB < idxA {
		t.Fatalf("expected host 'a' before 'b': %q", out)
	}
	if !strings.Contains(out, "Error:") {
		t.Fatalf("expected error message for host a: %q", out)
	}
}

// (no helpers)
