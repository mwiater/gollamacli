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

// Test helper to create a temporary working directory and chdir into it.
func withTempWorkdir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	return dir
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
	withTempWorkdir(t)

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
	cfg := Config{Hosts: []Host{{Name: "o", URL: "u1", Type: "ollama"}, {Name: "l", URL: "u2", Type: "lmstudio"}, {Name: "x", URL: "u3", Type: "unknown"}}}
	hosts := createHosts(cfg)
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].GetType() != "ollama" || hosts[1].GetType() != "lmstudio" {
		t.Fatalf("unexpected types: %s, %s", hosts[0].GetType(), hosts[1].GetType())
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

func Test_ListModels_OllamaAndLMStudio(t *testing.T) {
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

	// LM Studio server
	lm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v0/models" {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "lm1"}, {"id": "lm2"}}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(lm.Close)

	oh := &OllamaHost{Name: "o", URL: ollama.URL}
	gotO, err := oh.ListModels()
	if err != nil {
		t.Fatalf("ollama ListModels err: %v", err)
	}
	joined := strings.Join(gotO, " ")
	if !strings.Contains(joined, "x") || !strings.Contains(joined, "y") {
		t.Fatalf("expected x and y in %v", gotO)
	}

	lh := &LMStudioHost{Name: "l", URL: lm.URL}
	gotL, err := lh.ListModels()
	if err != nil {
		t.Fatalf("lmstudio ListModels err: %v", err)
	}
	if !strings.Contains(strings.Join(gotL, " "), "lm1") {
		t.Fatalf("expected lm1 in %v", gotL)
	}
}

func Test_PullModels_SkipsLMStudioAndCallsOllama(t *testing.T) {
	withTempWorkdir(t)

	var ollamaHits int
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/pull" {
			ollamaHits++
		}
	}))
	t.Cleanup(ollama.Close)

	var lmHits int
	lm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lmHits++
	}))
	t.Cleanup(lm.Close)

	cfg := Config{Hosts: []Host{
		{Name: "o", URL: ollama.URL, Type: "ollama", Models: []string{"m1"}},
		{Name: "l", URL: lm.URL, Type: "lmstudio", Models: []string{"n1"}},
	}}
	writeConfig(t, cfg)

	out := captureOutput(t, PullModels)

	if ollamaHits != 1 {
		t.Fatalf("expected 1 pull hit, got %d", ollamaHits)
	}
	if lmHits != 0 {
		t.Fatalf("expected 0 LM Studio hits, got %d", lmHits)
	}
	if !strings.Contains(out, "Pulling models is not supported for l (lmstudio)") {
		t.Fatalf("missing LM Studio skip message: %q", out)
	}
}

func Test_DeleteModels_SkipsLMStudioAndCallsOllama(t *testing.T) {
	withTempWorkdir(t)

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

	var lmHits int
	lm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lmHits++
	}))
	t.Cleanup(lm.Close)

	cfg := Config{Hosts: []Host{
		{Name: "o", URL: ollama.URL, Type: "ollama", Models: []string{"keep"}},
		{Name: "l", URL: lm.URL, Type: "lmstudio", Models: []string{"x"}},
	}}
	writeConfig(t, cfg)

	out := captureOutput(t, DeleteModels)

	if deleteHits != 1 {
		t.Fatalf("expected 1 delete hit, got %d", deleteHits)
	}
	if lmHits != 0 {
		t.Fatalf("expected 0 LM Studio hits, got %d", lmHits)
	}
	if !strings.Contains(out, "Deleting models is not supported for l (lmstudio)") {
		t.Fatalf("missing LM Studio skip message: %q", out)
	}
}

func Test_UnloadModels_SkipsLMStudioAndCallsOllama(t *testing.T) {
	withTempWorkdir(t)

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

	var lmHits int
	lm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lmHits++
	}))
	t.Cleanup(lm.Close)

	cfg := Config{Hosts: []Host{
		{Name: "o", URL: ollama.URL, Type: "ollama"},
		{Name: "l", URL: lm.URL, Type: "lmstudio"},
	}}
	writeConfig(t, cfg)

	out := captureOutput(t, UnloadModels)

	if unloadHits != 1 {
		t.Fatalf("expected 1 unload hit, got %d", unloadHits)
	}
	if lmHits != 0 {
		t.Fatalf("expected 0 LM Studio hits, got %d", lmHits)
	}
	if !strings.Contains(out, "Unloading models is not supported for l (lmstudio)") {
		t.Fatalf("missing LM Studio skip message: %q", out)
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
	withTempWorkdir(t)
	t.Setenv("NO_COLOR", "1")

	// success Ollama host named 'b'
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ps":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{}})
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{{"name": "m"}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ollama.Close)

	// failing LM Studio host named 'a'
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(bad.Close)

	cfg := Config{Hosts: []Host{
		{Name: "b", URL: ollama.URL, Type: "ollama"},
		{Name: "a", URL: bad.URL, Type: "lmstudio"},
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

func Test_LMStudioHost_DeleteAndUnloadMessages(t *testing.T) {
	h := &LMStudioHost{Name: "l"}

	delMsg := captureOutput(t, func() { h.DeleteModel("m") })
	if !strings.Contains(delMsg, "Deleting models is not supported for LM Studio host: l") {
		t.Fatalf("unexpected delete message: %q", delMsg)
	}

	unloadMsg := captureOutput(t, func() { h.UnloadModel("m") })
	if !strings.Contains(unloadMsg, "Unloading models is not supported for LM Studio host: l") {
		t.Fatalf("unexpected unload message: %q", unloadMsg)
	}
}

func Test_LMStudioHost_ListModels_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{bad"))
	}))
	t.Cleanup(srv.Close)

	h := &LMStudioHost{Name: "l", URL: srv.URL}
	if _, err := h.ListModels(); err == nil || !strings.Contains(err.Error(), "error parsing models") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

// (no helpers)
